package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/ekhrunov/messenger/server/internal/domain"
	"github.com/ekhrunov/messenger/server/internal/e2ee"
	"github.com/ekhrunov/messenger/server/internal/repository"
)

var ErrIdentityKeyNotFound = errors.New("identity key not found")
var ErrE2EEAccessDenied = errors.New("e2ee access denied")

// ErrKeyVersionExists is returned when a new chat key version reuses an existing key id.
var ErrKeyVersionExists = repository.ErrKeyVersionExists

// ErrAlreadyMember is returned when a user added to a chat is already a member.
var ErrAlreadyMember = repository.ErrAlreadyMember

type E2EEService struct {
	repo             repository.E2EERepository
	chatRepo         repository.ChatRepository
	userRepo         repository.UserRepository
	oidcProviderRepo repository.OIDCProviderRepository
}

func NewE2EEService(
	repo repository.E2EERepository,
	chatRepo repository.ChatRepository,
	userRepo repository.UserRepository,
	oidcProviderRepo repository.OIDCProviderRepository,
) *E2EEService {
	return &E2EEService{
		repo:             repo,
		chatRepo:         chatRepo,
		userRepo:         userRepo,
		oidcProviderRepo: oidcProviderRepo,
	}
}

func (s *E2EEService) PutIdentityKey(ctx context.Context, userID string, publicKey json.RawMessage, tokenUser TokenUser) error {
	if len(publicKey) == 0 {
		return fmt.Errorf("publicKey is required")
	}
	if len(publicKey) > e2ee.MaxIdentityKeyJSONLen {
		return fmt.Errorf("publicKey is too large")
	}

	if err := requireCurrentUserMatches(ctx, tokenUser, userID, s.userRepo, s.oidcProviderRepo, ErrE2EEAccessDenied); err != nil {
		return err
	}

	var parsed domain.IdentityPublicKey
	if err := json.Unmarshal(publicKey, &parsed); err != nil {
		return fmt.Errorf("invalid publicKey: %w", err)
	}
	if parsed.V != 1 || parsed.Alg != "hybrid-kem-mlkem768-x25519" {
		return fmt.Errorf("unsupported identity key version or algorithm")
	}
	if strings.TrimSpace(parsed.PublicKey) == "" {
		return fmt.Errorf("identity key fields are required")
	}
	if err := e2ee.ValidateIdentityPublicKeyBytes(parsed.PublicKey); err != nil {
		return err
	}

	return s.repo.UpsertIdentityKey(ctx, userID, "default", publicKey)
}

func (s *E2EEService) GetIdentityKey(ctx context.Context, userID string) (domain.UserIdentityKey, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return domain.UserIdentityKey{}, fmt.Errorf("user_id is required")
	}

	item, err := s.repo.GetIdentityKey(ctx, userID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return domain.UserIdentityKey{}, ErrIdentityKeyNotFound
		}
		return domain.UserIdentityKey{}, err
	}

	return item, nil
}

type ChatKeyWrapInput struct {
	UserID string
	Wrap   json.RawMessage
}

type CreateE2EEChatInput struct {
	Name                  string
	UsersUIDs             []string
	KeyID                 string
	Wraps                 []ChatKeyWrapInput
	DisappearAfterMinutes int
}

func (s *E2EEService) CreateChat(ctx context.Context, input CreateE2EEChatInput, tokenUser TokenUser) (domain.Chat, error) {
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return domain.Chat{}, fmt.Errorf("name is required")
	}

	keyID := strings.TrimSpace(input.KeyID)
	if keyID == "" {
		return domain.Chat{}, fmt.Errorf("e2ee.key_id is required")
	}

	userIDs := make([]string, 0, len(input.UsersUIDs))
	for _, userID := range input.UsersUIDs {
		userID = strings.TrimSpace(userID)
		if userID != "" {
			userIDs = append(userIDs, userID)
		}
	}
	if len(userIDs) == 0 {
		return domain.Chat{}, fmt.Errorf("users_uids is required")
	}

	currentUserID, err := ResolveCurrentUserID(ctx, tokenUser, s.userRepo, s.oidcProviderRepo)
	if err != nil {
		return domain.Chat{}, err
	}
	if !slices.Contains(userIDs, currentUserID) {
		return domain.Chat{}, ErrChatAccessDenied
	}

	if len(input.Wraps) != len(userIDs) {
		return domain.Chat{}, fmt.Errorf("e2ee.wraps must include every chat member")
	}

	wrapsByUser := make(map[string]json.RawMessage, len(input.Wraps))
	for _, item := range input.Wraps {
		userID := strings.TrimSpace(item.UserID)
		if userID == "" {
			return domain.Chat{}, fmt.Errorf("e2ee.wrap user_id is required")
		}
		if !slices.Contains(userIDs, userID) {
			return domain.Chat{}, fmt.Errorf("e2ee.wrap user is not a chat member")
		}
		if len(item.Wrap) == 0 {
			return domain.Chat{}, fmt.Errorf("e2ee.wrap is required")
		}
		if len(item.Wrap) > e2ee.MaxWrapJSONLen {
			return domain.Chat{}, fmt.Errorf("e2ee.wrap is too large")
		}
		if _, exists := wrapsByUser[userID]; exists {
			return domain.Chat{}, fmt.Errorf("duplicate e2ee.wrap for user")
		}
		wrapsByUser[userID] = item.Wrap
	}

	for _, userID := range userIDs {
		if _, ok := wrapsByUser[userID]; !ok {
			return domain.Chat{}, fmt.Errorf("missing e2ee.wrap for user %s", userID)
		}
	}

	disappearAfterMinutes := input.DisappearAfterMinutes
	if disappearAfterMinutes == 0 {
		disappearAfterMinutes = domain.DefaultDisappearAfterMinutes
	}
	if disappearAfterMinutes < 1 {
		return domain.Chat{}, fmt.Errorf("disappear_after_minutes must be positive")
	}

	return s.repo.CreateChatWithKeys(ctx, name, userIDs, keyID, currentUserID, wrapsByUser, disappearAfterMinutes)
}

func (s *E2EEService) ListKeyWraps(ctx context.Context, chatID string, tokenUser TokenUser) ([]domain.UserChatKeyWrap, error) {
	chatID = strings.TrimSpace(chatID)
	if chatID == "" {
		return nil, fmt.Errorf("chat_id is required")
	}

	currentUserID, err := ResolveCurrentUserID(ctx, tokenUser, s.userRepo, s.oidcProviderRepo)
	if err != nil {
		return nil, err
	}

	if err := requireUserBelongsToChat(ctx, s.chatRepo, chatID, currentUserID, ErrE2EEAccessDenied); err != nil {
		return nil, err
	}

	return s.repo.ListKeyWrapsForUser(ctx, chatID, currentUserID)
}

// AddKeyVersion adds a new chat key version with one wrap per current chat member. Existing versions and wraps are
// kept, so members holding older wraps keep reading older messages. The caller must be a chat member.
func (s *E2EEService) AddKeyVersion(ctx context.Context, chatID, keyID string, wraps []ChatKeyWrapInput, tokenUser TokenUser) error {
	chatID = strings.TrimSpace(chatID)
	if chatID == "" {
		return fmt.Errorf("chat_id is required")
	}
	keyID = strings.TrimSpace(keyID)
	if keyID == "" {
		return fmt.Errorf("key_id is required")
	}

	currentUserID, err := ResolveCurrentUserID(ctx, tokenUser, s.userRepo, s.oidcProviderRepo)
	if err != nil {
		return err
	}
	if err := requireUserBelongsToChat(ctx, s.chatRepo, chatID, currentUserID, ErrE2EEAccessDenied); err != nil {
		return err
	}

	members, err := s.chatRepo.ListMembers(ctx, chatID)
	if err != nil {
		return err
	}
	memberIDs := make([]string, 0, len(members))
	for _, member := range members {
		memberIDs = append(memberIDs, member.ID)
	}

	wrapsByUser, err := collectKeyWraps(wraps, memberIDs)
	if err != nil {
		return err
	}

	return s.repo.AddKeyVersion(ctx, chatID, keyID, currentUserID, wrapsByUser)
}

// collectKeyWraps validates one wrap per expected user (no extras, duplicates or oversized wraps) and indexes them
// by user id.
func collectKeyWraps(wraps []ChatKeyWrapInput, expectedUserIDs []string) (map[string]json.RawMessage, error) {
	if len(wraps) != len(expectedUserIDs) {
		return nil, fmt.Errorf("wraps must include every chat member")
	}

	wrapsByUser := make(map[string]json.RawMessage, len(wraps))
	for _, item := range wraps {
		userID := strings.TrimSpace(item.UserID)
		if userID == "" {
			return nil, fmt.Errorf("wrap user_id is required")
		}
		if !slices.Contains(expectedUserIDs, userID) {
			return nil, fmt.Errorf("wrap user is not a chat member")
		}
		if len(item.Wrap) == 0 {
			return nil, fmt.Errorf("wrap is required")
		}
		if len(item.Wrap) > e2ee.MaxWrapJSONLen {
			return nil, fmt.Errorf("wrap is too large")
		}
		if _, exists := wrapsByUser[userID]; exists {
			return nil, fmt.Errorf("duplicate wrap for user")
		}
		wrapsByUser[userID] = item.Wrap
	}
	return wrapsByUser, nil
}

// AlreadyMembersError lists the requested users that already belong to the chat.
type AlreadyMembersError struct{ UserIDs []string }

func (e *AlreadyMembersError) Error() string { return "users are already chat members" }

// MissingIdentityKeysError lists the requested users that do not exist or have no identity key registered.
type MissingIdentityKeysError struct{ UserIDs []string }

func (e *MissingIdentityKeysError) Error() string { return "users have no identity key registered" }

// AddMembers adds users to an existing chat together with one new chat key version that is wrapped for every
// current member and every new user. New users get a wrap only for this version, so they cannot read earlier
// messages. The caller must be a chat admin. The whole batch succeeds or nothing is added.
func (s *E2EEService) AddMembers(ctx context.Context, chatID string, userIDs []string, keyID string, wraps []ChatKeyWrapInput, tokenUser TokenUser) error {
	chatID = strings.TrimSpace(chatID)
	if chatID == "" {
		return fmt.Errorf("chat_id is required")
	}
	keyID = strings.TrimSpace(keyID)
	if keyID == "" {
		return fmt.Errorf("key_id is required")
	}

	newIDs := make([]string, 0, len(userIDs))
	for _, userID := range userIDs {
		userID = strings.TrimSpace(userID)
		if userID == "" {
			return fmt.Errorf("user_ids entries are required")
		}
		if slices.Contains(newIDs, userID) {
			return fmt.Errorf("duplicate user in user_ids")
		}
		newIDs = append(newIDs, userID)
	}
	if len(newIDs) == 0 {
		return fmt.Errorf("user_ids is required")
	}
	if len(newIDs) > e2ee.MaxAddMembersBatch {
		return fmt.Errorf("user_ids has too many entries (max %d)", e2ee.MaxAddMembersBatch)
	}

	currentUserID, err := ResolveCurrentUserID(ctx, tokenUser, s.userRepo, s.oidcProviderRepo)
	if err != nil {
		return err
	}
	if err := requireUserBelongsToChat(ctx, s.chatRepo, chatID, currentUserID, ErrE2EEAccessDenied); err != nil {
		return err
	}
	isAdmin, err := s.chatRepo.IsAdmin(ctx, chatID, currentUserID)
	if err != nil {
		return err
	}
	if !isAdmin {
		return ErrChatNotAdmin
	}

	members, err := s.chatRepo.ListMembers(ctx, chatID)
	if err != nil {
		return err
	}
	allIDs := make([]string, 0, len(members)+len(newIDs))
	var already []string
	for _, member := range members {
		allIDs = append(allIDs, member.ID)
		if slices.Contains(newIDs, member.ID) {
			already = append(already, member.ID)
		}
	}
	if len(already) > 0 {
		return &AlreadyMembersError{UserIDs: already}
	}

	var missing []string
	for _, userID := range newIDs {
		if _, err := s.GetIdentityKey(ctx, userID); err != nil {
			if errors.Is(err, ErrIdentityKeyNotFound) {
				missing = append(missing, userID)
				continue
			}
			return err
		}
	}
	if len(missing) > 0 {
		return &MissingIdentityKeysError{UserIDs: missing}
	}

	allIDs = append(allIDs, newIDs...)
	wrapsByUser, err := collectKeyWraps(wraps, allIDs)
	if err != nil {
		return err
	}

	return s.repo.AddMembersWithKeyVersion(ctx, chatID, newIDs, keyID, currentUserID, wrapsByUser)
}
