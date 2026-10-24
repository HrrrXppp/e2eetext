package handler

import (
	"encoding/xml"
	"net"
	"net/http"
	"strings"

	"github.com/ekhrunov/messenger/server/internal/config"
)

// SEOHandler writes /sitemap.xml and /robots.txt for the host that requested
// them. Hosts and paths come from config.json (config.SitemapConfig). The
// listener forwards those paths to the server; ads.txt stays a static client
// file because its contents do not depend on the host.
type SEOHandler struct {
	hosts     map[string]struct{}
	canonical string
	paths     []string
}

func NewSEOHandler(sitemap config.SitemapConfig) *SEOHandler {
	sitemap = sitemap.WithDefaults()
	hosts := make(map[string]struct{}, len(sitemap.Hosts))
	for _, host := range sitemap.Hosts {
		hosts[host] = struct{}{}
	}
	return &SEOHandler{
		hosts:     hosts,
		canonical: sitemap.CanonicalHost(),
		paths:     append([]string(nil), sitemap.Paths...),
	}
}

func (h *SEOHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	host := h.publicSitemapHost(r.Host)
	w.Header().Set("Vary", "Host")
	switch r.URL.Path {
	case "/sitemap.xml":
		w.Header().Set("Content-Type", "application/xml; charset=utf-8")
		_, _ = w.Write([]byte(h.sitemapXML(host)))
	case "/robots.txt":
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte(h.robotsTXT(host)))
	default:
		http.NotFound(w, r)
	}
}

// publicSitemapHost returns the browser host when it is one of the configured
// names. It uses the Host header the load balancer forwarded, not
// X-Forwarded-Host, so the URLs match the sitemap location the crawler fetched.
func (h *SEOHandler) publicSitemapHost(requestHost string) string {
	host := requestHost
	if parsed, _, err := net.SplitHostPort(host); err == nil {
		host = parsed
	}
	host = strings.TrimSuffix(strings.ToLower(strings.TrimSpace(host)), ".")
	if _, ok := h.hosts[host]; ok {
		return host
	}
	return h.canonical
}

func (h *SEOHandler) sitemapXML(host string) string {
	var b strings.Builder
	b.WriteString(xml.Header)
	b.WriteString("<urlset xmlns=\"http://www.sitemaps.org/schemas/sitemap/0.9\">\n")
	for _, path := range h.paths {
		b.WriteString("  <url><loc>")
		_ = xml.EscapeText(&b, []byte("https://"+host+path))
		b.WriteString("</loc></url>\n")
	}
	b.WriteString("</urlset>\n")
	return b.String()
}

func (h *SEOHandler) robotsTXT(host string) string {
	return "User-agent: *\n" +
		"Allow: /\n" +
		"\n" +
		"Disallow: /api/\n" +
		"Disallow: /oauth/\n" +
		"Disallow: /health/\n" +
		"\n" +
		"Sitemap: https://" + host + "/sitemap.xml\n"
}
