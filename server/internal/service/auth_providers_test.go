package service

import (
	"context"
	"encoding/base64"
	"errors"
	"reflect"
	"testing"

	"github.com/ekhrunov/messenger/server/internal/config"
	"github.com/ekhrunov/messenger/server/internal/domain"
)

type mockOIDCProviderRepository struct {
	providers []domain.OIDCProvider
	err       error
}

func (m *mockOIDCProviderRepository) List(_ context.Context) ([]domain.OIDCProvider, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.providers, nil
}

func (m *mockOIDCProviderRepository) GetByName(_ context.Context, name string) (domain.OIDCProvider, error) {
	for _, provider := range m.providers {
		if ProviderSlug(provider.Name) == ProviderSlug(name) {
			return provider, nil
		}
	}
	return domain.OIDCProvider{}, errors.New("oidc provider not found")
}

func TestAuthService_ListProviders(t *testing.T) {
	picture := []byte{0x89, 0x50}
	repo := &mockOIDCProviderRepository{
		providers: []domain.OIDCProvider{
			{
				ID:   "11111111-1111-1111-1111-111111111111",
				Name: "Google",
				Link: "https://accounts.google.com",
			},
			{
				ID:      "22222222-2222-2222-2222-222222222222",
				Name:    "GitHub",
				Link:    "https://token.actions.githubusercontent.com",
				Picture: picture,
			},
		},
	}

	svc := NewAuthService(config.Config{}, repo)

	views, err := svc.ListProviders(context.Background())
	if err != nil {
		t.Fatalf("ListProviders() error = %v", err)
	}

	if len(views) != 2 {
		t.Fatalf("ListProviders() len = %d, want 2", len(views))
	}

	if views[0].Slug != "google" {
		t.Errorf("views[0].Slug = %q, want google", views[0].Slug)
	}
	if views[0].Picture != "" {
		t.Errorf("views[0].Picture = %q, want empty", views[0].Picture)
	}

	if views[1].Slug != "github" {
		t.Errorf("views[1].Slug = %q, want github", views[1].Slug)
	}
	if views[1].Picture != base64.StdEncoding.EncodeToString(picture) {
		t.Errorf("views[1].Picture = %q", views[1].Picture)
	}
}

func TestAuthService_ListProviders_Error(t *testing.T) {
	repo := &mockOIDCProviderRepository{err: errors.New("database unavailable")}
	svc := NewAuthService(config.Config{}, repo)

	_, err := svc.ListProviders(context.Background())
	if err == nil {
		t.Fatal("ListProviders() error = nil, want error")
	}
}

// TestAuthService_ProviderByIssuerAndAudience_DisambiguatesSharedIssuer
// covers the exact regression this repo's e2e suite hit: two provider rows
// ("OIDC" and "GoogleE2E") seeded with the same issuer link, so an
// issuer-only lookup can't tell them apart and silently resolves to
// whichever name sorts first — authenticating a legitimate "OIDC" token
// against "GoogleE2E"'s client_id and failing its audience check
// (surfacing as a 401 on every request right after a successful sign-in).
// Also exercises an "Apple" provider row sharing the same issuer, since
// disambiguation must hold for any number of providers sharing an issuer,
// not just the two the original regression happened to involve.
func TestAuthService_ProviderByIssuerAndAudience_DisambiguatesSharedIssuer(t *testing.T) {
	const sharedIssuer = "http://127.0.0.1:9998"

	repo := &mockOIDCProviderRepository{
		providers: []domain.OIDCProvider{
			// Alphabetically first, so an issuer-only lookup would always
			// return this one regardless of which provider actually issued
			// the token.
			{ID: "1", Name: "Apple", Link: sharedIssuer},
			{ID: "2", Name: "GoogleE2E", Link: sharedIssuer},
			{ID: "3", Name: "OIDC", Link: sharedIssuer},
		},
	}

	cfg := config.Config{
		OAuthCredentials: map[string]config.OAuthCredential{
			"apple":     {ClientID: "e2e-test-apple-client"},
			"googlee2e": {ClientID: "e2e-test-google-client"},
			"oidc":      {ClientID: "e2e-test-client"},
		},
	}

	svc := NewAuthService(cfg, repo)

	tests := []struct {
		name     string
		audience []string
		want     string
	}{
		{name: "token issued to OIDC", audience: []string{"e2e-test-client"}, want: "OIDC"},
		{name: "token issued to GoogleE2E", audience: []string{"e2e-test-google-client"}, want: "GoogleE2E"},
		{name: "token issued to Apple", audience: []string{"e2e-test-apple-client"}, want: "Apple"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := svc.providerByIssuerAndAudience(context.Background(), sharedIssuer, tt.audience)
			if err != nil {
				t.Fatalf("providerByIssuerAndAudience() error = %v", err)
			}
			if provider.Name != tt.want {
				t.Errorf("providerByIssuerAndAudience() resolved %q, want %q", provider.Name, tt.want)
			}
		})
	}
}

func TestAuthService_ProviderByIssuerAndAudience_UnmatchedAudienceErrors(t *testing.T) {
	const sharedIssuer = "http://127.0.0.1:9998"

	repo := &mockOIDCProviderRepository{
		providers: []domain.OIDCProvider{
			{ID: "1", Name: "GoogleE2E", Link: sharedIssuer},
			{ID: "2", Name: "OIDC", Link: sharedIssuer},
		},
	}
	cfg := config.Config{
		OAuthCredentials: map[string]config.OAuthCredential{
			"googlee2e": {ClientID: "e2e-test-google-client"},
			"oidc":      {ClientID: "e2e-test-client"},
		},
	}
	svc := NewAuthService(cfg, repo)

	if _, err := svc.providerByIssuerAndAudience(context.Background(), sharedIssuer, []string{"someone-elses-client"}); err == nil {
		t.Fatal("providerByIssuerAndAudience() error = nil, want error for unmatched audience")
	}
}

func TestAuthService_ProviderByIssuerAndAudience_SingleMatchIgnoresAudience(t *testing.T) {
	// A provider with a unique issuer (the common case) should resolve even
	// with no/empty audience info, same as the old issuer-only lookup did.
	repo := &mockOIDCProviderRepository{
		providers: []domain.OIDCProvider{
			{ID: "1", Name: "Google", Link: "https://accounts.google.com"},
		},
	}
	svc := NewAuthService(config.Config{}, repo)

	provider, err := svc.providerByIssuerAndAudience(context.Background(), "https://accounts.google.com", nil)
	if err != nil {
		t.Fatalf("providerByIssuerAndAudience() error = %v", err)
	}
	if provider.Name != "Google" {
		t.Errorf("providerByIssuerAndAudience() resolved %q, want Google", provider.Name)
	}
}

func TestAuthService_ProviderByIssuerAndAudience_NoIssuerMatchErrors(t *testing.T) {
	repo := &mockOIDCProviderRepository{
		providers: []domain.OIDCProvider{
			{ID: "1", Name: "Google", Link: "https://accounts.google.com"},
		},
	}
	svc := NewAuthService(config.Config{}, repo)

	if _, err := svc.providerByIssuerAndAudience(context.Background(), "https://unknown.example.com", nil); err == nil {
		t.Fatal("providerByIssuerAndAudience() error = nil, want error for unknown issuer")
	}
}

func TestTokenAudience(t *testing.T) {
	tests := []struct {
		name    string
		claims  string
		want    []string
		wantErr bool
	}{
		{name: "single string audience", claims: `{"aud":"client-a"}`, want: []string{"client-a"}},
		{name: "array audience", claims: `{"aud":["client-a","client-b"]}`, want: []string{"client-a", "client-b"}},
		{name: "missing audience", claims: `{}`, want: nil},
		{name: "empty string audience", claims: `{"aud":""}`, want: nil},
		{name: "invalid audience type", claims: `{"aud":42}`, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token := testJWTWithPayload(t, tt.claims)
			got, err := tokenAudience(token)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("tokenAudience(%s) error = nil, want error", tt.claims)
				}
				return
			}
			if err != nil {
				t.Fatalf("tokenAudience(%s) error = %v", tt.claims, err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("tokenAudience(%s) = %v, want %v", tt.claims, got, tt.want)
			}
		})
	}
}

// testJWTWithPayload builds an unsigned-but-well-formed JWT string whose
// claims segment is exactly payloadJSON — enough for the unverified claim
// readers (tokenIssuer/tokenAudience) under test here, which never check
// the signature themselves (the real oidc.IDTokenVerifier.Verify call
// downstream does that against the actual JWKS).
func testJWTWithPayload(t *testing.T, payloadJSON string) string {
	t.Helper()
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"none"}`))
	payload := base64.RawURLEncoding.EncodeToString([]byte(payloadJSON))
	return header + "." + payload + ".sig"
}

func TestProviderSlug(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{name: "Google", want: "google"},
		{name: "  GitHub  ", want: "github"},
	}

	for _, tt := range tests {
		if got := ProviderSlug(tt.name); got != tt.want {
			t.Errorf("ProviderSlug(%q) = %q, want %q", tt.name, got, tt.want)
		}
	}
}
