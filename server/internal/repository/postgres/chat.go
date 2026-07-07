package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/ekhrunov/messenger/server/internal/domain"
	"github.com/ekhrunov/messenger/server/internal/repository"
)

type ChatRepository struct {
	db *sql.DB
}

func NewChatRepository(db *sql.DB) *ChatRepository {
	return &ChatRepository{db: db}
}

var _ repository.ChatRepository = (*ChatRepository)(nil)

func (r *ChatRepository) List(ctx context.Context, filter repository.ChatFilter) ([]domain.Chat, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT c.id,
			uc.name,
			c.admin_user_id,
			c.kem_public_key,
			uc.wrapped_chat_private_key,
			uc.kem_ciphertext,
			c.created_at,
			c.updated_at,
			COUNT(um.message_id)::int AS unread_message_count
		FROM chats c
		INNER JOIN user_chats uc ON uc.chat_id = c.id AND uc.user_id = $1
		LEFT JOIN unread_messages um ON um.chat_id = c.id AND um.user_id = $1
		GROUP BY c.id, uc.name, c.admin_user_id, c.kem_public_key, uc.wrapped_chat_private_key, uc.kem_ciphertext, c.created_at, c.updated_at
		ORDER BY c.updated_at DESC`, filter.UserID)
	if err != nil {
		return nil, fmt.Errorf("list chats: %w", err)
	}
	defer rows.Close()

	var chats []domain.Chat
	for rows.Next() {
		chat, err := scanChat(rows)
		if err != nil {
			return nil, err
		}
		chats = append(chats, chat)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate chats: %w", err)
	}

	return chats, nil
}

func (r *ChatRepository) Create(ctx context.Context, name string, adminUserID string, kemPublicKey []byte, members []repository.ChatMemberKey) (domain.Chat, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.Chat{}, fmt.Errorf("begin create chat tx: %w", err)
	}
	defer tx.Rollback()

	var chat domain.Chat
	row := tx.QueryRowContext(ctx, `
		INSERT INTO chats (admin_user_id, kem_public_key)
		VALUES ($1, $2)
		RETURNING id, admin_user_id, kem_public_key, created_at, updated_at`,
		adminUserID, kemPublicKey)

	if err := row.Scan(&chat.ID, &chat.AdminUserID, &chat.KemPublicKey, &chat.CreatedAt, &chat.UpdatedAt); err != nil {
		return domain.Chat{}, fmt.Errorf("insert chat: %w", err)
	}

	chat.Name = name

	for _, member := range members {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO user_chats (chat_id, user_id, name, wrapped_chat_private_key, kem_ciphertext)
			VALUES ($1, $2, $3, $4, $5)`,
			chat.ID, member.UserID, nullString(name), member.WrappedChatPrivateKey, member.KemCiphertext); err != nil {
			return domain.Chat{}, fmt.Errorf("insert user_chat: %w", err)
		}

		if member.UserID == adminUserID {
			chat.WrappedChatPrivateKey = member.WrappedChatPrivateKey
			chat.KemCiphertext = member.KemCiphertext
		}
	}

	if err := tx.Commit(); err != nil {
		return domain.Chat{}, fmt.Errorf("commit create chat tx: %w", err)
	}

	return chat, nil
}

func (r *ChatRepository) UserBelongsToChat(ctx context.Context, chatID, userID string) (bool, error) {
	var belongs bool
	if err := r.db.QueryRowContext(ctx, `
		SELECT EXISTS(
			SELECT 1
			FROM user_chats
			WHERE chat_id = $1 AND user_id = $2
		)`, chatID, userID).Scan(&belongs); err != nil {
		return false, fmt.Errorf("check chat membership: %w", err)
	}

	return belongs, nil
}

func (r *ChatRepository) UnreadCountsForChat(ctx context.Context, chatID string) ([]repository.ChatUnreadCount, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT uc.user_id, COUNT(um.message_id)::int
		FROM user_chats uc
		LEFT JOIN unread_messages um
			ON um.chat_id = uc.chat_id AND um.user_id = uc.user_id
		WHERE uc.chat_id = $1
		GROUP BY uc.user_id`, chatID)
	if err != nil {
		return nil, fmt.Errorf("unread counts for chat: %w", err)
	}
	defer rows.Close()

	var counts []repository.ChatUnreadCount
	for rows.Next() {
		var item repository.ChatUnreadCount
		if err := rows.Scan(&item.UserID, &item.Count); err != nil {
			return nil, fmt.Errorf("scan unread count: %w", err)
		}
		counts = append(counts, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate unread counts: %w", err)
	}

	return counts, nil
}

func (r *ChatRepository) ListUnreadChatsForUser(ctx context.Context, userID string) ([]repository.UserChatUnread, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT um.chat_id, COUNT(um.message_id)::int, c.updated_at
		FROM unread_messages um
		INNER JOIN chats c ON c.id = um.chat_id
		WHERE um.user_id = $1
		GROUP BY um.chat_id, c.updated_at
		ORDER BY c.updated_at DESC`, userID)
	if err != nil {
		return nil, fmt.Errorf("list unread chats for user: %w", err)
	}
	defer rows.Close()

	var chats []repository.UserChatUnread
	for rows.Next() {
		var item repository.UserChatUnread
		if err := rows.Scan(&item.ChatID, &item.Count, &item.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan unread chat: %w", err)
		}
		chats = append(chats, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate unread chats: %w", err)
	}

	return chats, nil
}

type chatRow interface {
	Scan(dest ...any) error
}

func scanChat(row chatRow) (domain.Chat, error) {
	var chat domain.Chat
	var name sql.NullString

	if err := row.Scan(
		&chat.ID,
		&name,
		&chat.AdminUserID,
		&chat.KemPublicKey,
		&chat.WrappedChatPrivateKey,
		&chat.KemCiphertext,
		&chat.CreatedAt,
		&chat.UpdatedAt,
		&chat.UnreadMessageCount,
	); err != nil {
		return domain.Chat{}, fmt.Errorf("scan chat: %w", err)
	}

	chat.Name = name.String

	return chat, nil
}
