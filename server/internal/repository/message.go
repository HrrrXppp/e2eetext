package repository

import (
	"context"
	"errors"

	"github.com/ekhrunov/messenger/server/internal/domain"
)

var ErrMessageNotFound = errors.New("message not found")

type MessageFilter struct {
	ChatID string
	UserID string
}

type MessageRepository interface {
	List(ctx context.Context, filter MessageFilter) ([]domain.Message, error)
	Create(ctx context.Context, message domain.Message) (domain.Message, error)
	GetByID(ctx context.Context, id string) (domain.Message, error)
	MarkRead(ctx context.Context, messageID, userID string) error
}
