package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/ekhrunov/messenger/server/internal/repository"
)

func TestE2EERepository_AddMembersWithKeyVersion(t *testing.T) {
	wraps := map[string]json.RawMessage{"u1": json.RawMessage(`{"v":1}`), "u2": json.RawMessage(`{"v":1}`)}

	t.Run("stores memberships, version and wraps in one transaction", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("create sqlmock: %v", err)
		}
		defer db.Close()

		mock.ExpectBegin()
		mock.ExpectExec(`INSERT INTO user_chats`).WithArgs("chat", `["u1","u2"]`).WillReturnResult(sqlmock.NewResult(0, 2))
		mock.ExpectExec(`INSERT INTO chat_key_versions`).WithArgs("key", "chat", "admin").WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectExec(`INSERT INTO user_chat_key_wraps`).WillReturnResult(sqlmock.NewResult(0, 2))
		mock.ExpectCommit()

		if err := NewE2EERepository(db).AddMembersWithKeyVersion(context.Background(), "chat", []string{"u1", "u2"}, "key", "admin", wraps); err != nil {
			t.Fatalf("AddMembersWithKeyVersion() error = %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("sql expectations: %v", err)
		}
	})

	t.Run("an existing member rolls the whole batch back", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("create sqlmock: %v", err)
		}
		defer db.Close()

		mock.ExpectBegin()
		mock.ExpectExec(`INSERT INTO user_chats`).WillReturnError(&pgconn.PgError{Code: "23505"})
		mock.ExpectRollback()

		err = NewE2EERepository(db).AddMembersWithKeyVersion(context.Background(), "chat", []string{"u1", "u2"}, "key", "admin", wraps)
		if !errors.Is(err, repository.ErrAlreadyMember) {
			t.Fatalf("error = %v, want ErrAlreadyMember", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("sql expectations: %v", err)
		}
	})

	t.Run("a reused key id rolls the memberships back", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("create sqlmock: %v", err)
		}
		defer db.Close()

		mock.ExpectBegin()
		mock.ExpectExec(`INSERT INTO user_chats`).WillReturnResult(sqlmock.NewResult(0, 2))
		mock.ExpectExec(`INSERT INTO chat_key_versions`).WillReturnError(&pgconn.PgError{Code: "23505"})
		mock.ExpectRollback()

		err = NewE2EERepository(db).AddMembersWithKeyVersion(context.Background(), "chat", []string{"u1", "u2"}, "key", "admin", wraps)
		if !errors.Is(err, repository.ErrKeyVersionExists) {
			t.Fatalf("error = %v, want ErrKeyVersionExists", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("sql expectations: %v", err)
		}
	})
}
