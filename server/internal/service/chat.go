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
