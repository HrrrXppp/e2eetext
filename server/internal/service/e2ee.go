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
	Name      string
	UsersUIDs []string
	KeyID     string
	Wraps     []ChatKeyWrapInput
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

	return s.repo.CreateChatWithKeys(ctx, name, userIDs, keyID, currentUserID, wrapsByUser)
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
