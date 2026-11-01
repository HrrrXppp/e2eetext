package repository

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/ekhrunov/messenger/server/internal/domain"
)

// ErrKeyVersionExists is returned by AddKeyVersion when the key id is already used.
var ErrKeyVersionExists = errors.New("chat key version already exists")

// ErrAlreadyMember is returned by AddMembersWithKeyVersion when one of the users already belongs to the chat.
var ErrAlreadyMember = errors.New("user is already a chat member")

type E2EERepository interface {
	UpsertIdentityKey(ctx context.Context, userID, deviceID string, publicKey json.RawMessage) error
	GetIdentityKey(ctx context.Context, userID string) (domain.UserIdentityKey, error)

	CreateChatWithKeys(
		ctx context.Context,
		name string,
		userIDs []string,
		keyID, createdBy string,
		wraps map[string]json.RawMessage,
		disappearAfterMinutes int,
	) (domain.Chat, error)

	// AddKeyVersion inserts a new chat key version and one wrap per member in one transaction.
	// Existing versions and wraps are never changed. It returns ErrKeyVersionExists for a duplicate keyID.
	AddKeyVersion(ctx context.Context, chatID, keyID, createdBy string, wraps map[string]json.RawMessage) error

	// AddMembersWithKeyVersion adds the users to the chat, inserts a new chat key version and one wrap per
	// resulting member in one transaction. Existing versions and wraps are never changed. It returns
	// ErrAlreadyMember if any user already belongs to the chat and ErrKeyVersionExists for a duplicate keyID;
	// in both cases nothing is stored.
	AddMembersWithKeyVersion(ctx context.Context, chatID string, userIDs []string, keyID, createdBy string, wraps map[string]json.RawMessage) error

	// ListKeyWrapsForUser returns every wrap the user holds for the chat, oldest first (the last item is the newest key).
	ListKeyWrapsForUser(ctx context.Context, chatID, userID string) ([]domain.UserChatKeyWrap, error)
}
