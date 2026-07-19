// Command mockoidc is a throwaway, spec-compliant-enough OIDC provider used
// only by the client's Playwright E2E suite (client/e2e/). It plays the role
// of a real identity provider for a browser-driven OAuth authorization-code
// flow: discovery + JWKS + an auto-approving /authorize endpoint (no login
// form — it mints a fresh random test identity on every hit) + /token. This
// lets the real client and server exercise the exact same sign-in code path
// they use with Google in production, without any Google credentials.
//
// It is never built into a production image and is not started by
// docker-compose.yml.
package main

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	jose "github.com/go-jose/go-jose/v4"
	"github.com/go-jose/go-jose/v4/jwt"
)

const keyID = "mockoidc-e2e-key"

type pendingCode struct {
	subject string
	name    string
	nonce   string
	expires time.Time
}

type server struct {
	key     *rsa.PrivateKey
	baseURL string
	mu      sync.Mutex
	codes   map[string]pendingCode
}

func main() {
	addr := flag.String("addr", ":9998", "address to listen on")
	baseURL := flag.String("base-url", "http://127.0.0.1:9998", "externally reachable base URL for this server")
	flag.Parse()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		log.Fatalf("generate signing key: %v", err)
	}

	s := &server{key: key, baseURL: strings.TrimRight(*baseURL, "/"), codes: make(map[string]pendingCode)}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /.well-known/openid-configuration", s.handleDiscovery)
	mux.HandleFunc("GET /jwks", s.handleJWKS)
	mux.HandleFunc("GET /authorize", s.handleAuthorize)
	mux.HandleFunc("POST /token", s.handleToken)

	log.Printf("mockoidc listening on %s (base url %s)", *addr, s.baseURL)
	if err := http.ListenAndServe(*addr, mux); err != nil {
		log.Fatalf("serve: %v", err)
	}
}

func (s *server) handleDiscovery(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"issuer":                                s.baseURL,
		"authorization_endpoint":                s.baseURL + "/authorize",
		"token_endpoint":                        s.baseURL + "/token",
		"jwks_uri":                              s.baseURL + "/jwks",
		"response_types_supported":              []string{"code"},
		"subject_types_supported":               []string{"public"},
		"id_token_signing_alg_values_supported": []string{"RS256"},
		"scopes_supported":                      []string{"openid", "profile"},
		"token_endpoint_auth_methods_supported": []string{"client_secret_post", "client_secret_basic"},
	})
}

func (s *server) handleJWKS(w http.ResponseWriter, r *http.Request) {
	jwk := jose.JSONWebKey{
		Key:       &s.key.PublicKey,
		KeyID:     keyID,
		Algorithm: string(jose.RS256),
		Use:       "sig",
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(jose.JSONWebKeySet{Keys: []jose.JSONWebKey{jwk}})
}

// handleAuthorize plays the role of the provider's login page: instead of
// showing a form, it immediately "authenticates" a fresh random test user
// and redirects back to the caller's redirect_uri with an authorization
// code, exactly as a real provider would after a human signed in.
func (s *server) handleAuthorize(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	redirectURI := query.Get("redirect_uri")
	state := query.Get("state")
	nonce := query.Get("nonce")

	if redirectURI == "" {
		http.Error(w, "missing redirect_uri", http.StatusBadRequest)
		return
	}

	subject := "e2e-" + randomHex(8)
	code := randomHex(16)

	s.mu.Lock()
	s.codes[code] = pendingCode{
		subject: subject,
		name:    "E2E Test User",
		nonce:   nonce,
		expires: time.Now().Add(2 * time.Minute),
	}
	s.mu.Unlock()

	redirectURL := fmt.Sprintf("%s?code=%s&state=%s", redirectURI, code, state)
	http.Redirect(w, r, redirectURL, http.StatusFound)
}

func (s *server) handleToken(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form body", http.StatusBadRequest)
		return
	}

	code := r.PostFormValue("code")
	clientID := r.PostFormValue("client_id")
	if clientID == "" {
		if basicClientID, _, ok := r.BasicAuth(); ok {
			clientID = basicClientID
		}
	}

	s.mu.Lock()
	pending, ok := s.codes[code]
	if ok {
		delete(s.codes, code)
	}
	s.mu.Unlock()

	if !ok || time.Now().After(pending.expires) {
		http.Error(w, "invalid or expired code", http.StatusBadRequest)
		return
	}

	idToken, err := s.issueIDToken(pending.subject, pending.name, clientID, pending.nonce, time.Hour)
	if err != nil {
		http.Error(w, "issue id token: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"access_token": "mock-access-token-" + randomHex(8),
		"token_type":   "Bearer",
		"expires_in":   3600,
		"id_token":     idToken,
	})
}

func (s *server) issueIDToken(subject, name, audience, nonce string, ttl time.Duration) (string, error) {
	signer, err := jose.NewSigner(
		jose.SigningKey{Algorithm: jose.RS256, Key: s.key},
		(&jose.SignerOptions{}).WithType("JWT").WithHeader("kid", keyID),
	)
	if err != nil {
		return "", fmt.Errorf("create signer: %w", err)
	}

	now := time.Now()
	claims := jwt.Claims{
		Issuer:   s.baseURL,
		Subject:  subject,
		Audience: jwt.Audience{audience},
		Expiry:   jwt.NewNumericDate(now.Add(ttl)),
		IssuedAt: jwt.NewNumericDate(now),
	}
	extra := map[string]any{"name": name}
	if nonce != "" {
		extra["nonce"] = nonce
	}

	return jwt.Signed(signer).Claims(claims).Claims(extra).Serialize()
}

func randomHex(n int) string {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		panic(err)
	}
	return hex.EncodeToString(buf)
}
