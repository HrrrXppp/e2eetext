package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/ekhrunov/messenger/server/internal/repository"
)

func TestChatRepository_List_ByUserID(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create sqlmock: %v", err)
	}
	defer db.Close()

	repo := NewChatRepository(db)
	createdAt := time.Date(2026, 6, 11, 12, 0, 0, 0, time.UTC)
	userID := "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"

	rows := sqlmock.NewRows([]string{"id", "name", "created_at", "updated_at", "unread_message_count"}).
		AddRow("11111111-1111-1111-1111-111111111111", "General", createdAt, createdAt, 3).
		AddRow("22222222-2222-2222-2222-222222222222", "Random", createdAt, createdAt, 0)

	mock.ExpectQuery(`SELECT c\.id, uc\.name, c\.created_at,[\s\S]+ORDER BY c\.updated_at DESC`).
		WithArgs(userID).
		WillReturnRows(rows)

	chats, err := repo.List(context.Background(), repository.ChatFilter{UserID: userID})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(chats) != 2 {
		t.Fatalf("List() len = %d, want 2", len(chats))
	}
	if chats[0].ID != "11111111-1111-1111-1111-111111111111" {
		t.Errorf("first chat ID = %q", chats[0].ID)
	}
	if chats[0].Name != "General" {
		t.Errorf("first chat Name = %q", chats[0].Name)
	}
	if chats[0].UnreadMessageCount != 3 {
		t.Errorf("first chat UnreadMessageCount = %d, want 3", chats[0].UnreadMessageCount)
	}
	if chats[1].UnreadMessageCount != 0 {
		t.Errorf("second chat UnreadMessageCount = %d, want 0", chats[1].UnreadMessageCount)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

func TestChatRepository_Create(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create sqlmock: %v", err)
	}
	defer db.Close()

	repo := NewChatRepository(db)
	createdAt := time.Date(2026, 6, 11, 12, 0, 0, 0, time.UTC)
	chatID := "11111111-1111-1111-1111-111111111111"
	userID1 := "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
	userID2 := "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"

	rows := sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).
		AddRow(chatID, createdAt, createdAt)

	mock.ExpectBegin()
	mock.ExpectQuery(`INSERT INTO chats DEFAULT VALUES\s+RETURNING id, created_at, updated_at`).
		WillReturnRows(rows)
	mock.ExpectExec(`INSERT INTO user_chats \(chat_id, user_id, name\)\s+VALUES \(\$1, \$2, \$3\)`).
		WithArgs(chatID, userID1, "Project Chat").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(`INSERT INTO user_chats \(chat_id, user_id, name\)\s+VALUES \(\$1, \$2, \$3\)`).
		WithArgs(chatID, userID2, "Project Chat").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	chat, err := repo.Create(context.Background(), "Project Chat", []string{userID1, userID2})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if chat.ID != chatID {
		t.Errorf("ID = %q", chat.ID)
	}
	if chat.Name != "Project Chat" {
		t.Errorf("Name = %q", chat.Name)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

func TestChatRepository_UserBelongsToChat(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create sqlmock: %v", err)
	}
	defer db.Close()

	repo := NewChatRepository(db)
	chatID := "11111111-1111-1111-1111-111111111111"
	userID := "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"

	rows := sqlmock.NewRows([]string{"exists"}).AddRow(true)
	mock.ExpectQuery(`SELECT EXISTS\(\s*SELECT 1\s+FROM user_chats\s+WHERE chat_id = \$1 AND user_id = \$2\s*\)`).
		WithArgs(chatID, userID).
		WillReturnRows(rows)

	belongs, err := repo.UserBelongsToChat(context.Background(), chatID, userID)
	if err != nil {
		t.Fatalf("UserBelongsToChat() error = %v", err)
	}
	if !belongs {
		t.Fatal("UserBelongsToChat() = false, want true")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}
