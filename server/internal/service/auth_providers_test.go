package service

import (
	"context"
	"encoding/base64"
	"errors"
	"testing"

	"github.com/ekhrunov/messenger/server/internal/config"
	"github.com/ekhrunov/messenger/server/internal/domain"
)

type mockOIDCProviderRepository struct {
	providers []domain.OIDCProvider
	err       error
}

func (m *mockOIDCProviderRepository) List(_ context.Context) ([]domain.OIDCProvider, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.providers, nil
}

func (m *mockOIDCProviderRepository) GetByName(_ context.Context, name string) (domain.OIDCProvider, error) {
	for _, provider := range m.providers {
		if ProviderSlug(provider.Name) == ProviderSlug(name) {
			return provider, nil
		}
	}
	return domain.OIDCProvider{}, errors.New("oidc provider not found")
}

func TestAuthService_ListProviders(t *testing.T) {
	picture := []byte{0x89, 0x50}
	repo := &mockOIDCProviderRepository{
		providers: []domain.OIDCProvider{
			{
				ID:   "11111111-1111-1111-1111-111111111111",
				Name: "Google",
				Link: "https://accounts.google.com",
			},
			{
				ID:      "22222222-2222-2222-2222-222222222222",
				Name:    "GitHub",
				Link:    "https://token.actions.githubusercontent.com",
				Picture: picture,
			},
		},
	}

	svc := NewAuthService(config.Config{}, repo)

	views, err := svc.ListProviders(context.Background())
	if err != nil {
		t.Fatalf("ListProviders() error = %v", err)
	}

	if len(views) != 2 {
		t.Fatalf("ListProviders() len = %d, want 2", len(views))
	}

	if views[0].Slug != "google" {
		t.Errorf("views[0].Slug = %q, want google", views[0].Slug)
	}
	if views[0].Picture != "" {
		t.Errorf("views[0].Picture = %q, want empty", views[0].Picture)
	}

	if views[1].Slug != "github" {
		t.Errorf("views[1].Slug = %q, want github", views[1].Slug)
	}
	if views[1].Picture != base64.StdEncoding.EncodeToString(picture) {
		t.Errorf("views[1].Picture = %q", views[1].Picture)
	}
}

func TestAuthService_ListProviders_Error(t *testing.T) {
	repo := &mockOIDCProviderRepository{err: errors.New("database unavailable")}
	svc := NewAuthService(config.Config{}, repo)

	_, err := svc.ListProviders(context.Background())
	if err == nil {
		t.Fatal("ListProviders() error = nil, want error")
	}
}

func TestProviderSlug(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{name: "Google", want: "google"},
		{name: "  GitHub  ", want: "github"},
	}

	for _, tt := range tests {
		if got := ProviderSlug(tt.name); got != tt.want {
			t.Errorf("ProviderSlug(%q) = %q, want %q", tt.name, got, tt.want)
		}
	}
}
