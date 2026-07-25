package app

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/ekhrunov/messenger/server/internal/config"
	"github.com/ekhrunov/messenger/server/internal/node"
)

func newTestHandler(t *testing.T) http.Handler {
	t.Helper()

	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create sqlmock: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	application, err := New(config.Config{}, logger, db, node.Registry{ID: "99999999-9999-9999-9999-999999999999"})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	return application.Handler()
}

func parseAllow(header string) map[string]struct{} {
	methods := make(map[string]struct{})
	for _, method := range strings.Split(header, ",") {
		method = strings.TrimSpace(method)
		if method != "" {
			methods[method] = struct{}{}
		}
	}
	return methods
}

func assertAllowContains(t *testing.T, got string, want ...string) {
	t.Helper()

	methods := parseAllow(got)
	for _, method := range want {
		if _, ok := methods[method]; !ok {
			t.Fatalf("Allow = %q, want methods %v", got, want)
		}
	}
}

func assertMethodNotAllowed(
	t *testing.T,
	handler http.Handler,
	method string,
	path string,
	allowed ...string,
) {
	t.Helper()

	req := httptest.NewRequest(method, path, nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("%s %s: status = %d, want %d", method, path, rec.Code, http.StatusMethodNotAllowed)
	}
	assertAllowContains(t, rec.Header().Get("Allow"), allowed...)
}

func assertMethodAllowed(t *testing.T, handler http.Handler, method string, path string) {
	t.Helper()

	req := httptest.NewRequest(method, path, nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code == http.StatusMethodNotAllowed {
		t.Fatalf("%s %s: status = %d, method should be allowed", method, path, rec.Code)
	}
}

func TestAllowedMethods_Health(t *testing.T) {
	handler := newTestHandler(t)

	assertMethodAllowed(t, handler, http.MethodGet, "/health")
	assertMethodNotAllowed(t, handler, http.MethodPost, "/health", http.MethodGet)
	assertMethodNotAllowed(t, handler, http.MethodPut, "/health", http.MethodGet)
	assertMethodNotAllowed(t, handler, http.MethodPatch, "/health", http.MethodGet)
	assertMethodNotAllowed(t, handler, http.MethodDelete, "/health", http.MethodGet)
}

func TestAllowedMethods_AuthProviders(t *testing.T) {
	handler := newTestHandler(t)

	assertMethodAllowed(t, handler, http.MethodGet, "/api/v1/auth/providers")
	assertMethodNotAllowed(t, handler, http.MethodPost, "/api/v1/auth/providers", http.MethodGet)
	assertMethodNotAllowed(t, handler, http.MethodDelete, "/api/v1/auth/providers", http.MethodGet)
}

func TestAllowedMethods_AuthLogin(t *testing.T) {
	handler := newTestHandler(t)

	assertMethodAllowed(t, handler, http.MethodGet, "/api/v1/auth/google/login")
	assertMethodNotAllowed(t, handler, http.MethodPost, "/api/v1/auth/google/login", http.MethodGet)
	assertMethodNotAllowed(t, handler, http.MethodDelete, "/api/v1/auth/google/login", http.MethodGet)
}

func TestAllowedMethods_AuthCallback(t *testing.T) {
	handler := newTestHandler(t)

	// GET is the default 302-redirect callback (e.g. Google); POST is
	// required by providers using response_mode=form_post (e.g. Apple
	// whenever name/email scopes are requested) — both hit the same URL.
	assertMethodAllowed(t, handler, http.MethodGet, "/api/v1/auth/google/callback")
	assertMethodAllowed(t, handler, http.MethodPost, "/api/v1/auth/google/callback")
	assertMethodNotAllowed(t, handler, http.MethodDelete, "/api/v1/auth/google/callback", http.MethodGet, http.MethodPost)
}

func TestAllowedMethods_AuthRefresh(t *testing.T) {
	handler := newTestHandler(t)

	assertMethodAllowed(t, handler, http.MethodPost, "/api/v1/auth/refresh")
	assertMethodNotAllowed(t, handler, http.MethodGet, "/api/v1/auth/refresh", http.MethodPost)
	assertMethodNotAllowed(t, handler, http.MethodDelete, "/api/v1/auth/refresh", http.MethodPost)
}

func TestAllowedMethods_UserCollection(t *testing.T) {
	handler := newTestHandler(t)

	assertMethodAllowed(t, handler, http.MethodGet, "/api/v1/user")
	assertMethodAllowed(t, handler, http.MethodPost, "/api/v1/user")
	assertMethodNotAllowed(t, handler, http.MethodPut, "/api/v1/user", http.MethodGet, http.MethodPost)
	assertMethodNotAllowed(t, handler, http.MethodPatch, "/api/v1/user", http.MethodGet, http.MethodPost)
	assertMethodNotAllowed(t, handler, http.MethodDelete, "/api/v1/user", http.MethodGet, http.MethodPost)
}

func TestAllowedMethods_Search(t *testing.T) {
	handler := newTestHandler(t)

	assertMethodAllowed(t, handler, http.MethodGet, "/api/v1/search")
	assertMethodNotAllowed(t, handler, http.MethodPost, "/api/v1/search", http.MethodGet)
	assertMethodNotAllowed(t, handler, http.MethodPatch, "/api/v1/search", http.MethodGet)
	assertMethodNotAllowed(t, handler, http.MethodDelete, "/api/v1/search", http.MethodGet)
}

func TestApp_SearchRouteReturnsUnauthorizedWithoutAuth(t *testing.T) {
	handler := newTestHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/search?q=test", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code == http.StatusNotFound {
		t.Fatal("GET /api/search returned 404; route is not registered")
	}
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestAllowedMethods_UserByID(t *testing.T) {
	handler := newTestHandler(t)
	scopedPath := "/api/v1/user/99999999-9999-9999-9999-999999999999/11111111-1111-1111-1111-111111111111"

	assertMethodAllowed(t, handler, http.MethodGet, scopedPath)
	assertMethodAllowed(t, handler, http.MethodPatch, scopedPath)
	assertMethodNotAllowed(t, handler, http.MethodPost, scopedPath, http.MethodGet, http.MethodPatch)
	assertMethodNotAllowed(t, handler, http.MethodPut, scopedPath, http.MethodGet, http.MethodPatch)
	assertMethodNotAllowed(t, handler, http.MethodDelete, scopedPath, http.MethodGet, http.MethodPatch)
}

func TestAllowedMethods_Chat(t *testing.T) {
	handler := newTestHandler(t)

	assertMethodAllowed(t, handler, http.MethodGet, "/api/v1/chat")
	assertMethodAllowed(t, handler, http.MethodPost, "/api/v1/chat")
	assertMethodNotAllowed(t, handler, http.MethodPut, "/api/v1/chat", http.MethodGet, http.MethodPost)
	assertMethodNotAllowed(t, handler, http.MethodPatch, "/api/v1/chat", http.MethodGet, http.MethodPost)
	assertMethodNotAllowed(t, handler, http.MethodDelete, "/api/v1/chat", http.MethodGet, http.MethodPost)
}

func TestAllowedMethods_Message(t *testing.T) {
	handler := newTestHandler(t)
	scopedPath := "/api/v1/message/99999999-9999-9999-9999-999999999999/11111111-1111-1111-1111-111111111111"

	assertMethodAllowed(t, handler, http.MethodGet, "/api/v1/message")
	assertMethodAllowed(t, handler, http.MethodPost, "/api/v1/message")
	assertMethodAllowed(t, handler, http.MethodPatch, scopedPath)
	assertMethodNotAllowed(t, handler, http.MethodPut, "/api/v1/message", http.MethodGet, http.MethodPost)
	assertMethodNotAllowed(t, handler, http.MethodPatch, "/api/v1/message", http.MethodGet, http.MethodPost)
	assertMethodNotAllowed(t, handler, http.MethodDelete, "/api/v1/message", http.MethodGet, http.MethodPost)
}

func TestAllowedMethods_WebSocket(t *testing.T) {
	handler := newTestHandler(t)

	assertMethodAllowed(t, handler, http.MethodGet, "/api/v1/ws")
	assertMethodNotAllowed(t, handler, http.MethodPost, "/api/v1/ws", http.MethodGet)
	assertMethodNotAllowed(t, handler, http.MethodDelete, "/api/v1/ws", http.MethodGet)
}
