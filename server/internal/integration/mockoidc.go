//go:build integration

package integration

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"time"

	jose "github.com/go-jose/go-jose/v4"
	"github.com/go-jose/go-jose/v4/jwt"
)

// mockOIDCIssuer is a minimal, spec-compliant OIDC issuer used only by
// integration tests: it serves discovery + JWKS documents and mints signed
// ID tokens, so AuthService's real verification path (resolve provider by
// issuer, fetch discovery/JWKS, verify signature+audience+expiry) runs
// against real cryptography instead of being bypassed.
type mockOIDCIssuer struct {
	server *httptest.Server
	key    *rsa.PrivateKey
	keyID  string
}

const mockOIDCKeyID = "integration-test-key"

func newMockOIDCIssuer() (*mockOIDCIssuer, error) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("generate mock oidc signing key: %w", err)
	}

	issuer := &mockOIDCIssuer{key: key, keyID: mockOIDCKeyID}

	mux := http.NewServeMux()
	mux.HandleFunc("/.well-known/openid-configuration", issuer.handleDiscovery)
	mux.HandleFunc("/jwks", issuer.handleJWKS)

	issuer.server = httptest.NewServer(mux)
	return issuer, nil
}

func (m *mockOIDCIssuer) URL() string {
	return m.server.URL
}

func (m *mockOIDCIssuer) Close() {
	m.server.Close()
}

func (m *mockOIDCIssuer) handleDiscovery(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"issuer":                                m.URL(),
		"authorization_endpoint":                m.URL() + "/authorize",
		"token_endpoint":                        m.URL() + "/token",
		"jwks_uri":                              m.URL() + "/jwks",
		"response_types_supported":              []string{"code"},
		"subject_types_supported":                []string{"public"},
		"id_token_signing_alg_values_supported": []string{"RS256"},
	})
}

func (m *mockOIDCIssuer) handleJWKS(w http.ResponseWriter, r *http.Request) {
	jwk := jose.JSONWebKey{
		Key:       &m.key.PublicKey,
		KeyID:     m.keyID,
		Algorithm: string(jose.RS256),
		Use:       "sig",
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(jose.JSONWebKeySet{Keys: []jose.JSONWebKey{jwk}})
}

// issueIDToken mints a signed RS256 ID token for the given subject/audience,
// matching what a real OIDC provider would return after a login flow.
func (m *mockOIDCIssuer) issueIDToken(subject, name, audience string, ttl time.Duration) (string, error) {
	signer, err := jose.NewSigner(
		jose.SigningKey{Algorithm: jose.RS256, Key: m.key},
		(&jose.SignerOptions{}).WithType("JWT").WithHeader("kid", m.keyID),
	)
	if err != nil {
		return "", fmt.Errorf("create signer: %w", err)
	}

	now := time.Now()
	claims := jwt.Claims{
		Issuer:   m.URL(),
		Subject:  subject,
		Audience: jwt.Audience{audience},
		Expiry:   jwt.NewNumericDate(now.Add(ttl)),
		IssuedAt: jwt.NewNumericDate(now),
	}
	extra := map[string]any{"name": name}

	raw, err := jwt.Signed(signer).Claims(claims).Claims(extra).Serialize()
	if err != nil {
		return "", fmt.Errorf("sign id token: %w", err)
	}

	return raw, nil
}
