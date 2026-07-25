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

	rows := sqlmock.NewRows([]string{"id", "name", "link", "picture", "scopes", "response_mode", "client_secret_strategy"}).
		AddRow("11111111-1111-1111-1111-111111111111", "Google", "https://accounts.google.com", nil, "openid profile", nil, "static").
		AddRow("22222222-2222-2222-2222-222222222222", "Apple", "https://appleid.apple.com", []byte{0x01, 0x02}, "openid name", "form_post", "private_key_jwt")

	mock.ExpectQuery(`SELECT id, name, link, picture, scopes, response_mode, client_secret_strategy\s+FROM oidc_providers\s+ORDER BY name`).
		WillReturnRows(rows)

	providers, err := repo.List(context.Background())
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(providers) != 2 {
		t.Fatalf("List() len = %d, want 2", len(providers))
	}

	google := providers[0]
	if len(google.Scopes) != 2 || google.Scopes[0] != "openid" || google.Scopes[1] != "profile" {
		t.Errorf("google.Scopes = %v, want [openid profile]", google.Scopes)
	}
	if google.ResponseMode != "" {
		t.Errorf("google.ResponseMode = %q, want empty", google.ResponseMode)
	}
	if google.ClientSecretStrategy != "static" {
		t.Errorf("google.ClientSecretStrategy = %q, want static", google.ClientSecretStrategy)
	}

	apple := providers[1]
	if len(apple.Scopes) != 2 || apple.Scopes[1] != "name" {
		t.Errorf("apple.Scopes = %v, want [openid name]", apple.Scopes)
	}
	if apple.ResponseMode != "form_post" {
		t.Errorf("apple.ResponseMode = %q, want form_post", apple.ResponseMode)
	}
	if apple.ClientSecretStrategy != "private_key_jwt" {
		t.Errorf("apple.ClientSecretStrategy = %q, want private_key_jwt", apple.ClientSecretStrategy)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

func TestOIDCProviderRepository_GetByName(t *testing.T) {
	tests := []struct {
		name         string
		slug         string
		providerName string
		link         string
		scopes       string
		responseMode any
		strategy     string
	}{
		{
			name:         "google",
			slug:         "google",
			providerName: "Google",
			link:         "https://accounts.google.com",
			scopes:       "openid profile",
			responseMode: nil,
			strategy:     "static",
		},
		{
			name:         "apple",
			slug:         "apple",
			providerName: "Apple",
			link:         "https://appleid.apple.com",
			scopes:       "openid name",
			responseMode: "form_post",
			strategy:     "private_key_jwt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatalf("create sqlmock: %v", err)
			}
			defer db.Close()

			repo := NewOIDCProviderRepository(db)

			rows := sqlmock.NewRows([]string{"id", "name", "link", "picture", "scopes", "response_mode", "client_secret_strategy"}).
				AddRow("11111111-1111-1111-1111-111111111111", tt.providerName, tt.link, nil, tt.scopes, tt.responseMode, tt.strategy)

			mock.ExpectQuery(`SELECT id, name, link, picture, scopes, response_mode, client_secret_strategy\s+FROM oidc_providers\s+WHERE lower\(name\) = lower\(\$1\)`).
				WithArgs(tt.slug).
				WillReturnRows(rows)

			provider, err := repo.GetByName(context.Background(), tt.slug)
			if err != nil {
				t.Fatalf("GetByName() error = %v", err)
			}

			if provider.Name != tt.providerName {
				t.Errorf("Name = %q, want %s", provider.Name, tt.providerName)
			}
			if provider.ClientSecretStrategy != tt.strategy {
				t.Errorf("ClientSecretStrategy = %q, want %s", provider.ClientSecretStrategy, tt.strategy)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatalf("sql expectations: %v", err)
			}
		})
	}
}

func TestOIDCProviderRepository_GetByName_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create sqlmock: %v", err)
	}
	defer db.Close()

	repo := NewOIDCProviderRepository(db)

	mock.ExpectQuery(`SELECT id, name, link, picture, scopes, response_mode, client_secret_strategy\s+FROM oidc_providers\s+WHERE lower\(name\) = lower\(\$1\)`).
		WithArgs("missing").
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "link", "picture", "scopes", "response_mode", "client_secret_strategy"}))

	_, err = repo.GetByName(context.Background(), "missing")
	if err == nil {
		t.Fatal("GetByName() error = nil, want not found")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}
