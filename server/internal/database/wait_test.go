package database

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestHostFromURL(t *testing.T) {
	tests := []struct {
		url  string
		want string
	}{
		{
			url:  "postgres://messenger:messenger@localhost:5432/messenger?sslmode=disable",
			want: "localhost:5432",
		},
		{
			url:  "postgres://messenger:messenger@127.0.0.1:5433/messenger",
			want: "127.0.0.1:5433",
		},
	}

	for _, tt := range tests {
		if got := hostFromURL(tt.url); got != tt.want {
			t.Errorf("hostFromURL(%q) = %q, want %q", tt.url, got, tt.want)
		}
	}
}

func TestWaitFor_ContextCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := waitFor(ctx, "postgres://messenger:messenger@127.0.0.1:9/messenger?sslmode=disable", time.Second, time.Millisecond)
	if err == nil || !strings.Contains(err.Error(), "context canceled") {
		t.Fatalf("waitFor() error = %v, want context canceled", err)
	}
}
