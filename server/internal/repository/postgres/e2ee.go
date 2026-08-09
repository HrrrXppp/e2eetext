package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/ekhrunov/messenger/server/internal/domain"
	"github.com/ekhrunov/messenger/server/internal/repository"
)

type E2EERepository struct {
	db *sql.DB
}

func NewE2EERepository(db *sql.DB) *E2EERepository {
	return &E2EERepository{db: db}
}

var _ repository.E2EERepository = (*E2EERepository)(nil)

func (r *E2EERepository) UpsertIdentityKey(ctx context.Context, userID, deviceID string, publicKey json.RawMessage) error {
	if deviceID == "" {
		deviceID = "default"
	}

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO user_identity_keys (user_id, device_id, public_key)
		VALUES ($1, $2, $3)
		ON CONFLICT (user_id) DO UPDATE
		SET device_id = EXCLUDED.device_id,
		    public_key = EXCLUDED.public_key,
		    updated_at = NOW()`, userID, deviceID, publicKey)
	if err != nil {
		return fmt.Errorf("upsert identity key: %w", err)
	}

	return nil
}

func (r *E2EERepository) GetIdentityKey(ctx context.Context, userID string) (domain.UserIdentityKey, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT user_id, device_id, public_key, created_at, updated_at
		FROM user_identity_keys
		WHERE user_id = $1`, userID)

	var item domain.UserIdentityKey
	if err := row.Scan(&item.UserID, &item.DeviceID, &item.PublicKey, &item.CreatedAt, &item.UpdatedAt); err != nil {
		if err == sql.ErrNoRows {
			return domain.UserIdentityKey{}, fmt.Errorf("identity key not found")
		}
		return domain.UserIdentityKey{}, fmt.Errorf("get identity key: %w", err)
	}

	return item, nil
}

func (r *E2EERepository) CreateChatWithKeys(
	ctx context.Context,
	name string,
	userIDs []string,
	keyID, createdBy string,
	wraps map[string]json.RawMessage,
	disappearAfterMinutes int,
) (domain.Chat, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.Chat{}, fmt.Errorf("begin create e2ee chat tx: %w", err)
	}
	defer tx.Rollback()

	var chat domain.Chat
	row := tx.QueryRowContext(ctx, `
		INSERT INTO chats (disappear_after_minutes, created_by)
		VALUES ($1, $2)
		RETURNING id, disappear_after_minutes, created_by, created_at, updated_at`, disappearAfterMinutes, createdBy)
	var chatCreatedBy sql.NullString
	if err := row.Scan(&chat.ID, &chat.DisappearAfterMinutes, &chatCreatedBy, &chat.CreatedAt, &chat.UpdatedAt); err != nil {
		return domain.Chat{}, fmt.Errorf("insert chat: %w", err)
	}
	chat.CreatedBy = chatCreatedBy.String
	chat.Name = name

	for _, userID := range userIDs {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO user_chats (chat_id, user_id, name)
			VALUES ($1, $2, $3)`, chat.ID, userID, nullString(name)); err != nil {
			return domain.Chat{}, fmt.Errorf("insert user_chat: %w", err)
		}
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO chat_admins (chat_id, user_id)
		VALUES ($1, $2)`, chat.ID, createdBy); err != nil {
		return domain.Chat{}, fmt.Errorf("insert chat admin: %w", err)
	}
	chat.IsAdmin = true

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO chat_key_versions (id, chat_id, created_by)
		VALUES ($1, $2, $3)`, keyID, chat.ID, createdBy); err != nil {
		return domain.Chat{}, fmt.Errorf("insert chat key version: %w", err)
	}

	for userID, wrap := range wraps {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO user_chat_key_wraps (chat_id, key_id, user_id, wrap)
			VALUES ($1, $2, $3, $4)`, chat.ID, keyID, userID, wrap); err != nil {
			return domain.Chat{}, fmt.Errorf("insert chat key wrap: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return domain.Chat{}, fmt.Errorf("commit create e2ee chat tx: %w", err)
	}

	return chat, nil
}

func (r *E2EERepository) ListKeyWrapsForUser(ctx context.Context, chatID, userID string) ([]domain.UserChatKeyWrap, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT chat_id, key_id, user_id, wrap, created_at
		FROM user_chat_key_wraps
		WHERE chat_id = $1 AND user_id = $2
		ORDER BY created_at DESC
		LIMIT 1`, chatID, userID)
	if err != nil {
		return nil, fmt.Errorf("list key wraps: %w", err)
	}
	defer rows.Close()

	var wraps []domain.UserChatKeyWrap
	for rows.Next() {
		var item domain.UserChatKeyWrap
		if err := rows.Scan(&item.ChatID, &item.KeyID, &item.UserID, &item.Wrap, &item.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan key wrap: %w", err)
		}
		wraps = append(wraps, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate key wraps: %w", err)
	}

	return wraps, nil
}
