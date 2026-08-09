//go:build integration

package integration

import (
	"bytes"
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/ekhrunov/messenger/server/internal/app"
	"github.com/ekhrunov/messenger/server/internal/config"
	"github.com/ekhrunov/messenger/server/internal/database"
	"github.com/ekhrunov/messenger/server/internal/node"
)

const mockOAuthClientID = "integration-test-client"

// harness boots the real HTTP application (real Postgres, real handlers,
// real auth verification) behind an httptest.Server, plus a mock OIDC
// issuer so tests can mint valid ID tokens without touching Google.
type harness struct {
	t      *testing.T
	server *httptest.Server
	db     *sql.DB
	oidc   *mockOIDCIssuer
}

// newHarness requires a reachable Postgres via the INTEGRATION_DATABASE_URL
// (falling back to DATABASE_URL) environment variable — e.g.
// `docker-compose up -d postgres` locally. Tests are skipped, not failed,
// when neither is set, so `go test ./...` (no -tags=integration, so this
// file isn't even compiled) and `go test -tags=integration ./...` without a
// database both stay green with a clear reason.
func newHarness(t *testing.T) *harness {
	t.Helper()

	databaseURL := os.Getenv("INTEGRATION_DATABASE_URL")
	if databaseURL == "" {
		databaseURL = os.Getenv("DATABASE_URL")
	}
	if databaseURL == "" {
		t.Skip("set INTEGRATION_DATABASE_URL (or DATABASE_URL) to a reachable Postgres to run integration tests, e.g. via `docker-compose up -d postgres`")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := database.WaitFor(ctx, databaseURL); err != nil {
		t.Fatalf("database not reachable: %v", err)
	}
	if err := database.Migrate(databaseURL); err != nil {
		t.Fatalf("run migrations: %v", err)
	}

	db, err := database.Open(databaseURL)
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	resetData(t, db)

	oidcIssuer, err := newMockOIDCIssuer()
	if err != nil {
		t.Fatalf("start mock oidc issuer: %v", err)
	}
	t.Cleanup(oidcIssuer.Close)

	seedOIDCProvider(t, db, oidcIssuer.URL())

	cfg := config.Config{
		Addr:        "127.0.0.1:0",
		DatabaseURL: databaseURL,
		OAuthCredentials: map[string]config.OAuthCredential{
			"mock": {ClientID: mockOAuthClientID, ClientSecret: "unused-in-tests"},
		},
	}

	nodeRegistry, err := node.Load(ctx, db)
	if err != nil {
		t.Fatalf("load node registry: %v", err)
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	application, err := app.New(cfg, logger, db, nodeRegistry)
	if err != nil {
		t.Fatalf("build app: %v", err)
	}

	server := httptest.NewServer(application.Handler())
	t.Cleanup(server.Close)

	return &harness{t: t, server: server, db: db, oidc: oidcIssuer}
}

// resetData truncates every table this test suite touches so runs are
// idempotent regardless of what a previous (possibly failed) run left
// behind. Scoped to exactly the tables integration tests write to.
func resetData(t *testing.T, db *sql.DB) {
	t.Helper()
	_, err := db.Exec(`
		TRUNCATE TABLE
			unread_messages,
			messages,
			user_chat_key_wraps,
			chat_key_versions,
			chat_admins,
			user_chats,
			chats,
			user_identity_keys,
			users
		RESTART IDENTITY CASCADE`)
	if err != nil {
		// Tables may not exist yet on a brand-new database before the first
		// migration runs in this process; ignore and let Migrate create them.
		t.Logf("reset data (ignored, likely pre-migration): %v", err)
	}
}

func seedOIDCProvider(t *testing.T, db *sql.DB, issuerURL string) string {
	t.Helper()

	var id string
	err := db.QueryRow(`
		INSERT INTO oidc_providers (name, link)
		VALUES ('Mock', $1)
		RETURNING id`, issuerURL).Scan(&id)
	if err != nil {
		t.Fatalf("seed mock oidc provider: %v", err)
	}
	return id
}

// user represents a signed-in integration-test identity: a mock ID token
// good enough to authenticate as, plus the resolved server-side user ID
// once registered via POST /api/v1/user.
type user struct {
	Subject string
	IDToken string
	ID      string // scoped id, e.g. "<nodeId>/<localId>"
}

func (h *harness) newUser(name string) *user {
	h.t.Helper()

	subject := name + "-" + randomHex(h.t, 8)
	idToken, err := h.oidc.issueIDToken(subject, name, mockOAuthClientID, time.Hour)
	if err != nil {
		h.t.Fatalf("issue id token for %s: %v", name, err)
	}

	return &user{Subject: subject, IDToken: idToken}
}

func randomHex(t *testing.T, n int) string {
	t.Helper()
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		t.Fatalf("generate random suffix: %v", err)
	}
	return hex.EncodeToString(buf)
}

// do sends a JSON request with the given user's bearer token (or none, if
// user is nil) and decodes a JSON response into out (if non-nil).
func (h *harness) do(method, path string, body any, actor *user, out any) *http.Response {
	h.t.Helper()

	var reqBody io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			h.t.Fatalf("marshal request body: %v", err)
		}
		reqBody = bytes.NewReader(raw)
	}

	req, err := http.NewRequest(method, h.server.URL+path, reqBody)
	if err != nil {
		h.t.Fatalf("build request: %v", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if actor != nil {
		req.Header.Set("Authorization", "Bearer "+actor.IDToken)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		h.t.Fatalf("do request %s %s: %v", method, path, err)
	}
	defer resp.Body.Close()

	if out != nil {
		if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
			h.t.Fatalf("decode response body for %s %s: %v", method, path, err)
		}
	} else {
		_, _ = io.Copy(io.Discard, resp.Body)
	}

	return resp
}
