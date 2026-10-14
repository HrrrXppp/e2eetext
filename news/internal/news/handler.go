// Package news implements the public, read-only news API.
package news

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
)

const (
	DefaultLimit = 10
	MaxLimit     = 50

	// CacheControl is sent on successful list responses; CloudFront caches for an hour.
	CacheControl = "public, max-age=3600, s-maxage=3600"
)

// Response is the body of GET /api/v1/news.
type Response struct {
	Items      []Item  `json:"items"`
	NextCursor *string `json:"next_cursor"`
}

// NewHandler routes GET /api/v1/news and GET /health.
func NewHandler(lister Lister, version string) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "version": version}, "no-store")
	})
	mux.HandleFunc("GET /api/v1/news", func(w http.ResponseWriter, r *http.Request) {
		list(w, r, lister)
	})
	return mux
}

func list(w http.ResponseWriter, r *http.Request, lister Lister) {
	q := r.URL.Query()

	limit := DefaultLimit
	if s := q.Get("limit"); s != "" {
		n, err := strconv.Atoi(s)
		if err != nil || n < 1 || n > MaxLimit {
			writeError(w, http.StatusBadRequest, "limit must be between 1 and "+strconv.Itoa(MaxLimit))
			return
		}
		limit = n
	}

	var after *Cursor
	if s := q.Get("before"); s != "" {
		c, err := DecodeCursor(s)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid before cursor")
			return
		}
		after = &c
	}

	items, err := lister.List(r.Context(), after, limit+1)
	if err != nil {
		log.Printf("list news: %v", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	resp := Response{Items: items}
	if len(items) > limit {
		resp.Items = items[:limit]
		last := resp.Items[limit-1]
		next := Cursor{PublishedAt: last.PublishedAt, ID: last.ID}.Encode()
		resp.NextCursor = &next
	}
	writeJSON(w, http.StatusOK, resp, CacheControl)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg}, "no-store")
}

func writeJSON(w http.ResponseWriter, status int, v any, cacheControl string) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", cacheControl)
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
