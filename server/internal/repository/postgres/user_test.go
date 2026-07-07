package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/ekhrunov/messenger/server/internal/domain"
	"github.com/ekhrunov/messenger/server/internal/repository"
)

func TestUserRepository_List_WithFilters(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create sqlmock: %v", err)
	}
	defer db.Close()

	repo := NewUserRepository(db)
	createdAt := time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC)

	rows := sqlmock.NewRows([]string{"id", "oidc_provider_id", "subject", "name", "kem_public_key", "created_at", "updated_at"}).
		AddRow(
			"11111111-1111-1111-1111-111111111111",
			"aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
			"google-subject-1",
			"Test User",
			[]byte("user-kem-public-key"),
			createdAt,
			createdAt,
		)

	mock.ExpectQuery(`SELECT id, oidc_provider_id, subject, name, kem_public_key, created_at, updated_at\s+FROM users\s+WHERE 1=1 AND subject = \$1 AND oidc_provider_id = \$2 ORDER BY created_at DESC`).
		WithArgs("google-subject-1", "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa").
		WillReturnRows(rows)

	users, err := repo.List(context.Background(), repository.UserFilter{
		Subject:        "google-subject-1",
		OIDCProviderID: "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
	})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(users) != 1 {
		t.Fatalf("List() len = %d, want 1", len(users))
	}
	if users[0].Name != "Test User" {
		t.Errorf("Name = %q", users[0].Name)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

func TestUserRepository_List_WithNameFilter(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create sqlmock: %v", err)
	}
	defer db.Close()

	repo := NewUserRepository(db)
	createdAt := time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC)

	rows := sqlmock.NewRows([]string{"id", "oidc_provider_id", "subject", "name", "kem_public_key", "created_at", "updated_at"}).
		AddRow(
			"11111111-1111-1111-1111-111111111111",
			"aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
			"google-subject-1",
			"Alice",
			[]byte("user-kem-public-key"),
			createdAt,
			createdAt,
		)

	mock.ExpectQuery(`SELECT id, oidc_provider_id, subject, name, kem_public_key, created_at, updated_at\s+FROM users\s+WHERE 1=1 AND subject = \$1 AND oidc_provider_id = \$2 AND name = \$3 ORDER BY created_at DESC`).
		WithArgs("google-subject-1", "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", "Alice").
		WillReturnRows(rows)

	users, err := repo.List(context.Background(), repository.UserFilter{
		Subject:        "google-subject-1",
		OIDCProviderID: "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
		Name:           "Alice",
	})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(users) != 1 {
		t.Fatalf("List() len = %d, want 1", len(users))
	}
	if users[0].Name != "Alice" {
		t.Errorf("Name = %q", users[0].Name)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

func TestUserRepository_Create(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create sqlmock: %v", err)
	}
	defer db.Close()

	repo := NewUserRepository(db)
	createdAt := time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC)

	rows := sqlmock.NewRows([]string{"id", "oidc_provider_id", "subject", "name", "kem_public_key", "created_at", "updated_at"}).
		AddRow(
			"11111111-1111-1111-1111-111111111111",
			"aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
			"google-subject-1",
			"Test User",
			[]byte("user-kem-public-key"),
			createdAt,
			createdAt,
		)

	mock.ExpectQuery(`INSERT INTO users \(oidc_provider_id, subject, name, kem_public_key\)\s+VALUES \(\$1, \$2, \$3, \$4\)\s+RETURNING id, oidc_provider_id, subject, name, kem_public_key, created_at, updated_at`).
		WithArgs("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", "google-subject-1", "Test User", []byte("user-kem-public-key")).
		WillReturnRows(rows)

	user, err := repo.Create(context.Background(), domain.User{
		OIDCProviderID: "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
		Subject:        "google-subject-1",
		Name:           "Test User",
		KemPublicKey:   []byte("user-kem-public-key"),
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if user.ID != "11111111-1111-1111-1111-111111111111" {
		t.Errorf("ID = %q", user.ID)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

func TestUserRepository_GetByID(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create sqlmock: %v", err)
	}
	defer db.Close()

	repo := NewUserRepository(db)
	createdAt := time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC)

	rows := sqlmock.NewRows([]string{"id", "oidc_provider_id", "subject", "name", "kem_public_key", "created_at", "updated_at"}).
		AddRow(
			"11111111-1111-1111-1111-111111111111",
			"aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
			"google-subject-1",
			nil,
			[]byte("user-kem-public-key"),
			createdAt,
			createdAt,
		)

	mock.ExpectQuery(`SELECT id, oidc_provider_id, subject, name, kem_public_key, created_at, updated_at\s+FROM users\s+WHERE id = \$1`).
		WithArgs("11111111-1111-1111-1111-111111111111").
		WillReturnRows(rows)

	user, err := repo.GetByID(context.Background(), "11111111-1111-1111-1111-111111111111")
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if user.Subject != "google-subject-1" {
		t.Errorf("Subject = %q", user.Subject)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

func TestUserRepository_UpdateName(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create sqlmock: %v", err)
	}
	defer db.Close()

	repo := NewUserRepository(db)
	createdAt := time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC)
	updatedAt := time.Date(2026, 6, 11, 12, 0, 0, 0, time.UTC)

	rows := sqlmock.NewRows([]string{"id", "oidc_provider_id", "subject", "name", "kem_public_key", "created_at", "updated_at"}).
		AddRow(
			"11111111-1111-1111-1111-111111111111",
			"aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
			"google-subject-1",
			"New Name",
			[]byte("user-kem-public-key"),
			createdAt,
			updatedAt,
		)

	mock.ExpectQuery(`UPDATE users\s+SET name = \$1, updated_at = NOW\(\)\s+WHERE id = \$2\s+RETURNING id, oidc_provider_id, subject, name, kem_public_key, created_at, updated_at`).
		WithArgs("New Name", "11111111-1111-1111-1111-111111111111").
		WillReturnRows(rows)

	user, err := repo.UpdateName(context.Background(), "11111111-1111-1111-1111-111111111111", "New Name")
	if err != nil {
		t.Fatalf("UpdateName() error = %v", err)
	}
	if user.Name != "New Name" {
		t.Errorf("Name = %q", user.Name)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

func TestUserRepository_Search_ByName(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create sqlmock: %v", err)
	}
	defer db.Close()

	repo := NewUserRepository(db)
	createdAt := time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC)
	nodeID := "99999999-9999-9999-9999-999999999999"

	rows := sqlmock.NewRows([]string{"id", "oidc_provider_id", "subject", "name", "kem_public_key", "created_at", "updated_at"}).
		AddRow(
			"11111111-1111-1111-1111-111111111111",
			"aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
			"google-subject-1",
			"Alice",
			[]byte("user-kem-public-key"),
			createdAt,
			createdAt,
		)

	mock.ExpectQuery(`SELECT id, oidc_provider_id, subject, name, kem_public_key, created_at, updated_at\s+FROM users WHERE \(name ILIKE \$1\) ORDER BY name NULLS LAST, created_at DESC LIMIT \$2`).
		WithArgs("%ali%", 20).
		WillReturnRows(rows)

	users, err := repo.Search(context.Background(), repository.UserSearchFilter{
		NameQuery: "ali",
		Limit:     20,
	}, nodeID)
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if len(users) != 1 {
		t.Fatalf("Search() len = %d, want 1", len(users))
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

func TestUserRepository_Search_ByScopedID(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create sqlmock: %v", err)
	}
	defer db.Close()

	repo := NewUserRepository(db)
	createdAt := time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC)
	nodeID := "99999999-9999-9999-9999-999999999999"

	rows := sqlmock.NewRows([]string{"id", "oidc_provider_id", "subject", "name", "kem_public_key", "created_at", "updated_at"}).
		AddRow(
			"11111111-1111-1111-1111-111111111111",
			"aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
			"google-subject-1",
			"Alice",
			[]byte("user-kem-public-key"),
			createdAt,
			createdAt,
		)

	mock.ExpectQuery(`SELECT id, oidc_provider_id, subject, name, kem_public_key, created_at, updated_at\s+FROM users WHERE \(\(id::text ILIKE \$1 OR \(\$2 \|\| '/' \|\| id::text\) ILIKE \$1\)\) ORDER BY name NULLS LAST, created_at DESC LIMIT \$3`).
		WithArgs("%1111%", nodeID, 20).
		WillReturnRows(rows)

	users, err := repo.Search(context.Background(), repository.UserSearchFilter{
		UserIDQuery: "1111",
		Limit:       20,
	}, nodeID)
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if len(users) != 1 {
		t.Fatalf("Search() len = %d, want 1", len(users))
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

func TestUserRepository_Search_CyrillicName(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create sqlmock: %v", err)
	}
	defer db.Close()

	repo := NewUserRepository(db)
	createdAt := time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC)
	nodeID := "99999999-9999-9999-9999-999999999999"

	rows := sqlmock.NewRows([]string{"id", "oidc_provider_id", "subject", "name", "kem_public_key", "created_at", "updated_at"}).
		AddRow(
			"11111111-1111-1111-1111-111111111111",
			"aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
			"google-subject-1",
			"Евгений Хрунов",
			[]byte("user-kem-public-key"),
			createdAt,
			createdAt,
		)

	mock.ExpectQuery(`SELECT id, oidc_provider_id, subject, name, kem_public_key, created_at, updated_at\s+FROM users WHERE \(name ILIKE \$1\) ORDER BY name NULLS LAST, created_at DESC LIMIT \$2`).
		WithArgs("%Евг%", 20).
		WillReturnRows(rows)

	users, err := repo.Search(context.Background(), repository.UserSearchFilter{
		NameQuery: "Евг",
		Limit:     20,
	}, nodeID)
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if len(users) != 1 {
		t.Fatalf("Search() len = %d, want 1", len(users))
	}
	if users[0].Name != "Евгений Хрунов" {
		t.Errorf("Name = %q", users[0].Name)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}
