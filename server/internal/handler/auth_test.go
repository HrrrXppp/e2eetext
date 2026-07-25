package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
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

func TestAuthHandler_Callback_GET_MissingCodeOrState(t *testing.T) {
	repo := &stubOIDCProviderRepository{}
	authService := service.NewAuthService(config.Config{}, repo)
	handler := NewAuthHandler(authService, testNode())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/google/callback?state=abc", nil)
	req.SetPathValue("provider", "google")
	rec := httptest.NewRecorder()

	handler.Callback(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

// POST is the callback method required by providers using
// response_mode=form_post (Apple): code/state/the one-time "user" field
// arrive as a POST form body instead of GET query params. This confirms the
// handler reads from the right place for POST without needing a full
// AuthService round trip (which requires real OIDC discovery).
func TestAuthHandler_Callback_POST_ReadsFormBody(t *testing.T) {
	repo := &stubOIDCProviderRepository{}
	authService := service.NewAuthService(config.Config{}, repo)
	handler := NewAuthHandler(authService, testNode())

	form := url.Values{}
	form.Set("state", "abc")
	// code intentionally omitted so we exercise the "read from POST form,
	// then validate" path without needing a working AuthService/OIDC
	// discovery to reach a 200.
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/apple/callback", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetPathValue("provider", "apple")
	rec := httptest.NewRecorder()

	handler.Callback(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}

	var body map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body["error"] != "missing oauth code or state" {
		t.Errorf("error = %q, want %q", body["error"], "missing oauth code or state")
	}
}

func TestAuthHandler_Callback_POST_SurfacesProviderErrorFromForm(t *testing.T) {
	repo := &stubOIDCProviderRepository{}
	authService := service.NewAuthService(config.Config{}, repo)
	handler := NewAuthHandler(authService, testNode())

	form := url.Values{}
	form.Set("error", "access_denied")
	form.Set("error_description", "the user cancelled")
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/apple/callback", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetPathValue("provider", "apple")
	rec := httptest.NewRecorder()

	handler.Callback(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}

	var body map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body["error"] != "the user cancelled" {
		t.Errorf("error = %q, want %q", body["error"], "the user cancelled")
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
