package repository

import (
	"context"
	"errors"
	"time"

	"github.com/ekhrunov/messenger/server/internal/domain"
)

var (
	// ErrChatMemberNotFound is returned by RemoveMember when the user is not a member of the chat.
	ErrChatMemberNotFound = errors.New("chat member not found")
	// ErrLastChatAdmin is returned by RemoveMember when the removal would leave the chat without an admin.
	ErrLastChatAdmin = errors.New("cannot remove the last chat admin")
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
	Create(ctx context.Context, name string, userIDs []string, disappearAfterMinutes int, createdBy string) (domain.Chat, error)
	UserBelongsToChat(ctx context.Context, chatID, userID string) (bool, error)
	IsAdmin(ctx context.Context, chatID, userID string) (bool, error)
	ListMembers(ctx context.Context, chatID string) ([]domain.ChatMember, error)
	// RemoveMember deletes the user's membership, unread state, key wraps and admin row for the chat in one transaction.
	RemoveMember(ctx context.Context, chatID, userID string) error
	UnreadCountsForChat(ctx context.Context, chatID string) ([]ChatUnreadCount, error)
	ListUnreadChatsForUser(ctx context.Context, userID string) ([]UserChatUnread, error)
}
