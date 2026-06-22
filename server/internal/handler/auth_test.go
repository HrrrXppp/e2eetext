package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ekhrunov/messenger/server/internal/config"
	"github.com/ekhrunov/messenger/server/internal/domain"
	"github.com/ekhrunov/messenger/server/internal/service"
)

type stubOIDCProviderRepository struct {
	providers []domain.OIDCProvider
	err       error
}

func (s *stubOIDCProviderRepository) List(_ context.Context) ([]domain.OIDCProvider, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.providers, nil
}

func (s *stubOIDCProviderRepository) GetByName(_ context.Context, name string) (domain.OIDCProvider, error) {
	for _, provider := range s.providers {
		if service.ProviderSlug(provider.Name) == service.ProviderSlug(name) {
			return provider, nil
		}
	}
	return domain.OIDCProvider{}, errors.New("oidc provider not found")
}

func TestAuthHandler_ListProviders(t *testing.T) {
	repo := &stubOIDCProviderRepository{
		providers: []domain.OIDCProvider{
			{
				ID:   "11111111-1111-1111-1111-111111111111",
				Name: "Google",
				Link: "https://accounts.google.com",
			},
		},
	}
	authService := service.NewAuthService(config.Config{}, repo)
	handler := NewAuthHandler(authService, testNode())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/providers", nil)
	rec := httptest.NewRecorder()

	handler.ListProviders(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var providers []service.ProviderView
	if err := json.NewDecoder(rec.Body).Decode(&providers); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if len(providers) != 1 {
		t.Fatalf("providers len = %d, want 1", len(providers))
	}
	if providers[0].Name != "Google" {
		t.Errorf("providers[0].Name = %q, want Google", providers[0].Name)
	}
	if providers[0].Slug != "google" {
		t.Errorf("providers[0].Slug = %q, want google", providers[0].Slug)
	}
	if providers[0].ID != scopedID("11111111-1111-1111-1111-111111111111") {
		t.Errorf("providers[0].ID = %q", providers[0].ID)
	}
}

func TestAuthHandler_Refresh_Validation(t *testing.T) {
	repo := &stubOIDCProviderRepository{}
	authService := service.NewAuthService(config.Config{}, repo)
	handler := NewAuthHandler(authService, testNode())

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", bytes.NewReader([]byte(`{"provider":"google"}`)))
	rec := httptest.NewRecorder()

	handler.Refresh(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestAuthHandler_ListProviders_Error(t *testing.T) {
	repo := &stubOIDCProviderRepository{err: errors.New("database unavailable")}
	authService := service.NewAuthService(config.Config{}, repo)
	handler := NewAuthHandler(authService, testNode())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/providers", nil)
	rec := httptest.NewRecorder()

	handler.ListProviders(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
}
