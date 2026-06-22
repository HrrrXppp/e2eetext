package handler

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ekhrunov/messenger/server/internal/config"
	"github.com/ekhrunov/messenger/server/internal/service"
	"github.com/ekhrunov/messenger/server/internal/ws"
)

func TestWSHandler_UnauthorizedWithoutToken(t *testing.T) {
	authService := service.NewAuthService(config.Config{}, nil)
	handler := NewWSHandler(
		authService,
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		ws.NewHub(),
		nil,
		nil,
	)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/ws", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}
