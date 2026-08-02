package service

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"testing"

	jose "github.com/go-jose/go-jose/v4"
	"github.com/go-jose/go-jose/v4/jwt"
	"golang.org/x/oauth2"

	"github.com/ekhrunov/messenger/server/internal/config"
	"github.com/ekhrunov/messenger/server/internal/domain"
)

// newTestOIDCDiscoveryServer serves just enough of an OIDC discovery
// document for oidc.NewProvider to succeed, so oauthConfig()'s real
// discovery step runs against real HTTP instead of being bypassed.
func newTestOIDCDiscoveryServer(t *testing.T) *httptest.Server {
	t.Helper()

	var srv *httptest.Server
	mux := http.NewServeMux()
	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"issuer":                 srv.URL,
			"authorization_endpoint": srv.URL + "/authorize",
			"token_endpoint":         srv.URL + "/token",
			"jwks_uri":               srv.URL + "/jwks",
		})
	})
	srv = httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

// TestAuthService_OAuthConfig is parametrized by domain.OIDCProvider, one
// case per client_secret_strategy, so both providers this server supports
// (Google/static, Apple/private_key_jwt) are exercised the same way instead
// of drifting apart as separate ad hoc tests.
func TestAuthService_OAuthConfig(t *testing.T) {
	pemKey, key := generateTestECKeyPEM(t)

	tests := []struct {
		name       string
		provider   domain.OIDCProvider
		credential config.OAuthCredential
		wantScopes []string
		verify     func(t *testing.T, srv *httptest.Server, oauthCfg *oauth2.Config)
	}{
		{
			name: "google static strategy uses configured secret and DB scopes",
			provider: domain.OIDCProvider{
				Name:                 "Google",
				Scopes:               []string{"openid", "profile"},
				ClientSecretStrategy: "static",
			},
			credential: config.OAuthCredential{ClientID: "gid", ClientSecret: "gsecret"},
			wantScopes: []string{"openid", "profile"},
			verify: func(t *testing.T, _ *httptest.Server, oauthCfg *oauth2.Config) {
				t.Helper()
				if oauthCfg.ClientSecret != "gsecret" {
					t.Errorf("ClientSecret = %q, want gsecret (static strategy)", oauthCfg.ClientSecret)
				}
			},
		},
		{
			name: "apple private_key_jwt strategy mints signed secret",
			provider: domain.OIDCProvider{
				Name:                 "Apple",
				Scopes:               []string{"openid", "name"},
				ResponseMode:         "form_post",
				ClientSecretStrategy: "private_key_jwt",
			},
			credential: config.OAuthCredential{
				ClientID:                "services-id",
				PrivateKeyJWTIssuer:     "team-id",
				PrivateKeyJWTKeyID:      "key-id",
				PrivateKeyJWTPrivateKey: pemKey,
			},
			wantScopes: []string{"openid", "name"},
			verify: func(t *testing.T, srv *httptest.Server, oauthCfg *oauth2.Config) {
				t.Helper()
				parsed, err := jwt.ParseSigned(oauthCfg.ClientSecret, []jose.SignatureAlgorithm{jose.ES256})
				if err != nil {
					t.Fatalf("ClientSecret is not a valid signed JWT: %v", err)
				}

				var claims jwt.Claims
				if err := parsed.Claims(&key.PublicKey, &claims); err != nil {
					t.Fatalf("verify minted client secret signature: %v", err)
				}
				if claims.Subject != "services-id" {
					t.Errorf("sub = %q, want services-id (credential.ClientID)", claims.Subject)
				}
				if claims.Issuer != "team-id" {
					t.Errorf("iss = %q, want team-id", claims.Issuer)
				}
				if len(claims.Audience) != 1 || claims.Audience[0] != srv.URL {
					t.Errorf("aud = %v, want [%s] (provider.Link)", claims.Audience, srv.URL)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := newTestOIDCDiscoveryServer(t)

			provider := tt.provider
			provider.Link = srv.URL

			cfg := config.Config{
				OAuthCredentials: map[string]config.OAuthCredential{
					ProviderSlug(provider.Name): tt.credential,
				},
			}
			svc := NewAuthService(cfg, &mockOIDCProviderRepository{})

			oauthCfg, _, err := svc.oauthConfig(context.Background(), provider, nil)
			if err != nil {
				t.Fatalf("oauthConfig() error = %v", err)
			}

			if !reflect.DeepEqual(oauthCfg.Scopes, tt.wantScopes) {
				t.Errorf("Scopes = %v, want %v", oauthCfg.Scopes, tt.wantScopes)
			}

			tt.verify(t, srv, oauthCfg)
		})
	}
}

// TestAuthService_BeginLogin_ResponseMode is parametrized by
// domain.OIDCProvider and covers both providers configured today: Apple
// (response_mode set in the DB, so it must reach the authorize URL) and
// Google (no response_mode configured, so it must be omitted).
func TestAuthService_BeginLogin_ResponseMode(t *testing.T) {
	pemKey, _ := generateTestECKeyPEM(t)

	tests := []struct {
		name             string
		slug             string
		provider         domain.OIDCProvider
		credentials      map[string]config.OAuthCredential
		wantResponseMode string
	}{
		{
			name: "apple sets response_mode when configured",
			slug: "apple",
			provider: domain.OIDCProvider{
				Name:                 "Apple",
				Scopes:               []string{"openid", "name"},
				ResponseMode:         "form_post",
				ClientSecretStrategy: "private_key_jwt",
			},
			credentials: map[string]config.OAuthCredential{
				"apple": {
					ClientID:                "services-id",
					PrivateKeyJWTIssuer:     "team-id",
					PrivateKeyJWTKeyID:      "key-id",
					PrivateKeyJWTPrivateKey: pemKey,
				},
			},
			wantResponseMode: "form_post",
		},
		{
			name: "google omits response_mode when not configured",
			slug: "google",
			provider: domain.OIDCProvider{
				Name:                 "Google",
				Scopes:               []string{"openid", "profile"},
				ClientSecretStrategy: "static",
			},
			credentials: map[string]config.OAuthCredential{
				"google": {ClientID: "gid", ClientSecret: "gsecret"},
			},
			wantResponseMode: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := newTestOIDCDiscoveryServer(t)

			provider := tt.provider
			provider.Link = srv.URL

			cfg := config.Config{OAuthCredentials: tt.credentials}
			repo := &mockOIDCProviderRepository{providers: []domain.OIDCProvider{provider}}
			svc := NewAuthService(cfg, repo)

			req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/"+tt.slug+"/login", nil)
			rec := httptest.NewRecorder()

			if err := svc.BeginLogin(rec, req, tt.slug); err != nil {
				t.Fatalf("BeginLogin() error = %v", err)
			}

			location, err := url.Parse(rec.Header().Get("Location"))
			if err != nil {
				t.Fatalf("parse Location header: %v", err)
			}
			if got := location.Query().Get("response_mode"); got != tt.wantResponseMode {
				t.Errorf("response_mode = %q, want %q", got, tt.wantResponseMode)
			}
		})
	}
}

// TestAuthService_CompleteLogin_RejectsPOSTForNonFormPostProvider covers the
// restriction CompleteLogin enforces itself: the router registers both GET
// and POST on every provider's callback path (it can't know a given
// provider's response_mode, stored in the DB, until the path is parsed), so
// only a provider actually configured for response_mode=form_post (Apple)
// may legitimately POST its callback.
func TestAuthService_CompleteLogin_RejectsPOSTForNonFormPostProvider(t *testing.T) {
	srv := newTestOIDCDiscoveryServer(t)

	cfg := config.Config{
		OAuthCredentials: map[string]config.OAuthCredential{
			"google": {ClientID: "gid", ClientSecret: "gsecret"},
		},
	}
	repo := &mockOIDCProviderRepository{providers: []domain.OIDCProvider{
		{Name: "Google", Link: srv.URL, Scopes: []string{"openid", "profile"}, ClientSecretStrategy: "static"},
	}}
	svc := NewAuthService(cfg, repo)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/google/callback", nil)
	req.AddCookie(&http.Cookie{Name: oauthStateCookie, Value: "state123"})
	req.AddCookie(&http.Cookie{Name: oauthNonceCookie, Value: "nonce123"})
	rec := httptest.NewRecorder()

	err := svc.CompleteLogin(rec, req, "google", "code123", "state123", "")
	if err == nil {
		t.Fatal("CompleteLogin() error = nil, want error rejecting POST for a non-form_post provider")
	}
	if !strings.Contains(err.Error(), "does not accept a POST callback") {
		t.Errorf("CompleteLogin() error = %v, want mention of rejecting the POST callback", err)
	}
}

func TestAuthService_BeginLogin_OmitsAccessTypeAndPromptForPrivateKeyJWT(t *testing.T) {
	srv := newTestOIDCDiscoveryServer(t)
	pemKey, _ := generateTestECKeyPEM(t)

	cfg := config.Config{
		OAuthCredentials: map[string]config.OAuthCredential{
			"apple": {
				ClientID:                "services-id",
				PrivateKeyJWTIssuer:     "team-id",
				PrivateKeyJWTKeyID:      "key-id",
				PrivateKeyJWTPrivateKey: pemKey,
			},
		},
	}
	repo := &mockOIDCProviderRepository{providers: []domain.OIDCProvider{
		{
			Name:                 "Apple",
			Link:                 srv.URL,
			Scopes:               []string{"openid", "name"},
			ResponseMode:         "form_post",
			ClientSecretStrategy: "private_key_jwt",
		},
	}}
	svc := NewAuthService(cfg, repo)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/apple/login", nil)
	rec := httptest.NewRecorder()

	if err := svc.BeginLogin(rec, req, "apple"); err != nil {
		t.Fatalf("BeginLogin() error = %v", err)
	}

	location, err := url.Parse(rec.Header().Get("Location"))
	if err != nil {
		t.Fatalf("parse Location header: %v", err)
	}
	if got := location.Query().Get("access_type"); got != "" {
		t.Errorf("access_type = %q, want empty for private_key_jwt provider", got)
	}
	if got := location.Query().Get("prompt"); got != "" {
		t.Errorf("prompt = %q, want empty for private_key_jwt provider", got)
	}
}

func TestAuthService_BeginLogin_SetsAccessTypeAndPromptForStaticStrategy(t *testing.T) {
	srv := newTestOIDCDiscoveryServer(t)

	cfg := config.Config{
		OAuthCredentials: map[string]config.OAuthCredential{
			"google": {ClientID: "gid", ClientSecret: "gsecret"},
		},
	}
	repo := &mockOIDCProviderRepository{providers: []domain.OIDCProvider{
		{
			Name:                 "Google",
			Link:                 srv.URL,
			Scopes:               []string{"openid", "profile"},
			ClientSecretStrategy: "static",
		},
	}}
	svc := NewAuthService(cfg, repo)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/google/login", nil)
	rec := httptest.NewRecorder()

	if err := svc.BeginLogin(rec, req, "google"); err != nil {
		t.Fatalf("BeginLogin() error = %v", err)
	}

	location, err := url.Parse(rec.Header().Get("Location"))
	if err != nil {
		t.Fatalf("parse Location header: %v", err)
	}
	if got := location.Query().Get("access_type"); got != "offline" {
		t.Errorf("access_type = %q, want offline for static-strategy provider", got)
	}
	if got := location.Query().Get("prompt"); got != "consent" {
		t.Errorf("prompt = %q, want consent for static-strategy provider", got)
	}
}

func TestAuthService_BeginLogin_SetsSameSiteNoneSecureCookiesOverHTTPS(t *testing.T) {
	srv := newTestOIDCDiscoveryServer(t)

	cfg := config.Config{
		OAuthCredentials: map[string]config.OAuthCredential{
			"google": {ClientID: "gid", ClientSecret: "gsecret"},
		},
	}
	repo := &mockOIDCProviderRepository{providers: []domain.OIDCProvider{
		{Name: "Google", Link: srv.URL, Scopes: []string{"openid", "profile"}, ClientSecretStrategy: "static"},
	}}
	svc := NewAuthService(cfg, repo)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/google/login", nil)
	req.Header.Set("X-Forwarded-Proto", "https")
	rec := httptest.NewRecorder()

	if err := svc.BeginLogin(rec, req, "google"); err != nil {
		t.Fatalf("BeginLogin() error = %v", err)
	}

	cookies := rec.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("expected oauth_state/oauth_nonce cookies to be set")
	}
	for _, c := range cookies {
		if c.SameSite != http.SameSiteNoneMode {
			t.Errorf("cookie %s SameSite = %v, want SameSiteNoneMode over HTTPS", c.Name, c.SameSite)
		}
		if !c.Secure {
			t.Errorf("cookie %s Secure = false, want true over HTTPS (required for SameSite=None)", c.Name)
		}
	}
}

func TestAuthService_BeginLogin_SetsSameSiteLaxCookiesOverPlainHTTP(t *testing.T) {
	srv := newTestOIDCDiscoveryServer(t)

	cfg := config.Config{
		OAuthCredentials: map[string]config.OAuthCredential{
			"google": {ClientID: "gid", ClientSecret: "gsecret"},
		},
	}
	repo := &mockOIDCProviderRepository{providers: []domain.OIDCProvider{
		{Name: "Google", Link: srv.URL, Scopes: []string{"openid", "profile"}, ClientSecretStrategy: "static"},
	}}
	svc := NewAuthService(cfg, repo)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/google/login", nil)
	rec := httptest.NewRecorder()

	if err := svc.BeginLogin(rec, req, "google"); err != nil {
		t.Fatalf("BeginLogin() error = %v", err)
	}

	cookies := rec.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("expected oauth_state/oauth_nonce cookies to be set")
	}
	for _, c := range cookies {
		if c.SameSite != http.SameSiteLaxMode {
			t.Errorf("cookie %s SameSite = %v, want SameSiteLaxMode over plain HTTP", c.Name, c.SameSite)
		}
		if c.Secure {
			t.Errorf("cookie %s Secure = true, want false over plain HTTP", c.Name)
		}
	}
}

func TestParseOneTimeUserName(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want string
	}{
		{
			name: "full name",
			raw:  `{"name":{"firstName":"Ada","lastName":"Appleseed"},"email":"ada@example.com"}`,
			want: "Ada Appleseed",
		},
		{
			name: "missing last name",
			raw:  `{"name":{"firstName":"Ada"},"email":"ada@example.com"}`,
			want: "Ada",
		},
		{name: "empty string", raw: "", want: ""},
		{name: "no name field (repeat sign-in)", raw: `{"email":"ada@example.com"}`, want: ""},
		{name: "invalid json", raw: "not json", want: ""},
		{name: "whitespace only names", raw: `{"name":{"firstName":"  ","lastName":"  "}}`, want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := parseOneTimeUserName(tt.raw); got != tt.want {
				t.Errorf("parseOneTimeUserName(%q) = %q, want %q", tt.raw, got, tt.want)
			}
		})
	}
}
