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

	rows := sqlmock.NewRows([]string{"id", "name", "disappear_after_minutes", "created_by", "is_admin", "created_at", "updated_at", "unread_message_count"}).
		AddRow("11111111-1111-1111-1111-111111111111", "General", 86400, userID, true, createdAt, createdAt, 3).
		AddRow("22222222-2222-2222-2222-222222222222", "Random", 86400, nil, false, createdAt, createdAt, 0)

	mock.ExpectQuery(`SELECT c\.id,[\s\S]+uc\.name,[\s\S]+c\.disappear_after_minutes,[\s\S]+ORDER BY c\.updated_at DESC`).
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
	if !chats[0].IsAdmin {
		t.Error("first chat IsAdmin = false, want true")
	}
	if chats[1].UnreadMessageCount != 0 {
		t.Errorf("second chat UnreadMessageCount = %d, want 0", chats[1].UnreadMessageCount)
	}
	if chats[1].IsAdmin {
		t.Error("second chat IsAdmin = true, want false")
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

	rows := sqlmock.NewRows([]string{"id", "disappear_after_minutes", "created_by", "created_at", "updated_at"}).
		AddRow(chatID, 86400, userID1, createdAt, createdAt)

	mock.ExpectBegin()
	mock.ExpectQuery(`INSERT INTO chats \(disappear_after_minutes, created_by\)\s+VALUES \(\$1, \$2\)\s+RETURNING id, disappear_after_minutes, created_by, created_at, updated_at`).
		WithArgs(86400, userID1).
		WillReturnRows(rows)
	mock.ExpectExec(`INSERT INTO user_chats \(chat_id, user_id, name\)\s+VALUES \(\$1, \$2, \$3\)`).
		WithArgs(chatID, userID1, "Project Chat").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(`INSERT INTO user_chats \(chat_id, user_id, name\)\s+VALUES \(\$1, \$2, \$3\)`).
		WithArgs(chatID, userID2, "Project Chat").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(`INSERT INTO chat_admins \(chat_id, user_id\)\s+VALUES \(\$1, \$2\)`).
		WithArgs(chatID, userID1).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	chat, err := repo.Create(context.Background(), "Project Chat", []string{userID1, userID2}, 86400, userID1)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if chat.ID != chatID {
		t.Errorf("ID = %q", chat.ID)
	}
	if chat.Name != "Project Chat" {
		t.Errorf("Name = %q", chat.Name)
	}
	if chat.DisappearAfterMinutes != 86400 {
		t.Errorf("DisappearAfterMinutes = %d, want 86400", chat.DisappearAfterMinutes)
	}
	if chat.CreatedBy != userID1 {
		t.Errorf("CreatedBy = %q, want %q", chat.CreatedBy, userID1)
	}
	if !chat.IsAdmin {
		t.Error("IsAdmin = false, want true for the creator")
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

func TestChatRepository_UnreadCountsForChat(t *testing.T) {
	chatID := "11111111-1111-1111-1111-111111111111"
	userAlice := "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
	userBob := "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"
	userCarol := "cccccccc-cccc-cccc-cccc-cccccccccccc"
	otherChatID := "22222222-2222-2222-2222-222222222222"

	tests := []struct {
		name   string
		chatID string
		rows   []repository.ChatUnreadCount
		want   []repository.ChatUnreadCount
	}{
		{
			name:   "two members mixed unread",
			chatID: chatID,
			rows: []repository.ChatUnreadCount{
				{UserID: userAlice, Count: 3},
				{UserID: userBob, Count: 0},
			},
			want: []repository.ChatUnreadCount{
				{UserID: userAlice, Count: 3},
				{UserID: userBob, Count: 0},
			},
		},
		{
			name:   "three members different counts",
			chatID: chatID,
			rows: []repository.ChatUnreadCount{
				{UserID: userAlice, Count: 0},
				{UserID: userBob, Count: 1},
				{UserID: userCarol, Count: 5},
			},
			want: []repository.ChatUnreadCount{
				{UserID: userAlice, Count: 0},
				{UserID: userBob, Count: 1},
				{UserID: userCarol, Count: 5},
			},
		},
		{
			name:   "single member with unread",
			chatID: chatID,
			rows: []repository.ChatUnreadCount{
				{UserID: userAlice, Count: 2},
			},
			want: []repository.ChatUnreadCount{
				{UserID: userAlice, Count: 2},
			},
		},
		{
			name:   "single member all read",
			chatID: chatID,
			rows: []repository.ChatUnreadCount{
				{UserID: userBob, Count: 0},
			},
			want: []repository.ChatUnreadCount{
				{UserID: userBob, Count: 0},
			},
		},
		{
			name:   "no members",
			chatID: chatID,
			want:   nil,
		},
		{
			name:   "different chat id",
			chatID: otherChatID,
			rows: []repository.ChatUnreadCount{
				{UserID: userAlice, Count: 4},
				{UserID: userCarol, Count: 2},
			},
			want: []repository.ChatUnreadCount{
				{UserID: userAlice, Count: 4},
				{UserID: userCarol, Count: 2},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatalf("create sqlmock: %v", err)
			}
			defer db.Close()

			repo := NewChatRepository(db)

			sqlRows := sqlmock.NewRows([]string{"user_id", "count"})
			for _, row := range tt.rows {
				sqlRows.AddRow(row.UserID, row.Count)
			}

			mock.ExpectQuery(`SELECT uc\.user_id, COUNT\(um\.message_id\)::int[\s\S]+GROUP BY uc\.user_id`).
				WithArgs(tt.chatID).
				WillReturnRows(sqlRows)

			got, err := repo.UnreadCountsForChat(context.Background(), tt.chatID)
			if err != nil {
				t.Fatalf("UnreadCountsForChat() error = %v", err)
			}
			if len(got) != len(tt.want) {
				t.Fatalf("UnreadCountsForChat() len = %d, want %d", len(got), len(tt.want))
			}
			for i := range tt.want {
				if got[i].UserID != tt.want[i].UserID {
					t.Errorf("got[%d].UserID = %q, want %q", i, got[i].UserID, tt.want[i].UserID)
				}
				if got[i].Count != tt.want[i].Count {
					t.Errorf("got[%d].Count = %d, want %d", i, got[i].Count, tt.want[i].Count)
				}
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatalf("sql expectations: %v", err)
			}
		})
	}
}

func TestChatRepository_ListUnreadChatsForUser(t *testing.T) {
	userID := "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
	chatID1 := "11111111-1111-1111-1111-111111111111"
	chatID2 := "22222222-2222-2222-2222-222222222222"
	updatedAt1 := time.Date(2026, 6, 12, 10, 0, 0, 0, time.UTC)
	updatedAt2 := time.Date(2026, 6, 11, 8, 0, 0, 0, time.UTC)

	tests := []struct {
		name    string
		userID  string
		rows    []repository.UserChatUnread
		wantLen int
	}{
		{
			name:   "multiple chats with unread",
			userID: userID,
			rows: []repository.UserChatUnread{
				{ChatID: chatID1, Count: 3, UpdatedAt: updatedAt1},
				{ChatID: chatID2, Count: 1, UpdatedAt: updatedAt2},
			},
			wantLen: 2,
		},
		{
			name:    "no unread chats",
			userID:  userID,
			wantLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatalf("create sqlmock: %v", err)
			}
			defer db.Close()

			repo := NewChatRepository(db)

			sqlRows := sqlmock.NewRows([]string{"chat_id", "count", "updated_at"})
			for _, row := range tt.rows {
				sqlRows.AddRow(row.ChatID, row.Count, row.UpdatedAt)
			}

			mock.ExpectQuery(`SELECT um\.chat_id, COUNT\(um\.message_id\)::int, c\.updated_at[\s\S]+ORDER BY c\.updated_at DESC`).
				WithArgs(tt.userID).
				WillReturnRows(sqlRows)

			got, err := repo.ListUnreadChatsForUser(context.Background(), tt.userID)
			if err != nil {
				t.Fatalf("ListUnreadChatsForUser() error = %v", err)
			}
			if len(got) != tt.wantLen {
				t.Fatalf("ListUnreadChatsForUser() len = %d, want %d", len(got), tt.wantLen)
			}
			for i, row := range tt.rows {
				if got[i].ChatID != row.ChatID {
					t.Errorf("got[%d].ChatID = %q, want %q", i, got[i].ChatID, row.ChatID)
				}
				if got[i].Count != row.Count {
					t.Errorf("got[%d].Count = %d, want %d", i, got[i].Count, row.Count)
				}
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatalf("sql expectations: %v", err)
			}
		})
	}
}
