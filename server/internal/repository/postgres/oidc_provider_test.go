package postgres

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestOIDCProviderRepository_List(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create sqlmock: %v", err)
	}
	defer db.Close()

	repo := NewOIDCProviderRepository(db)

	rows := sqlmock.NewRows([]string{"id", "name", "link", "picture"}).
		AddRow("11111111-1111-1111-1111-111111111111", "Google", "https://accounts.google.com", nil).
		AddRow("22222222-2222-2222-2222-222222222222", "GitHub", "https://github.com", []byte{0x01, 0x02})

	mock.ExpectQuery(`SELECT id, name, link, picture\s+FROM oidc_providers\s+ORDER BY name`).
		WillReturnRows(rows)

	providers, err := repo.List(context.Background())
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(providers) != 2 {
		t.Fatalf("List() len = %d, want 2", len(providers))
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

func TestOIDCProviderRepository_GetByName(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create sqlmock: %v", err)
	}
	defer db.Close()

	repo := NewOIDCProviderRepository(db)

	rows := sqlmock.NewRows([]string{"id", "name", "link", "picture"}).
		AddRow("11111111-1111-1111-1111-111111111111", "Google", "https://accounts.google.com", nil)

	mock.ExpectQuery(`SELECT id, name, link, picture\s+FROM oidc_providers\s+WHERE lower\(name\) = lower\(\$1\)`).
		WithArgs("google").
		WillReturnRows(rows)

	provider, err := repo.GetByName(context.Background(), "google")
	if err != nil {
		t.Fatalf("GetByName() error = %v", err)
	}

	if provider.Name != "Google" {
		t.Errorf("Name = %q, want Google", provider.Name)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

func TestOIDCProviderRepository_GetByName_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create sqlmock: %v", err)
	}
	defer db.Close()

	repo := NewOIDCProviderRepository(db)

	mock.ExpectQuery(`SELECT id, name, link, picture\s+FROM oidc_providers\s+WHERE lower\(name\) = lower\(\$1\)`).
		WithArgs("missing").
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "link", "picture"}))

	_, err = repo.GetByName(context.Background(), "missing")
	if err == nil {
		t.Fatal("GetByName() error = nil, want not found")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}
