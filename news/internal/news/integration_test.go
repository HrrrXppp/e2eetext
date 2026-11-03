//go:build integration

package news_test

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/ekhrunov/messenger/news/internal/database"
	"github.com/ekhrunov/messenger/news/internal/news"
)

// Requires INTEGRATION_DATABASE_URL pointing at a throwaway database.
func TestRepositoryKeysetPagination(t *testing.T) {
	url := os.Getenv("INTEGRATION_DATABASE_URL")
	if url == "" {
		t.Skip("INTEGRATION_DATABASE_URL not set")
	}
	if err := database.Migrate(url); err != nil {
		t.Fatal(err)
	}
	db, err := database.Open(url, 2)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	clear(t, db)

	ts := time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)
	for i := 0; i < 5; i++ {
		// Two items share a timestamp to exercise the id tiebreaker.
		at := ts.Add(time.Duration(i/2) * time.Hour)
		if _, err := db.Exec(`INSERT INTO news (title, body, published_at) VALUES ($1, 'b', $2)`, fmt.Sprintf("n%d", i), at); err != nil {
			t.Fatal(err)
		}
	}

	repo := news.NewRepository(db)
	var got []string
	var after *news.Cursor
	for {
		items, err := repo.List(context.Background(), after, 3)
		if err != nil {
			t.Fatal(err)
		}
		for _, it := range items {
			got = append(got, it.Title)
		}
		if len(items) < 3 {
			break
		}
		last := items[len(items)-1]
		after = &news.Cursor{PublishedAt: last.PublishedAt, ID: last.ID}
	}
	// Seeded launch item (2026) comes after all 2030 items.
	want := []string{"n4", "n3", "n2", "n1", "n0", "E2EE Text project started"}
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Errorf("got %v want %v", got, want)
	}
}

func clear(t *testing.T, db *sql.DB) {
	t.Helper()
	if _, err := db.Exec(`DELETE FROM news WHERE title <> 'E2EE Text project started'`); err != nil {
		t.Fatal(err)
	}
}
