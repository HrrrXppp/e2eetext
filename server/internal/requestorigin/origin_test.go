package requestorigin

import (
	"net/http/httptest"
	"testing"
)

func TestClientIP(t *testing.T) {
	tests := []struct {
		name       string
		xff        string
		remoteAddr string
		want       string
	}{
		{"remote addr fallback", "", "192.0.2.10:4321", "192.0.2.10"},
		{"single forwarded", "203.0.113.7", "10.0.0.1:80", "203.0.113.7"},
		{"multi hop uses last", "198.51.100.1, 203.0.113.7", "10.0.0.1:80", "203.0.113.7"},
		{"ipv6 forwarded", "2001:db8::1", "10.0.0.1:80", "2001:db8::1"},
		{"invalid forwarded falls back", "garbage", "192.0.2.10:4321", "192.0.2.10"},
		{"remote addr without port", "", "192.0.2.10", "192.0.2.10"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			req.RemoteAddr = tt.remoteAddr
			if tt.xff != "" {
				req.Header.Set("X-Forwarded-For", tt.xff)
			}
			if got := ClientIP(req); got != tt.want {
				t.Fatalf("ClientIP() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestOAuthClientCallbackURL_FromForwardedHeaders(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/auth/google/callback", nil)
	req.Host = "server:8080"
	req.Header.Set("X-Forwarded-Proto", "https")
	req.Header.Set("X-Forwarded-Host", "dev.e2eetext.net")

	got := OAuthClientCallbackURL(req)
	if got != "https://dev.e2eetext.net/oauth/callback" {
		t.Fatalf("OAuthClientCallbackURL() = %q, want https://dev.e2eetext.net/oauth/callback", got)
	}
}

func TestOAuthCallbackURL_FromForwardedHeaders(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/auth/google/callback", nil)
	req.Host = "server:8080"
	req.Header.Set("X-Forwarded-Proto", "https")
	req.Header.Set("X-Forwarded-Host", "app.example.com")

	got := OAuthCallbackURL(req, "google")
	if got != "https://app.example.com/api/v1/auth/google/callback" {
		t.Fatalf("OAuthCallbackURL() = %q, want https://app.example.com/api/v1/auth/google/callback", got)
	}
}

func TestOAuthCallbackURL_UsesDirectAPIHost(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/auth/google/callback", nil)
	req.Host = "localhost:8080"

	got := OAuthCallbackURL(req, "google")
	if got != "http://localhost:8080/api/v1/auth/google/callback" {
		t.Fatalf("OAuthCallbackURL() = %q, want http://localhost:8080/api/v1/auth/google/callback", got)
	}
}

func TestPublicBaseURL_FromForwardedHeaders(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/auth/google/callback", nil)
	req.Host = "server:8080"
	req.Header.Set("X-Forwarded-Proto", "https")
	req.Header.Set("X-Forwarded-Host", "app.example.com")

	got := PublicBaseURL(req)
	if got != "https://app.example.com" {
		t.Fatalf("PublicBaseURL() = %q, want https://app.example.com", got)
	}
}

func TestPublicBaseURL_UsesDirectAPIHost(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/auth/google/callback", nil)
	req.Host = "localhost:8080"

	got := PublicBaseURL(req)
	if got != "http://localhost:8080" {
		t.Fatalf("PublicBaseURL() = %q, want http://localhost:8080", got)
	}
}

func TestPublicBaseURL_UsesRequestHostWithoutProxy(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.Host = "localhost:5173"

	got := PublicBaseURL(req)
	if got != "http://localhost:5173" {
		t.Fatalf("PublicBaseURL() = %q, want http://localhost:5173", got)
	}
}

func TestMatchesPublicOrigin_AllowsMatchingHostname(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/ws", nil)
	req.Host = "internal-server:8080"
	req.Header.Set("X-Forwarded-Proto", "https")
	req.Header.Set("X-Forwarded-Host", "dev.e2eetext.net:443")
	req.Header.Set("Origin", "https://dev.e2eetext.net")

	if !MatchesPublicOrigin(req, req.Header.Get("Origin")) {
		t.Fatal("MatchesPublicOrigin() = false, want true for matching hostname")
	}
}

func TestMatchesPublicOrigin_RejectsForeignOrigin(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/ws", nil)
	req.Host = "dev.e2eetext.net"
	req.Header.Set("X-Forwarded-Proto", "https")
	req.Header.Set("Origin", "https://evil.example.com")

	if MatchesPublicOrigin(req, req.Header.Get("Origin")) {
		t.Fatal("MatchesPublicOrigin() = true, want false for foreign origin")
	}
}
