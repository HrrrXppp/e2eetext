package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/ekhrunov/messenger/server/internal/config"
	"github.com/ekhrunov/messenger/server/internal/domain"
	"github.com/ekhrunov/messenger/server/internal/repository"
	"github.com/ekhrunov/messenger/server/internal/requestorigin"
	"golang.org/x/oauth2"
)

const (
	oauthStateCookie  = "oauth_state"
	oauthNonceCookie  = "oauth_nonce"
	oauthCookieMaxAge = 10 * time.Minute

	clientSecretStrategyPrivateKeyJWT = "private_key_jwt"
	privateKeyJWTTTL                  = 5 * time.Minute

	responseModeFormPost = "form_post"
)

type contextKey string

const TokenUserContextKey contextKey = "tokenUser"

type TokenUser struct {
	Subject  string `json:"subject"`
	Name     string `json:"name,omitempty"`
	Provider string `json:"provider"`
}

type AuthService struct {
	cfg          config.Config
	providerRepo repository.OIDCProviderRepository
}

func NewAuthService(
	cfg config.Config,
	providerRepo repository.OIDCProviderRepository,
) *AuthService {
	return &AuthService{
		cfg:          cfg,
		providerRepo: providerRepo,
	}
}

type ProviderView struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Link    string `json:"link"`
	Slug    string `json:"slug"`
	Picture string `json:"picture,omitempty"`
}

func (s *AuthService) ListProviders(ctx context.Context) ([]ProviderView, error) {
	providers, err := s.providerRepo.List(ctx)
	if err != nil {
		return nil, err
	}

	views := make([]ProviderView, 0, len(providers))
	for _, provider := range providers {
		view := ProviderView{
			ID:   provider.ID,
			Name: provider.Name,
			Link: provider.Link,
			Slug: ProviderSlug(provider.Name),
		}
		if len(provider.Picture) > 0 {
			view.Picture = base64.StdEncoding.EncodeToString(provider.Picture)
		}
		views = append(views, view)
	}

	return views, nil
}

func (s *AuthService) BeginLogin(w http.ResponseWriter, r *http.Request, providerSlug string) error {
	provider, err := s.providerBySlug(r.Context(), providerSlug)
	if err != nil {
		return err
	}

	oauthConfig, _, err := s.oauthConfig(r.Context(), provider, r)
	if err != nil {
		return err
	}

	state, err := randomToken(32)
	if err != nil {
		return fmt.Errorf("generate oauth state: %w", err)
	}

	nonce, err := randomToken(32)
	if err != nil {
		return fmt.Errorf("generate oauth nonce: %w", err)
	}

	setTemporaryCookie(w, r, oauthStateCookie, state)
	setTemporaryCookie(w, r, oauthNonceCookie, nonce)

	authOpts := []oauth2.AuthCodeOption{
		oidc.Nonce(nonce),
	}
	if provider.ClientSecretStrategy != clientSecretStrategyPrivateKeyJWT {
		// AccessTypeOffline + prompt=consent are Google-shaped: they ask for
		// a refresh token and force the consent screen on every login.
		// Apple (and any other private_key_jwt provider) has different
		// authorize/token semantics, so keep these Google-only rather than
		// sending Google-specific params to every IdP.
		authOpts = append(authOpts,
			oauth2.AccessTypeOffline,
			oauth2.SetAuthURLParam("prompt", "consent"),
		)
	}
	if provider.ResponseMode != "" {
		authOpts = append(authOpts, oauth2.SetAuthURLParam("response_mode", provider.ResponseMode))
	}

	authURL := oauthConfig.AuthCodeURL(state, authOpts...)

	http.Redirect(w, r, authURL, http.StatusFound)
	return nil
}

func (s *AuthService) CompleteLogin(w http.ResponseWriter, r *http.Request, providerSlug, code, state, oneTimeUser string) error {
	if err := validateOAuthCookie(r, oauthStateCookie, state); err != nil {
		return err
	}

	nonce, err := readOAuthCookie(r, oauthNonceCookie)
	if err != nil {
		return err
	}

	clearTemporaryCookie(w, r, oauthStateCookie)
	clearTemporaryCookie(w, r, oauthNonceCookie)

	provider, err := s.providerBySlug(r.Context(), providerSlug)
	if err != nil {
		return err
	}

	// The router registers both GET and POST on this same callback path for
	// every provider, since it can't know a given provider's response_mode
	// (stored in the DB) until the path is parsed. Enforce the real
	// restriction here instead: only a provider configured for
	// response_mode=form_post (Apple) may POST its callback; anyone else
	// doing so isn't a real IdP callback (form_post response_mode is
	// exactly the OIDC-spec mechanism that makes a provider deliver the
	// callback via POST instead of a GET redirect).
	if r.Method == http.MethodPost && provider.ResponseMode != responseModeFormPost {
		return fmt.Errorf("provider %q does not accept a POST callback", providerSlug)
	}

	oauthConfig, verifier, err := s.oauthConfig(r.Context(), provider, r)
	if err != nil {
		return err
	}

	token, err := oauthConfig.Exchange(r.Context(), code)
	if err != nil {
		return fmt.Errorf("exchange oauth code: %w", err)
	}

	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok || rawIDToken == "" {
		return fmt.Errorf("missing id token in oauth response")
	}

	idToken, err := verifier.Verify(r.Context(), rawIDToken)
	if err != nil {
		return fmt.Errorf("verify id token: %w", err)
	}

	if idToken.Nonce != nonce {
		return fmt.Errorf("invalid oauth nonce")
	}

	redirectURL := fmt.Sprintf(
		"%s#id_token=%s&access_token=%s&provider=%s",
		requestorigin.OAuthClientCallbackURL(r),
		url.QueryEscape(rawIDToken),
		url.QueryEscape(token.AccessToken),
		url.QueryEscape(providerSlug),
	)
	if token.RefreshToken != "" {
		redirectURL += "&refresh_token=" + url.QueryEscape(token.RefreshToken)
	}
	// Some providers (Apple) hand us the user's name only once, out-of-band
	// from the ID token, on the very first authorization for a given
	// subject (its ID token never carries a name claim at all). Thread it
	// through to the client so it can relay it to ensureUserRegistered /
	// CreateFromToken the same way Google's ID-token-derived name flows
	// today.
	if name := parseOneTimeUserName(oneTimeUser); name != "" {
		redirectURL += "&name=" + url.QueryEscape(name)
	}
	http.Redirect(w, r, redirectURL, http.StatusFound)
	return nil
}

// oneTimeUserPayload mirrors the shape of Apple's one-time "user" form
// field: {"name":{"firstName":"...","lastName":"..."}}. We only request the
// "name" scope (not "email" — we don't keep the user's email address), so
// Apple's payload carries no email field to decode here.
type oneTimeUserPayload struct {
	Name *struct {
		FirstName string `json:"firstName"`
		LastName  string `json:"lastName"`
	} `json:"name"`
}

// parseOneTimeUserName extracts a display name from a provider's one-time
// "user" JSON payload, if any. Returns "" if raw is empty, invalid, or
// carries no name.
func parseOneTimeUserName(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}

	var payload oneTimeUserPayload
	if err := json.Unmarshal([]byte(raw), &payload); err != nil || payload.Name == nil {
		return ""
	}

	name := strings.TrimSpace(strings.Join(strings.Fields(
		payload.Name.FirstName+" "+payload.Name.LastName,
	), " "))
	return name
}

type RefreshTokenInput struct {
	Provider     string
	RefreshToken string
}

type RefreshTokenResult struct {
	IDToken      string
	AccessToken  string
	RefreshToken string
}

func (s *AuthService) RefreshTokens(ctx context.Context, input RefreshTokenInput) (RefreshTokenResult, error) {
	providerSlug := strings.ToLower(strings.TrimSpace(input.Provider))
	if providerSlug == "" {
		return RefreshTokenResult{}, fmt.Errorf("provider is required")
	}
	if strings.TrimSpace(input.RefreshToken) == "" {
		return RefreshTokenResult{}, fmt.Errorf("refresh_token is required")
	}

	provider, err := s.providerBySlug(ctx, providerSlug)
	if err != nil {
		return RefreshTokenResult{}, err
	}

	oauthConfig, _, err := s.oauthConfig(ctx, provider, nil)
	if err != nil {
		return RefreshTokenResult{}, err
	}

	tokenSource := oauthConfig.TokenSource(ctx, &oauth2.Token{
		RefreshToken: input.RefreshToken,
	})
	newToken, err := tokenSource.Token()
	if err != nil {
		return RefreshTokenResult{}, fmt.Errorf("refresh oauth token: %w", err)
	}

	rawIDToken, ok := newToken.Extra("id_token").(string)
	if !ok || rawIDToken == "" {
		return RefreshTokenResult{}, fmt.Errorf("missing id token in refresh response")
	}

	result := RefreshTokenResult{
		IDToken:     rawIDToken,
		AccessToken: newToken.AccessToken,
	}
	if newToken.RefreshToken != "" {
		result.RefreshToken = newToken.RefreshToken
	}

	return result, nil
}

func (s *AuthService) AuthenticateRequest(r *http.Request) (TokenUser, error) {
	rawToken, err := bearerToken(r)
	if err != nil {
		return TokenUser{}, err
	}

	return s.authenticateRawToken(r.Context(), rawToken)
}

func (s *AuthService) AuthenticateWebSocket(r *http.Request) (TokenUser, error) {
	rawToken := strings.TrimSpace(r.URL.Query().Get("token"))
	if rawToken == "" {
		var err error
		rawToken, err = bearerToken(r)
		if err != nil {
			return TokenUser{}, err
		}
	}

	return s.authenticateRawToken(r.Context(), rawToken)
}

func (s *AuthService) authenticateRawToken(ctx context.Context, rawToken string) (TokenUser, error) {
	user, err := s.VerifyIDToken(ctx, rawToken)
	if err == nil {
		return user, nil
	}

	return s.VerifyAccessToken(ctx, rawToken)
}

func (s *AuthService) VerifyAccessToken(ctx context.Context, accessToken string) (TokenUser, error) {
	accessToken = strings.TrimSpace(accessToken)
	if accessToken == "" {
		return TokenUser{}, fmt.Errorf("missing access token")
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		"https://oauth2.googleapis.com/tokeninfo?access_token="+url.QueryEscape(accessToken),
		nil,
	)
	if err != nil {
		return TokenUser{}, fmt.Errorf("build tokeninfo request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return TokenUser{}, fmt.Errorf("verify access token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return TokenUser{}, fmt.Errorf("verify access token: status %d", resp.StatusCode)
	}

	var payload struct {
		Sub   string `json:"sub"`
		Name  string `json:"name"`
		Iss   string `json:"iss"`
		Error string `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return TokenUser{}, fmt.Errorf("decode tokeninfo response: %w", err)
	}
	if payload.Error != "" || payload.Sub == "" {
		return TokenUser{}, fmt.Errorf("invalid access token")
	}

	providerSlug := "google"
	if payload.Iss != "" && !strings.Contains(payload.Iss, "google.com") {
		providerSlug = "oidc"
	}

	return TokenUser{
		Subject:  payload.Sub,
		Name:     payload.Name,
		Provider: providerSlug,
	}, nil
}

func (s *AuthService) VerifyIDToken(ctx context.Context, rawToken string) (TokenUser, error) {
	issuer, err := tokenIssuer(rawToken)
	if err != nil {
		return TokenUser{}, err
	}

	provider, err := s.providerByIssuer(ctx, issuer)
	if err != nil {
		return TokenUser{}, err
	}

	_, verifier, err := s.oauthConfig(ctx, provider, nil)
	if err != nil {
		return TokenUser{}, err
	}

	idToken, err := verifier.Verify(ctx, rawToken)
	if err != nil {
		return TokenUser{}, fmt.Errorf("verify id token: %w", err)
	}

	return tokenUserFromIDToken(idToken, provider.Name)
}

func TokenUserFromContext(ctx context.Context) (TokenUser, bool) {
	user, ok := ctx.Value(TokenUserContextKey).(TokenUser)
	return user, ok
}

func (s *AuthService) providerBySlug(ctx context.Context, slug string) (domain.OIDCProvider, error) {
	providers, err := s.providerRepo.List(ctx)
	if err != nil {
		return domain.OIDCProvider{}, err
	}

	for _, provider := range providers {
		if ProviderSlug(provider.Name) == slug {
			return provider, nil
		}
	}

	return domain.OIDCProvider{}, fmt.Errorf("oidc provider not found")
}

func (s *AuthService) providerByIssuer(ctx context.Context, issuer string) (domain.OIDCProvider, error) {
	providers, err := s.providerRepo.List(ctx)
	if err != nil {
		return domain.OIDCProvider{}, err
	}

	for _, provider := range providers {
		if strings.TrimRight(provider.Link, "/") == strings.TrimRight(issuer, "/") {
			return provider, nil
		}
	}

	return domain.OIDCProvider{}, fmt.Errorf("oidc provider not found for issuer")
}

func (s *AuthService) oauthConfig(ctx context.Context, provider domain.OIDCProvider, r *http.Request) (*oauth2.Config, *oidc.IDTokenVerifier, error) {
	slug := ProviderSlug(provider.Name)
	credential, ok := s.cfg.OAuthForProvider(slug)
	if !ok {
		return nil, nil, fmt.Errorf("%s oidc is not configured", slug)
	}

	oidcProvider, err := oidc.NewProvider(ctx, provider.Link)
	if err != nil {
		return nil, nil, fmt.Errorf("discover oidc provider: %w", err)
	}

	clientSecret := credential.ClientSecret
	if provider.ClientSecretStrategy == clientSecretStrategyPrivateKeyJWT {
		clientSecret, err = mintPrivateKeyJWT(
			credential.PrivateKeyJWTIssuer,
			credential.ClientID,
			provider.Link,
			credential.PrivateKeyJWTKeyID,
			credential.PrivateKeyJWTAlgorithm,
			credential.PrivateKeyJWTPrivateKey,
			privateKeyJWTTTL,
		)
		if err != nil {
			return nil, nil, fmt.Errorf("mint private_key_jwt client secret: %w", err)
		}
	}

	var redirectURL string
	if r != nil {
		redirectURL = requestorigin.OAuthCallbackURL(r, slug)
	}

	oauthConfig := &oauth2.Config{
		ClientID:     credential.ClientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
		Endpoint:     oidcProvider.Endpoint(),
		Scopes:       providerScopes(provider.Scopes),
	}

	verifier := oidcProvider.Verifier(&oidc.Config{ClientID: credential.ClientID})

	return oauthConfig, verifier, nil
}

// providerScopes returns the provider's DB-configured OAuth scopes, falling
// back to the historical Google-shaped default for callers that construct a
// domain.OIDCProvider without Scopes set (e.g. older rows, direct
// constructors in tests). In production every row has scopes set via the
// NOT NULL DEFAULT column, so this fallback should rarely trigger.
func providerScopes(scopes []string) []string {
	if len(scopes) == 0 {
		return []string{oidc.ScopeOpenID, "profile"}
	}
	return scopes
}

func tokenUserFromIDToken(idToken *oidc.IDToken, providerName string) (TokenUser, error) {
	var claims struct {
		Subject string `json:"sub"`
		Name    string `json:"name"`
	}
	if err := idToken.Claims(&claims); err != nil {
		return TokenUser{}, fmt.Errorf("parse id token claims: %w", err)
	}

	return TokenUser{
		Subject:  claims.Subject,
		Name:     claims.Name,
		Provider: ProviderSlug(providerName),
	}, nil
}

func tokenIssuer(rawToken string) (string, error) {
	parts := strings.Split(rawToken, ".")
	if len(parts) != 3 {
		return "", fmt.Errorf("invalid bearer token")
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return "", fmt.Errorf("decode bearer token: %w", err)
	}

	var claims struct {
		Issuer string `json:"iss"`
	}
	if err := json.Unmarshal(payload, &claims); err != nil {
		return "", fmt.Errorf("parse bearer token claims: %w", err)
	}
	if claims.Issuer == "" {
		return "", fmt.Errorf("missing token issuer")
	}

	return claims.Issuer, nil
}

func bearerToken(r *http.Request) (string, error) {
	header := r.Header.Get("Authorization")
	if header == "" {
		return "", fmt.Errorf("missing authorization header")
	}

	const prefix = "Bearer "
	if !strings.HasPrefix(header, prefix) {
		return "", fmt.Errorf("invalid authorization header")
	}

	token := strings.TrimSpace(strings.TrimPrefix(header, prefix))
	if token == "" {
		return "", fmt.Errorf("missing bearer token")
	}

	return token, nil
}

func ProviderSlug(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

func randomToken(size int) (string, error) {
	buf := make([]byte, size)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func setTemporaryCookie(w http.ResponseWriter, r *http.Request, name, value string) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     "/",
		MaxAge:   int(oauthCookieMaxAge.Seconds()),
		HttpOnly: true,
		SameSite: cookieSameSite(r),
		Secure:   requestorigin.IsHTTPS(r),
	})
}

func clearTemporaryCookie(w http.ResponseWriter, r *http.Request, name string) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: cookieSameSite(r),
		Secure:   requestorigin.IsHTTPS(r),
	})
}

// cookieSameSite selects SameSite=None for HTTPS requests so an IdP-initiated
// cross-site POST (e.g. Apple's response_mode=form_post callback, sent as a
// top-level navigation from appleid.apple.com) still carries oauth_state /
// oauth_nonce. SameSite=None is only honored by browsers alongside the
// Secure attribute, which requires HTTPS — so plain-HTTP local dev/mockoidc
// (same-origin) keeps the previous SameSite=Lax behavior.
func cookieSameSite(r *http.Request) http.SameSite {
	if requestorigin.IsHTTPS(r) {
		return http.SameSiteNoneMode
	}
	return http.SameSiteLaxMode
}

func validateOAuthCookie(r *http.Request, name, expected string) error {
	actual, err := readOAuthCookie(r, name)
	if err != nil {
		return err
	}
	if actual != expected {
		return fmt.Errorf("invalid oauth state")
	}
	return nil
}

func readOAuthCookie(r *http.Request, name string) (string, error) {
	cookie, err := r.Cookie(name)
	if err != nil {
		return "", fmt.Errorf("oauth cookie not found")
	}
	if cookie.Value == "" {
		return "", fmt.Errorf("oauth cookie not found")
	}
	return cookie.Value, nil
}
