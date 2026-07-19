package repository

import (
	"context"
	"time"

	"github.com/ekhrunov/messenger/server/internal/domain"
)

type ChatFilter struct {
	UserID string
}

type ChatUnreadCount struct {
	UserID string
	Count  int
}

type UserChatUnread struct {
	ChatID    string
	Count     int
	UpdatedAt time.Time
}

type ChatRepository interface {
	List(ctx context.Context, filter ChatFilter) ([]domain.Chat, error)
	Create(ctx context.Context, name string, userIDs []string, disappearAfterMinutes int) (domain.Chat, error)
	UserBelongsToChat(ctx context.Context, chatID, userID string) (bool, error)
	UnreadCountsForChat(ctx context.Context, chatID string) ([]ChatUnreadCount, error)
	ListUnreadChatsForUser(ctx context.Context, userID string) ([]UserChatUnread, error)
}
