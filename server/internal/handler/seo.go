package handler

import (
	"encoding/xml"
	"net"
	"net/http"
	"strings"
)

// canonicalPublicHost is the primary name. A request on any other host uses
// it, so localhost, the dig hostname, or a spoofed Host is not copied into the file.
const canonicalPublicHost = "e2eetext.com"

// publicHosts are the names on the prod certificate that all serve this app.
// A sitemap is only valid for the host that served it, so each of these gets
// its own URL list.
var publicHosts = map[string]struct{}{
	"e2eetext.com": {},
	"e2eetext.io":  {},
	"e2eetext.net": {},
	"e2eetext.org": {},
}

// sitemapPaths are the public pages crawlers should index. Keep
// client/public/sitemap.xml (the e2eetext.com copy) in sync with this list.
var sitemapPaths = []string{"/", "/privacy", "/terms", "/roadmap"}

// SEOHandler writes /sitemap.xml and /robots.txt for the host that requested
// them. The listener forwards those paths to the server; ads.txt stays a
// static client file because its contents do not depend on the host.
type SEOHandler struct{}

func NewSEOHandler() *SEOHandler {
	return &SEOHandler{}
}

func (h *SEOHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	host := publicSitemapHost(r.Host)
	w.Header().Set("Vary", "Host")
	switch r.URL.Path {
	case "/sitemap.xml":
		w.Header().Set("Content-Type", "application/xml; charset=utf-8")
		_, _ = w.Write([]byte(sitemapXML(host)))
	case "/robots.txt":
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte(robotsTXT(host)))
	default:
		http.NotFound(w, r)
	}
}

// publicSitemapHost returns the browser host when it is one of the public
// names. It uses the Host header the load balancer forwarded, not
// X-Forwarded-Host, so the URLs match the sitemap location the crawler fetched.
func publicSitemapHost(requestHost string) string {
	host := requestHost
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}
	host = strings.TrimSuffix(strings.ToLower(strings.TrimSpace(host)), ".")
	if _, ok := publicHosts[host]; ok {
		return host
	}
	return canonicalPublicHost
}

func sitemapXML(host string) string {
	var b strings.Builder
	b.WriteString(xml.Header)
	b.WriteString("<urlset xmlns=\"http://www.sitemaps.org/schemas/sitemap/0.9\">\n")
	for _, path := range sitemapPaths {
		b.WriteString("  <url><loc>")
		_ = xml.EscapeText(&b, []byte("https://"+host+path))
		b.WriteString("</loc></url>\n")
	}
	b.WriteString("</urlset>\n")
	return b.String()
}

func robotsTXT(host string) string {
	return "User-agent: *\n" +
		"Allow: /\n" +
		"\n" +
		"Disallow: /api/\n" +
		"Disallow: /oauth/\n" +
		"Disallow: /health/\n" +
		"\n" +
		"Sitemap: https://" + host + "/sitemap.xml\n"
}
