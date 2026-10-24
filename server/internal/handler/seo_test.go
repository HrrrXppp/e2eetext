package handler

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
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
			if got := publicSitemapHost(tt.host); got != tt.want {
				t.Fatalf("publicSitemapHost(%q) = %q, want %q", tt.host, got, tt.want)
			}
		})
	}
}

func TestSEOHandler_SitemapUsesRequestHost(t *testing.T) {
	h := NewSEOHandler()
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
	h := NewSEOHandler()
	req := httptest.NewRequest(http.MethodGet, "/robots.txt", nil)
	req.Host = "e2eetext.io"
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if got := rec.Body.String(); got != robotsTXT("e2eetext.io") {
		t.Fatalf("robots =\n%s\nwant\n%s", got, robotsTXT("e2eetext.io"))
	}
}

func TestCanonicalStaticSEOFilesMatchServer(t *testing.T) {
	dir := filepath.Join("..", "..", "..", "client", "public")
	sitemap, err := os.ReadFile(filepath.Join(dir, "sitemap.xml"))
	if err != nil {
		t.Fatalf("read sitemap.xml: %v", err)
	}
	if string(sitemap) != sitemapXML(canonicalPublicHost) {
		t.Fatalf("client/public/sitemap.xml is not the %s sitemap the server writes", canonicalPublicHost)
	}
	robots, err := os.ReadFile(filepath.Join(dir, "robots.txt"))
	if err != nil {
		t.Fatalf("read robots.txt: %v", err)
	}
	if string(robots) != robotsTXT(canonicalPublicHost) {
		t.Fatalf("client/public/robots.txt is not the %s robots.txt the server writes", canonicalPublicHost)
	}
}
