// Command mockoidc is a throwaway, spec-compliant-enough OIDC provider used
// only by the client's Playwright E2E suite (client/e2e/). It plays the role
// of a real identity provider for a browser-driven OAuth authorization-code
// flow: discovery + JWKS + an auto-approving /authorize endpoint (no login
// form — it mints a fresh random test identity on every hit) + /token. This
// lets the real client and server exercise the exact same sign-in code path
// they use with Google in production, without any Google credentials.
//
// It also serves a second, "Apple-like" issuer under the /apple prefix
// (/apple/.well-known/openid-configuration, /apple/authorize, /apple/token)
// that emulates Apple's OAuth peculiarities so the client/e2e/apple-sign-in
// spec exercises the real response_mode=form_post + private_key_jwt code
// paths, not just the default Google-shaped flow:
//   - /apple/authorize responds with an auto-submitting HTML form POST to
//     the redirect_uri (response_mode=form_post) instead of a 302 redirect,
//     and includes a one-time "user" JSON field (name + email) only on the
//     FIRST authorization for a given mock subject — tracked via a session
//     cookie scoped to this mock, exactly like a real IdP remembering a
//     signed-in browser. The minted ID token never carries a name claim.
//   - /apple/token requires the caller to authenticate with a real,
//     validly-signed ES256 JWT (RFC 7523 private_key_jwt) as client_secret,
//     verified against an EC keypair this mock itself generates at startup
//     and exposes (private half only) via /apple/test-client-key for the
//     E2E harness to feed into the server's config.
//
// It is never built into a production image and is not started by
// docker-compose.yml.
package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"flag"
	"fmt"
	"html"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	jose "github.com/go-jose/go-jose/v4"
	"github.com/go-jose/go-jose/v4/jwt"
)

const keyID = "mockoidc-e2e-key"

// appleSessionCookie is set by /apple/authorize on the first hit and read
// back on subsequent hits so the mock recognizes "the same signed-in
// person" across separate authorize requests in the same browser — exactly
// how a real IdP's session works, and what lets the E2E suite reuse the same
// mock subject for a second sign-in.
const appleSessionCookie = "mockoidc_apple_session"

type pendingCode struct {
	subject string
	name    string
	nonce   string
	expires time.Time
}

type server struct {
	key     *rsa.PrivateKey
	baseURL string

	mu                sync.Mutex
	codes             map[string]pendingCode
	appleSeenSubjects map[string]bool

	// appleClientKey is a throwaway EC keypair standing in for a real
	// client's registered private_key_jwt key: the server (acting as OAuth
	// client) signs with the private half (learned via
	// /apple/test-client-key); this mock verifies with the public half.
	appleClientKey *ecdsa.PrivateKey
	// appleIssuer is a mock "Team ID"-shaped value the server is expected
	// to echo back as the `iss` claim of its private_key_jwt assertions.
	appleIssuer string
}

func main() {
	addr := flag.String("addr", ":9998", "address to listen on")
	baseURL := flag.String("base-url", "http://127.0.0.1:9998", "externally reachable base URL for this server")
	flag.Parse()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		log.Fatalf("generate signing key: %v", err)
	}

	appleClientKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		log.Fatalf("generate apple client key: %v", err)
	}

	s := &server{
		key:               key,
		baseURL:           strings.TrimRight(*baseURL, "/"),
		codes:             make(map[string]pendingCode),
		appleSeenSubjects: make(map[string]bool),
		appleClientKey:    appleClientKey,
		appleIssuer:       "mock-apple-team-" + randomHex(6),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /.well-known/openid-configuration", s.handleDiscovery)
	mux.HandleFunc("GET /jwks", s.handleJWKS)
	mux.HandleFunc("GET /authorize", s.handleAuthorize)
	mux.HandleFunc("POST /token", s.handleToken)

	mux.HandleFunc("GET /apple/.well-known/openid-configuration", s.handleAppleDiscovery)
	mux.HandleFunc("GET /apple/authorize", s.handleAppleAuthorize)
	mux.HandleFunc("POST /apple/token", s.handleAppleToken)
	mux.HandleFunc("GET /apple/test-client-key", s.handleAppleTestClientKey)

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

	idToken, err := s.issueIDToken(s.baseURL, pending.subject, pending.name, clientID, pending.nonce, time.Hour)
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

// handleAppleDiscovery serves discovery for the /apple issuer, distinct from
// the default one above so this mock can present as two different OIDC
// providers to the server (see providerByIssuer in the auth service).
func (s *server) handleAppleDiscovery(w http.ResponseWriter, r *http.Request) {
	issuer := s.appleIssuerURL()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"issuer":                                issuer,
		"authorization_endpoint":                issuer + "/authorize",
		"token_endpoint":                        issuer + "/token",
		"jwks_uri":                              s.baseURL + "/jwks",
		"response_types_supported":              []string{"code"},
		"response_modes_supported":              []string{"form_post"},
		"subject_types_supported":               []string{"public"},
		"id_token_signing_alg_values_supported": []string{"RS256"},
		"scopes_supported":                      []string{"openid", "name", "email"},
		"token_endpoint_auth_methods_supported": []string{"client_secret_post"},
	})
}

func (s *server) appleIssuerURL() string {
	return s.baseURL + "/apple"
}

// handleAppleAuthorize emulates Apple's authorize endpoint: a form_post
// response instead of a redirect, and a one-time "user" name/email field
// sent only the first time a given mock subject authorizes.
func (s *server) handleAppleAuthorize(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	redirectURI := query.Get("redirect_uri")
	state := query.Get("state")
	nonce := query.Get("nonce")

	if redirectURI == "" {
		http.Error(w, "missing redirect_uri", http.StatusBadRequest)
		return
	}

	subject := s.appleSubjectForSession(w, r)
	code := randomHex(16)

	s.mu.Lock()
	s.codes[code] = pendingCode{
		subject: subject,
		nonce:   nonce,
		expires: time.Now().Add(2 * time.Minute),
	}
	firstTime := !s.appleSeenSubjects[subject]
	s.appleSeenSubjects[subject] = true
	s.mu.Unlock()

	var userJSON string
	if firstTime {
		payload, err := json.Marshal(map[string]any{
			"name": map[string]string{
				"firstName": "Ada",
				"lastName":  "AppleTester",
			},
			"email": subject + "@example-apple-e2e.test",
		})
		if err != nil {
			http.Error(w, "build user payload: "+err.Error(), http.StatusInternalServerError)
			return
		}
		userJSON = string(payload)
	}

	writeAppleFormPost(w, redirectURI, code, state, userJSON)
}

// appleSubjectForSession returns the subject already associated with this
// browser (via appleSessionCookie) if any, else mints a fresh one and sets
// the cookie — mirroring a real IdP remembering a signed-in session.
func (s *server) appleSubjectForSession(w http.ResponseWriter, r *http.Request) string {
	if cookie, err := r.Cookie(appleSessionCookie); err == nil && cookie.Value != "" {
		return cookie.Value
	}

	subject := "e2e-apple-" + randomHex(8)
	http.SetCookie(w, &http.Cookie{
		Name:  appleSessionCookie,
		Value: subject,
		Path:  "/",
	})
	return subject
}

func writeAppleFormPost(w http.ResponseWriter, redirectURI, code, state, userJSON string) {
	var fields strings.Builder
	fmt.Fprintf(&fields, `<input type="hidden" name="code" value="%s">`, html.EscapeString(code))
	fmt.Fprintf(&fields, `<input type="hidden" name="state" value="%s">`, html.EscapeString(state))
	if userJSON != "" {
		fmt.Fprintf(&fields, `<input type="hidden" name="user" value="%s">`, html.EscapeString(userJSON))
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, `<!DOCTYPE html>
<html>
<body onload="document.forms[0].submit()">
<form method="POST" action="%s">%s</form>
</body>
</html>`, html.EscapeString(redirectURI), fields.String())
}

func (s *server) handleAppleToken(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form body", http.StatusBadRequest)
		return
	}

	code := r.PostFormValue("code")
	clientID := r.PostFormValue("client_id")
	clientSecret := r.PostFormValue("client_secret")
	if clientID == "" || clientSecret == "" {
		if basicID, basicSecret, ok := r.BasicAuth(); ok {
			if clientID == "" {
				clientID = basicID
			}
			if clientSecret == "" {
				clientSecret = basicSecret
			}
		}
	}

	if err := s.verifyPrivateKeyJWTClientSecret(clientSecret, clientID); err != nil {
		http.Error(w, "invalid client_secret: "+err.Error(), http.StatusUnauthorized)
		return
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

	// Apple's real ID token never carries a name claim at all — the name is
	// only ever delivered via the one-time "user" form field above.
	idToken, err := s.issueIDToken(s.appleIssuerURL(), pending.subject, "", clientID, pending.nonce, time.Hour)
	if err != nil {
		http.Error(w, "issue id token: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"access_token": "mock-apple-access-token-" + randomHex(8),
		"token_type":   "Bearer",
		"expires_in":   3600,
		"id_token":     idToken,
	})
}

// verifyPrivateKeyJWTClientSecret performs a real ES256 signature
// verification of the client_secret against this mock's own known public
// key for the test client (rejecting anything that doesn't verify), then
// checks the iss/sub/aud claims match what's expected of a private_key_jwt
// assertion for this mock: sub is the client_id the caller authenticated
// as, aud is this mock's own /apple issuer URL, and iss is the value this
// mock generated at startup and exposed via /apple/test-client-key.
func (s *server) verifyPrivateKeyJWTClientSecret(rawJWT, clientID string) error {
	if rawJWT == "" {
		return fmt.Errorf("missing client_secret")
	}

	parsed, err := jwt.ParseSigned(rawJWT, []jose.SignatureAlgorithm{jose.ES256})
	if err != nil {
		return fmt.Errorf("parse jwt: %w", err)
	}

	var claims jwt.Claims
	if err := parsed.Claims(&s.appleClientKey.PublicKey, &claims); err != nil {
		return fmt.Errorf("verify signature: %w", err)
	}

	expected := jwt.Expected{
		Issuer:      s.appleIssuer,
		Subject:     clientID,
		AnyAudience: jwt.Audience{s.appleIssuerURL()},
		Time:        time.Now(),
	}
	if err := claims.Validate(expected); err != nil {
		return fmt.Errorf("validate claims: %w", err)
	}

	return nil
}

// handleAppleTestClientKey exposes the mock-generated EC private key (and
// the issuer value the mock expects back) so the E2E harness can feed them
// into the server's config.json, exercising the real
// mintPrivateKeyJWT/oauthConfig code path against this mock rather than a
// hardcoded, checked-in key. Test-only: never called by production code.
func (s *server) handleAppleTestClientKey(w http.ResponseWriter, r *http.Request) {
	der, err := x509.MarshalECPrivateKey(s.appleClientKey)
	if err != nil {
		http.Error(w, "marshal key: "+err.Error(), http.StatusInternalServerError)
		return
	}
	pemBytes := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: der})

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"private_key_pem": string(pemBytes),
		"issuer":          s.appleIssuer,
		"algorithm":       "ES256",
	})
}

func (s *server) issueIDToken(issuer, subject, name, audience, nonce string, ttl time.Duration) (string, error) {
	signer, err := jose.NewSigner(
		jose.SigningKey{Algorithm: jose.RS256, Key: s.key},
		(&jose.SignerOptions{}).WithType("JWT").WithHeader("kid", keyID),
	)
	if err != nil {
		return "", fmt.Errorf("create signer: %w", err)
	}

	now := time.Now()
	claims := jwt.Claims{
		Issuer:   issuer,
		Subject:  subject,
		Audience: jwt.Audience{audience},
		Expiry:   jwt.NewNumericDate(now.Add(ttl)),
		IssuedAt: jwt.NewNumericDate(now),
	}
	extra := map[string]any{}
	if name != "" {
		extra["name"] = name
	}
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
