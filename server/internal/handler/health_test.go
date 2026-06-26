package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ekhrunov/messenger/server/internal/version"
)

func TestHealthHandler_ReturnsStatusAndVersion(t *testing.T) {
	handler := NewHealthHandler()
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/health", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var body map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body["status"] != "ok" {
		t.Fatalf("status = %q, want ok", body["status"])
	}
	if body["version"] != version.Version {
		t.Fatalf("version = %q, want %q", body["version"], version.Version)
	}
}
