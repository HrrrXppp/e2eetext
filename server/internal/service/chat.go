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

type ChatMemberKeyInput struct {
	UserID                string
	WrappedChatPrivateKey []byte
	KemCiphertext         []byte
}

type CreateChatInput struct {
	Name         string
	KemPublicKey []byte
	Members      []ChatMemberKeyInput
}

func (s *ChatService) Create(ctx context.Context, input CreateChatInput, tokenUser TokenUser) (domain.Chat, error) {
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return domain.Chat{}, fmt.Errorf("name is required")
	}
	if len(input.KemPublicKey) == 0 {
		return domain.Chat{}, fmt.Errorf("kem_public_key is required")
	}

	seen := make(map[string]struct{}, len(input.Members))
	members := make([]repository.ChatMemberKey, 0, len(input.Members))
	userIDs := make([]string, 0, len(input.Members))
	for _, member := range input.Members {
		userID := strings.TrimSpace(member.UserID)
		if userID == "" {
			continue
		}
		if _, ok := seen[userID]; ok {
			continue
		}
		seen[userID] = struct{}{}

		if len(member.WrappedChatPrivateKey) == 0 {
			return domain.Chat{}, fmt.Errorf("wrapped_chat_private_key is required for user %s", userID)
		}
		if len(member.KemCiphertext) == 0 {
			return domain.Chat{}, fmt.Errorf("kem_ciphertext is required for user %s", userID)
		}

		userIDs = append(userIDs, userID)
		members = append(members, repository.ChatMemberKey{
			UserID:                userID,
			WrappedChatPrivateKey: member.WrappedChatPrivateKey,
			KemCiphertext:         member.KemCiphertext,
		})
	}
	if len(members) == 0 {
		return domain.Chat{}, fmt.Errorf("members is required")
	}

	currentUserID, err := ResolveCurrentUserID(ctx, tokenUser, s.userRepo, s.oidcProviderRepo)
	if err != nil {
		return domain.Chat{}, err
	}
	if !slices.Contains(userIDs, currentUserID) {
		return domain.Chat{}, ErrChatAccessDenied
	}

	return s.repo.Create(ctx, name, currentUserID, input.KemPublicKey, members)
}
