package repository

import (
	"context"
	"encoding/json"

	"github.com/ekhrunov/messenger/server/internal/domain"
)

type E2EERepository interface {
	UpsertIdentityKey(ctx context.Context, userID, deviceID string, publicKey json.RawMessage) error
	GetIdentityKey(ctx context.Context, userID string) (domain.UserIdentityKey, error)

	CreateChatWithKeys(
		ctx context.Context,
		name string,
		userIDs []string,
		keyID, createdBy string,
		wraps map[string]json.RawMessage,
	) (domain.Chat, error)

	ListKeyWrapsForUser(ctx context.Context, chatID, userID string) ([]domain.UserChatKeyWrap, error)
}
