package service

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/ekhrunov/messenger/server/internal/domain"
	"github.com/ekhrunov/messenger/server/internal/repository"
)

var ErrChatAccessDenied = errors.New("chat access denied")

// ErrChatSelfRemoval is returned when an admin tries to remove themselves via the member-removal endpoint.
var ErrChatSelfRemoval = errors.New("cannot remove yourself from the chat")

type ChatService struct {
	repo             repository.ChatRepository
	userRepo         repository.UserRepository
	oidcProviderRepo repository.OIDCProviderRepository
}

func NewChatService(
	repo repository.ChatRepository,
	userRepo repository.UserRepository,
	oidcProviderRepo repository.OIDCProviderRepository,
) *ChatService {
	return &ChatService{
		repo:             repo,
		userRepo:         userRepo,
		oidcProviderRepo: oidcProviderRepo,
	}
}

// ErrChatNotAdmin is returned when a non-admin attempts an admin-only chat action.
var ErrChatNotAdmin = errors.New("chat admin required")

// ErrChatIDRequired and ErrChatUserIDRequired are returned when a member operation is called without an ID.
var (
	ErrChatIDRequired     = errors.New("chat_id is required")
	ErrChatUserIDRequired = errors.New("user_id is required")
)

// ListMembers returns the members of a chat; the caller must be a member.
func (s *ChatService) ListMembers(ctx context.Context, chatID string, tokenUser TokenUser) ([]domain.ChatMember, error) {
	chatID = strings.TrimSpace(chatID)
	if chatID == "" {
		return nil, ErrChatIDRequired
	}

	currentUserID, err := ResolveCurrentUserID(ctx, tokenUser, s.userRepo, s.oidcProviderRepo)
	if err != nil {
		return nil, err
	}
	if err := requireUserBelongsToChat(ctx, s.repo, chatID, currentUserID, ErrChatAccessDenied); err != nil {
		return nil, err
	}

	return s.repo.ListMembers(ctx, chatID)
}

// RemoveMember removes another member from a chat. The caller must be a chat admin and cannot remove themselves
// (self-leave is a separate flow).
func (s *ChatService) RemoveMember(ctx context.Context, chatID, userID string, tokenUser TokenUser) error {
	chatID = strings.TrimSpace(chatID)
	userID = strings.TrimSpace(userID)
	if chatID == "" {
		return ErrChatIDRequired
	}
	if userID == "" {
		return ErrChatUserIDRequired
	}

	currentUserID, err := ResolveCurrentUserID(ctx, tokenUser, s.userRepo, s.oidcProviderRepo)
	if err != nil {
		return err
	}
	if err := requireUserBelongsToChat(ctx, s.repo, chatID, currentUserID, ErrChatAccessDenied); err != nil {
		return err
	}

	isAdmin, err := s.repo.IsAdmin(ctx, chatID, currentUserID)
	if err != nil {
		return err
	}
	if !isAdmin {
		return ErrChatNotAdmin
	}
	if currentUserID == userID {
		return ErrChatSelfRemoval
	}

	return s.repo.RemoveMember(ctx, chatID, userID)
}

func (s *ChatService) List(ctx context.Context, filter repository.ChatFilter, tokenUser TokenUser) ([]domain.Chat, error) {
	if filter.UserID == "" {
		return nil, fmt.Errorf("user_id is required")
	}

	if err := requireCurrentUserMatches(ctx, tokenUser, filter.UserID, s.userRepo, s.oidcProviderRepo, ErrChatAccessDenied); err != nil {
		return nil, err
	}

	return s.repo.List(ctx, filter)
}

type CreateChatInput struct {
	Name                  string
	UsersUIDs             []string
	DisappearAfterMinutes int
}

func (s *ChatService) Create(ctx context.Context, input CreateChatInput, tokenUser TokenUser) (domain.Chat, error) {
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return domain.Chat{}, fmt.Errorf("name is required")
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

	disappearAfterMinutes := input.DisappearAfterMinutes
	if disappearAfterMinutes == 0 {
		disappearAfterMinutes = domain.DefaultDisappearAfterMinutes
	}
	if disappearAfterMinutes < 1 {
		return domain.Chat{}, fmt.Errorf("disappear_after_minutes must be positive")
	}

	currentUserID, err := ResolveCurrentUserID(ctx, tokenUser, s.userRepo, s.oidcProviderRepo)
	if err != nil {
		return domain.Chat{}, err
	}
	if !slices.Contains(userIDs, currentUserID) {
		return domain.Chat{}, ErrChatAccessDenied
	}

	return s.repo.Create(ctx, name, userIDs, disappearAfterMinutes, currentUserID)
}
