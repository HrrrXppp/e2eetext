package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ekhrunov/messenger/server/internal/config"
	"github.com/ekhrunov/messenger/server/internal/domain"
	"github.com/ekhrunov/messenger/server/internal/service"
)

type stubOIDCProviderRepo struct{}

func (stubOIDCProviderRepo) List(_ context.Context) ([]domain.OIDCProvider, error) {
	return nil, nil
}

func (stubOIDCProviderRepo) GetByName(_ context.Context, _ string) (domain.OIDCProvider, error) {
	return domain.OIDCProvider{}, nil
}

func TestRequireAuth_MissingAuthorization(t *testing.T) {
	auth := service.NewAuthService(config.Config{}, stubOIDCProviderRepo{})
	nextCalled := false
	handler := RequireAuth(auth)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/chat", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if nextCalled {
		t.Fatal("next handler was called without authorization")
	}
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}

	var body map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body["error"] == "" {
		t.Fatal("expected error message in response body")
	}
}

func TestRequireAuth_InvalidAuthorizationHeader(t *testing.T) {
	auth := service.NewAuthService(config.Config{}, stubOIDCProviderRepo{})
	handler := RequireAuth(auth)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/chat", nil)
	req.Header.Set("Authorization", "Token not-bearer")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}
