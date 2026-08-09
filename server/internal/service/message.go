package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/ekhrunov/messenger/server/internal/domain"
	"github.com/ekhrunov/messenger/server/internal/e2ee"
	"github.com/ekhrunov/messenger/server/internal/repository"
)

var ErrMessageAccessDenied = errors.New("message access denied")
var ErrMessageNotFound = errors.New("message not found")

type MessageService struct {
	repo             repository.MessageRepository
	chatRepo         repository.ChatRepository
	userRepo         repository.UserRepository
	oidcProviderRepo repository.OIDCProviderRepository
}

func NewMessageService(
	repo repository.MessageRepository,
	chatRepo repository.ChatRepository,
	userRepo repository.UserRepository,
	oidcProviderRepo repository.OIDCProviderRepository,
) *MessageService {
	return &MessageService{
		repo:             repo,
		chatRepo:         chatRepo,
		userRepo:         userRepo,
		oidcProviderRepo: oidcProviderRepo,
	}
}

func (s *MessageService) List(ctx context.Context, filter repository.MessageFilter, tokenUser TokenUser) ([]domain.Message, error) {
	if filter.ChatID == "" {
		return nil, fmt.Errorf("chat_id is required")
	}

	currentUserID, err := ResolveCurrentUserID(ctx, tokenUser, s.userRepo, s.oidcProviderRepo)
	if err != nil {
		return nil, err
	}

	if err := requireUserBelongsToChat(ctx, s.chatRepo, filter.ChatID, currentUserID, ErrMessageAccessDenied); err != nil {
		return nil, err
	}

	return s.repo.List(ctx, repository.MessageFilter{
		ChatID: filter.ChatID,
		UserID: currentUserID,
	})
}

type CreateMessageInput struct {
	ChatID string
	UserID string
	Data   string
}

func (s *MessageService) Create(ctx context.Context, input CreateMessageInput, tokenUser TokenUser) (domain.Message, error) {
	if input.ChatID == "" {
		return domain.Message{}, fmt.Errorf("chat_id is required")
	}
	if input.UserID == "" {
		return domain.Message{}, fmt.Errorf("user_id is required")
	}
	if input.Data == "" {
		return domain.Message{}, fmt.Errorf("data is required")
	}
	if len(input.Data) > e2ee.MaxMessageDataLen {
		return domain.Message{}, fmt.Errorf("data exceeds maximum length")
	}

	if err := requireCurrentUserMatches(ctx, tokenUser, input.UserID, s.userRepo, s.oidcProviderRepo, ErrMessageAccessDenied); err != nil {
		return domain.Message{}, err
	}

	if err := requireUserBelongsToChat(ctx, s.chatRepo, input.ChatID, input.UserID, ErrMessageAccessDenied); err != nil {
		return domain.Message{}, err
	}

	created, err := s.repo.Create(ctx, domain.Message{
		ChatID: input.ChatID,
		UserID: input.UserID,
		Data:   input.Data,
	})
	if err != nil {
		return domain.Message{}, err
	}

	if user, err := s.userRepo.GetByID(ctx, input.UserID); err == nil {
		created.UserName = user.Name
	}

	return created, nil
}

func (s *MessageService) MarkRead(ctx context.Context, messageID string, tokenUser TokenUser) (domain.Message, error) {
	if messageID == "" {
		return domain.Message{}, fmt.Errorf("message id is required")
	}

	currentUserID, err := ResolveCurrentUserID(ctx, tokenUser, s.userRepo, s.oidcProviderRepo)
	if err != nil {
		return domain.Message{}, err
	}

	message, err := s.repo.GetByID(ctx, messageID)
	if err != nil {
		if errors.Is(err, repository.ErrMessageNotFound) {
			return domain.Message{}, ErrMessageNotFound
		}
		return domain.Message{}, err
	}

	if err := requireUserBelongsToChat(ctx, s.chatRepo, message.ChatID, currentUserID, ErrMessageAccessDenied); err != nil {
		return domain.Message{}, err
	}

	if err := s.repo.MarkRead(ctx, messageID, currentUserID); err != nil {
		return domain.Message{}, err
	}

	message.Unread = false
	return message, nil
}
