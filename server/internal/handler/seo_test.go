package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ekhrunov/messenger/server/internal/config"
)

func TestPublicSitemapHost(t *testing.T) {
	tests := []struct {
		host string
		want string
	}{
		{host: "e2eetext.com", want: "e2eetext.com"},
		{host: "E2EETEXT.IO", want: "e2eetext.io"},
		{host: "e2eetext.net:443", want: "e2eetext.net"},
		{host: "e2eetext.org.", want: "e2eetext.org"},
		{host: "app.example.com", want: "e2eetext.com"},
		{host: "localhost:8081", want: "e2eetext.com"},
		{host: "evil.example", want: "e2eetext.com"},
	}
	for _, tt := range tests {
		t.Run(tt.host, func(t *testing.T) {
			if got := NewSEOHandler(config.DefaultSitemap()).publicSitemapHost(tt.host); got != tt.want {
				t.Fatalf("publicSitemapHost(%q) = %q, want %q", tt.host, got, tt.want)
			}
		})
	}
}

func TestSEOHandler_SitemapUsesRequestHost(t *testing.T) {
	h := NewSEOHandler(config.DefaultSitemap())
	req := httptest.NewRequest(http.MethodGet, "/sitemap.xml", nil)
	req.Host = "e2eetext.org"
	req.Header.Set("X-Forwarded-Host", "e2eetext.com")
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if got := rec.Header().Get("Content-Type"); got != "application/xml; charset=utf-8" {
		t.Fatalf("content-type = %q", got)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "<loc>https://e2eetext.org/</loc>") {
		t.Fatalf("sitemap missing org home:\n%s", body)
	}
	if strings.Contains(body, "e2eetext.com") || strings.Contains(body, "app.example.com") {
		t.Fatalf("sitemap leaked another host:\n%s", body)
	}
	for _, path := range []string{"/privacy", "/terms", "/roadmap"} {
		loc := "<loc>https://e2eetext.org" + path + "</loc>"
		if !strings.Contains(body, loc) {
			t.Fatalf("sitemap missing %s:\n%s", loc, body)
		}
	}
}

func TestSEOHandler_RobotsPointsAtSameHost(t *testing.T) {
	h := NewSEOHandler(config.DefaultSitemap())
	req := httptest.NewRequest(http.MethodGet, "/robots.txt", nil)
	req.Host = "e2eetext.io"
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if got := rec.Body.String(); got != h.robotsTXT("e2eetext.io") {
		t.Fatalf("robots =\n%s\nwant\n%s", got, h.robotsTXT("e2eetext.io"))
	}
}

func TestSEOHandler_UsesConfiguredHostsAndPaths(t *testing.T) {
	h := NewSEOHandler(config.SitemapConfig{
		Hosts: []string{"alpha.example", "beta.example"},
		Paths: []string{"/"},
	})
	if got := h.publicSitemapHost("beta.example"); got != "beta.example" {
		t.Fatalf("known host = %q", got)
	}
	if got := h.publicSitemapHost("other.example"); got != "alpha.example" {
		t.Fatalf("fallback = %q, want alpha.example", got)
	}
	body := h.sitemapXML("beta.example")
	if !strings.Contains(body, "<loc>https://beta.example/</loc>") || strings.Contains(body, "/privacy") {
		t.Fatalf("sitemap =\n%s", body)
	}
}
