package news

import (
	"context"
	"database/sql"
	"time"
)

// Item is one news entry.
type Item struct {
	ID          int64     `json:"id"`
	Title       string    `json:"title"`
	Body        string    `json:"body"`
	PublishedAt time.Time `json:"publishedAt"`
}

// Lister returns news newest first.
type Lister interface {
	// List returns up to limit items strictly older than after (all items when nil).
	List(ctx context.Context, after *Cursor, limit int) ([]Item, error)
}

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) List(ctx context.Context, after *Cursor, limit int) ([]Item, error) {
	var (
		rows *sql.Rows
		err  error
	)
	const cols = `SELECT id, title, body, published_at FROM news`
	const order = ` ORDER BY published_at DESC, id DESC LIMIT `
	if after == nil {
		rows, err = r.db.QueryContext(ctx, cols+order+`$1`, limit)
	} else {
		rows, err = r.db.QueryContext(ctx,
			cols+` WHERE (published_at, id) < ($1, $2)`+order+`$3`,
			after.PublishedAt, after.ID, limit)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []Item{}
	for rows.Next() {
		var it Item
		if err := rows.Scan(&it.ID, &it.Title, &it.Body, &it.PublishedAt); err != nil {
			return nil, err
		}
		it.PublishedAt = it.PublishedAt.UTC()
		items = append(items, it)
	}
	return items, rows.Err()
}
