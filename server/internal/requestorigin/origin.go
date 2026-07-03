package requestorigin

import (
	"net"
	"net/http"
	"net/url"
	"strings"
)

// PublicBaseURL returns the browser-visible origin (scheme + host) for the request.
// Behind a reverse proxy it uses X-Forwarded-Proto and X-Forwarded-Host.
func PublicBaseURL(r *http.Request) string {
	scheme := forwardedProto(r)
	host := forwardedHost(r)
	if host != "" && scheme != "" {
		return scheme + "://" + host
	}

	if r.Host != "" {
		if r.TLS != nil {
			return "https://" + r.Host
		}
		return "http://" + r.Host
	}
	return ""
}

func forwardedProto(r *http.Request) string {
	if v := r.Header.Get("X-Forwarded-Proto"); v != "" {
		return strings.ToLower(strings.TrimSpace(strings.Split(v, ",")[0]))
	}
	if r.TLS != nil {
		return "https"
	}
	return "http"
}

func forwardedHost(r *http.Request) string {
	if v := r.Header.Get("X-Forwarded-Host"); v != "" {
		return strings.TrimSpace(strings.Split(v, ",")[0])
	}
	return r.Host
}

// OAuthCallbackURL is the OIDC redirect URI registered with the provider for this request.
func OAuthCallbackURL(r *http.Request, providerSlug string) string {
	base := strings.TrimRight(PublicBaseURL(r), "/")
	slug := strings.ToLower(strings.TrimSpace(providerSlug))
	return base + "/api/v1/auth/" + slug + "/callback"
}

// OAuthClientCallbackPath is where the server sends the browser after OAuth (SPA stores tokens from the hash).
const OAuthClientCallbackPath = "/oauth/callback"

// OAuthClientCallbackURL is the post-login redirect target for the browser.
func OAuthClientCallbackURL(r *http.Request) string {
	return strings.TrimRight(PublicBaseURL(r), "/") + OAuthClientCallbackPath
}

// MatchesPublicOrigin reports whether origin is allowed for browser requests to this API.
// It accepts an exact PublicBaseURL match and falls back to hostname comparison so ALB/proxy
// host/port differences (e.g. implicit :443) do not reject WebSocket upgrades.
func MatchesPublicOrigin(r *http.Request, origin string) bool {
	origin = strings.TrimRight(strings.TrimSpace(origin), "/")
	if origin == "" {
		return true
	}

	expected := PublicBaseURL(r)
	if expected != "" && origin == expected {
		return true
	}

	originHost := hostFromOrigin(origin)
	requestHost := hostFromRequest(r)
	return originHost != "" && requestHost != "" && originHost == requestHost
}

func hostFromOrigin(origin string) string {
	parsed, err := url.Parse(origin)
	if err != nil {
		return ""
	}
	return strings.ToLower(parsed.Hostname())
}

func hostFromRequest(r *http.Request) string {
	host := forwardedHost(r)
	if h, _, err := net.SplitHostPort(host); err == nil {
		return strings.ToLower(h)
	}
	return strings.ToLower(host)
}
