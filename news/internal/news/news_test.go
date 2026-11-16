package news

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type fakeLister struct {
	items []Item
	err   error
	after *Cursor
	limit int
}

func (f *fakeLister) List(_ context.Context, after *Cursor, limit int) ([]Item, error) {
	f.after, f.limit = after, limit
	if f.err != nil {
		return nil, f.err
	}
	if len(f.items) > limit {
		return f.items[:limit], nil
	}
	return f.items, nil
}

func mkItems(n int) []Item {
	base := time.Date(2026, 5, 31, 0, 0, 0, 0, time.UTC)
	out := make([]Item, n)
	for i := range out {
		out[i] = Item{ID: int64(n - i), Title: "t", Body: "b", PublishedAt: base.Add(time.Duration(n-i) * time.Hour)}
	}
	return out
}

func get(t *testing.T, l Lister, target string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	NewHandler(l, "test").ServeHTTP(rec, httptest.NewRequest(http.MethodGet, target, nil))
	return rec
}

func TestCursorRoundTrip(t *testing.T) {
	in := Cursor{PublishedAt: time.Date(2026, 5, 31, 1, 2, 3, 456000, time.UTC), ID: 42}
	out, err := DecodeCursor(in.Encode())
	if err != nil || !out.PublishedAt.Equal(in.PublishedAt) || out.ID != in.ID {
		t.Fatalf("got %+v, %v", out, err)
	}
}

func TestDecodeCursorInvalid(t *testing.T) {
	for _, s := range []string{"", "!!!", "YWJj", "MTo", "MTphYg"} {
		if _, err := DecodeCursor(s); !errors.Is(err, ErrInvalidCursor) {
			t.Errorf("%q: err = %v", s, err)
		}
	}
}

func TestListDefaultsAndCacheHeader(t *testing.T) {
	f := &fakeLister{items: mkItems(3)}
	rec := get(t, f, "/api/v1/news")
	if rec.Code != 200 || f.limit != DefaultLimit+1 || f.after != nil {
		t.Fatalf("code=%d limit=%d after=%v", rec.Code, f.limit, f.after)
	}
	if got := rec.Header().Get("Cache-Control"); got != CacheControl {
		t.Errorf("Cache-Control = %q", got)
	}
	var resp Response
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if len(resp.Items) != 3 || resp.NextCursor != nil {
		t.Errorf("resp = %+v", resp)
	}
}

func TestListEmptyIsArray(t *testing.T) {
	rec := get(t, &fakeLister{}, "/api/v1/news")
	var raw map[string]json.RawMessage
	_ = json.Unmarshal(rec.Body.Bytes(), &raw)
	if string(raw["next_cursor"]) != "null" {
		t.Errorf("next_cursor = %s", raw["next_cursor"])
	}
}

func TestListPagination(t *testing.T) {
	items := mkItems(5)
	f := &fakeLister{items: items}
	rec := get(t, f, "/api/v1/news?limit=2")
	var resp Response
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	if len(resp.Items) != 2 || resp.NextCursor == nil {
		t.Fatalf("resp = %+v", resp)
	}
	c, err := DecodeCursor(*resp.NextCursor)
	if err != nil || c.ID != resp.Items[1].ID || !c.PublishedAt.Equal(resp.Items[1].PublishedAt) {
		t.Fatalf("cursor = %+v, %v", c, err)
	}

	get(t, f, "/api/v1/news?limit=2&before="+*resp.NextCursor)
	if f.after == nil || f.after.ID != c.ID {
		t.Errorf("after = %+v", f.after)
	}
}

func TestListBadParams(t *testing.T) {
	for _, q := range []string{"limit=0", "limit=51", "limit=x", "before=zzz"} {
		rec := get(t, &fakeLister{}, "/api/v1/news?"+q)
		if rec.Code != http.StatusBadRequest || rec.Header().Get("Cache-Control") != "no-store" {
			t.Errorf("%s: code=%d cc=%q", q, rec.Code, rec.Header().Get("Cache-Control"))
		}
	}
}

func TestListMaxLimitAllowed(t *testing.T) {
	f := &fakeLister{}
	if rec := get(t, f, "/api/v1/news?limit=50"); rec.Code != 200 || f.limit != 51 {
		t.Errorf("code=%d limit=%d", rec.Code, f.limit)
	}
}

func TestListError(t *testing.T) {
	rec := get(t, &fakeLister{err: errors.New("boom")}, "/api/v1/news")
	if rec.Code != 500 || rec.Header().Get("Cache-Control") != "no-store" {
		t.Errorf("code=%d cc=%q", rec.Code, rec.Header().Get("Cache-Control"))
	}
}

func TestHealthAndMethods(t *testing.T) {
	if rec := get(t, &fakeLister{}, "/health"); rec.Code != 200 {
		t.Errorf("health code=%d", rec.Code)
	}
	rec := httptest.NewRecorder()
	NewHandler(&fakeLister{}, "t").ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/v1/news", nil))
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("POST code=%d", rec.Code)
	}
}
