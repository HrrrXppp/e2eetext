package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/ekhrunov/messenger/server/internal/domain"
	"github.com/ekhrunov/messenger/server/internal/repository"
)

func TestMessageRepository_List_ByChatID(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create sqlmock: %v", err)
	}
	defer db.Close()

	repo := NewMessageRepository(db)
	createdAt := time.Date(2026, 6, 11, 12, 0, 0, 0, time.UTC)
	chatID := "11111111-1111-1111-1111-111111111111"
	userID := "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"

	rows := sqlmock.NewRows([]string{"id", "chat_id", "user_id", "data", "created_at", "updated_at", "user_name", "unread"}).
		AddRow(
			"aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
			chatID,
			userID,
			"hello",
			createdAt,
			createdAt,
			"Alice",
			true,
		)

	mock.ExpectQuery(`SELECT m\.id,[\s\S]+ORDER BY m\.created_at ASC`).
		WithArgs(chatID, userID).
		WillReturnRows(rows)

	messages, err := repo.List(context.Background(), repository.MessageFilter{
		ChatID: chatID,
		UserID: userID,
	})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(messages) != 1 {
		t.Fatalf("List() len = %d, want 1", len(messages))
	}
	if messages[0].Data != "hello" {
		t.Errorf("Data = %q", messages[0].Data)
	}
	if messages[0].UserName != "Alice" {
		t.Errorf("UserName = %q, want Alice", messages[0].UserName)
	}
	if !messages[0].Unread {
		t.Fatal("Unread = false, want true")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

func TestMessageRepository_Create(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create sqlmock: %v", err)
	}
	defer db.Close()

	repo := NewMessageRepository(db)
	createdAt := time.Date(2026, 6, 11, 12, 0, 0, 0, time.UTC)
	chatID := "11111111-1111-1111-1111-111111111111"
	userID := "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"
	messageID := "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"

	rows := sqlmock.NewRows([]string{"id", "chat_id", "user_id", "data", "created_at", "updated_at"}).
		AddRow(
			messageID,
			chatID,
			userID,
			"hello world",
			createdAt,
			createdAt,
		)

	mock.ExpectBegin()
	mock.ExpectQuery(`INSERT INTO messages \(chat_id, user_id, data\)\s+VALUES \(\$1, \$2, \$3\)\s+RETURNING id, chat_id, user_id, data, created_at, updated_at`).
		WithArgs(chatID, userID, "hello world").
		WillReturnRows(rows)
	mock.ExpectExec(`INSERT INTO unread_messages \(message_id, user_id, chat_id\)\s+SELECT \$1, uc\.user_id, \$2\s+FROM user_chats uc\s+WHERE uc\.chat_id = \$2 AND uc\.user_id != \$3`).
		WithArgs(messageID, chatID, userID).
		WillReturnResult(sqlmock.NewResult(0, 2))
	mock.ExpectExec(`UPDATE chats\s+SET updated_at = \$1\s+WHERE id = \$2`).
		WithArgs(createdAt, chatID).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	message, err := repo.Create(context.Background(), domain.Message{
		ChatID: chatID,
		UserID: userID,
		Data:   "hello world",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if message.ID != messageID {
		t.Errorf("ID = %q", message.ID)
	}
	if message.Data != "hello world" {
		t.Errorf("Data = %q", message.Data)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

func TestMessageRepository_GetByID(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create sqlmock: %v", err)
	}
	defer db.Close()

	repo := NewMessageRepository(db)
	createdAt := time.Date(2026, 6, 11, 12, 0, 0, 0, time.UTC)
	messageID := "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
	chatID := "11111111-1111-1111-1111-111111111111"
	userID := "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"

	rows := sqlmock.NewRows([]string{"id", "chat_id", "user_id", "data", "created_at", "updated_at", "user_name"}).
		AddRow(messageID, chatID, userID, "hello", createdAt, createdAt, "Alice")

	mock.ExpectQuery(`SELECT m\.id, m\.chat_id, m\.user_id, m\.data, m\.created_at, m\.updated_at, COALESCE\(u\.name, ''\)\s+FROM messages m\s+INNER JOIN users u ON u\.id = m\.user_id\s+WHERE m\.id = \$1`).
		WithArgs(messageID).
		WillReturnRows(rows)

	message, err := repo.GetByID(context.Background(), messageID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if message.ID != messageID {
		t.Errorf("ID = %q", message.ID)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

func TestMessageRepository_MarkRead(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create sqlmock: %v", err)
	}
	defer db.Close()

	repo := NewMessageRepository(db)
	messageID := "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
	userID := "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"

	mock.ExpectExec(`DELETE FROM unread_messages\s+WHERE message_id = \$1 AND user_id = \$2`).
		WithArgs(messageID, userID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	if err := repo.MarkRead(context.Background(), messageID, userID); err != nil {
		t.Fatalf("MarkRead() error = %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}
