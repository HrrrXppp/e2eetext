package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/ekhrunov/messenger/server/internal/domain"
	"github.com/ekhrunov/messenger/server/internal/repository"
)

type MessageRepository struct {
	db *sql.DB
}

func NewMessageRepository(db *sql.DB) *MessageRepository {
	return &MessageRepository{db: db}
}

var _ repository.MessageRepository = (*MessageRepository)(nil)

func (r *MessageRepository) List(ctx context.Context, filter repository.MessageFilter) ([]domain.Message, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT m.id,
			m.chat_id,
			m.user_id,
			m.data,
			m.created_at,
			m.updated_at,
			COALESCE(u.name, '') AS user_name,
			EXISTS(
				SELECT 1
				FROM unread_messages um
				WHERE um.message_id = m.id AND um.user_id = $2
			) AS unread
		FROM messages m
		INNER JOIN users u ON u.id = m.user_id
		INNER JOIN chats c ON c.id = m.chat_id
		WHERE m.chat_id = $1
			AND m.created_at > NOW() - (c.disappear_after_minutes * INTERVAL '1 minute')
		ORDER BY m.created_at ASC`, filter.ChatID, filter.UserID)
	if err != nil {
		return nil, fmt.Errorf("list messages: %w", err)
	}
	defer rows.Close()

	var messages []domain.Message
	for rows.Next() {
		message, err := scanMessageWithUnread(rows)
		if err != nil {
			return nil, err
		}
		messages = append(messages, message)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate messages: %w", err)
	}

	return messages, nil
}

func (r *MessageRepository) Create(ctx context.Context, message domain.Message) (domain.Message, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.Message{}, fmt.Errorf("begin create message tx: %w", err)
	}
	defer tx.Rollback()

	row := tx.QueryRowContext(ctx, `
		INSERT INTO messages (chat_id, user_id, data)
		VALUES ($1, $2, $3)
		RETURNING id, chat_id, user_id, data, created_at, updated_at`,
		message.ChatID,
		message.UserID,
		message.Data,
	)

	created, err := scanMessage(row)
	if err != nil {
		return domain.Message{}, err
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO unread_messages (message_id, user_id, chat_id)
		SELECT $1, uc.user_id, $2
		FROM user_chats uc
		WHERE uc.chat_id = $2 AND uc.user_id != $3`,
		created.ID,
		message.ChatID,
		message.UserID,
	); err != nil {
		return domain.Message{}, fmt.Errorf("insert unread messages: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `
		UPDATE chats
		SET updated_at = $1
		WHERE id = $2`, created.CreatedAt, message.ChatID); err != nil {
		return domain.Message{}, fmt.Errorf("touch chat updated_at: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return domain.Message{}, fmt.Errorf("commit create message tx: %w", err)
	}

	return created, nil
}

func (r *MessageRepository) GetByID(ctx context.Context, id string) (domain.Message, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT m.id, m.chat_id, m.user_id, m.data, m.created_at, m.updated_at, COALESCE(u.name, '')
		FROM messages m
		INNER JOIN users u ON u.id = m.user_id
		WHERE m.id = $1`, id)

	message, err := scanMessageWithUserName(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Message{}, repository.ErrMessageNotFound
		}
		return domain.Message{}, err
	}
	return message, nil
}

func (r *MessageRepository) MarkRead(ctx context.Context, messageID, userID string) error {
	if _, err := r.db.ExecContext(ctx, `
		DELETE FROM unread_messages
		WHERE message_id = $1 AND user_id = $2`, messageID, userID); err != nil {
		return fmt.Errorf("mark message read: %w", err)
	}
	return nil
}

type messageRow interface {
	Scan(dest ...any) error
}

func scanMessage(row messageRow) (domain.Message, error) {
	var message domain.Message
	if err := row.Scan(
		&message.ID,
		&message.ChatID,
		&message.UserID,
		&message.Data,
		&message.CreatedAt,
		&message.UpdatedAt,
	); err != nil {
		return domain.Message{}, fmt.Errorf("scan message: %w", err)
	}
	return message, nil
}

func scanMessageWithUnread(row messageRow) (domain.Message, error) {
	var message domain.Message
	if err := row.Scan(
		&message.ID,
		&message.ChatID,
		&message.UserID,
		&message.Data,
		&message.CreatedAt,
		&message.UpdatedAt,
		&message.UserName,
		&message.Unread,
	); err != nil {
		return domain.Message{}, fmt.Errorf("scan message: %w", err)
	}
	return message, nil
}

func scanMessageWithUserName(row messageRow) (domain.Message, error) {
	var message domain.Message
	if err := row.Scan(
		&message.ID,
		&message.ChatID,
		&message.UserID,
		&message.Data,
		&message.CreatedAt,
		&message.UpdatedAt,
		&message.UserName,
	); err != nil {
		return domain.Message{}, fmt.Errorf("scan message: %w", err)
	}
	return message, nil
}
