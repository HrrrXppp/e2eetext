package database

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

const (
	defaultWaitTimeout  = 60 * time.Second
	defaultRetryBackoff = 1 * time.Second
)

func WaitFor(ctx context.Context, databaseURL string) error {
	return waitFor(ctx, databaseURL, defaultWaitTimeout, defaultRetryBackoff)
}

func waitFor(ctx context.Context, databaseURL string, timeout, backoff time.Duration) error {
	deadline := time.Now().Add(timeout)
	var lastErr error

	for attempt := 1; ; attempt++ {
		if err := ping(ctx, databaseURL); err == nil {
			return nil
		} else {
			lastErr = err
		}

		if time.Now().After(deadline) {
			return fmt.Errorf(
				"database not ready after %s: %w (start postgres with: docker compose up -d postgres)",
				timeout,
				lastErr,
			)
		}

		select {
		case <-ctx.Done():
			return fmt.Errorf("wait for database: %w", ctx.Err())
		case <-time.After(backoff):
		}

		_ = attempt
	}
}

func ping(ctx context.Context, databaseURL string) error {
	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return err
	}
	defer db.Close()

	pingCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	if err := db.PingContext(pingCtx); err != nil {
		if strings.Contains(err.Error(), "connection refused") {
			return fmt.Errorf("postgres is not running on %s: %w", hostFromURL(databaseURL), err)
		}
		return err
	}

	return nil
}

func hostFromURL(databaseURL string) string {
	// postgres://user:pass@host:port/db -> host:port
	withoutScheme := strings.TrimPrefix(databaseURL, "postgres://")
	if at := strings.LastIndex(withoutScheme, "@"); at >= 0 {
		withoutScheme = withoutScheme[at+1:]
	}
	if slash := strings.Index(withoutScheme, "/"); slash >= 0 {
		withoutScheme = withoutScheme[:slash]
	}
	if withoutScheme == "" {
		return "localhost:5432"
	}
	return withoutScheme
}
