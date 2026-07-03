package node

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
)

const separator = "/"

type Registry struct {
	ID string
}

func Load(ctx context.Context, db *sql.DB) (Registry, error) {
	var id string
	err := db.QueryRowContext(ctx, `SELECT id FROM node`).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		err = db.QueryRowContext(ctx, `
			INSERT INTO node DEFAULT VALUES
			RETURNING id`,
		).Scan(&id)
	}
	if err != nil {
		return Registry{}, fmt.Errorf("load node id: %w", err)
	}

	return Registry{ID: id}, nil
}

func (r Registry) ScopeID(localID string) string {
	localID = strings.TrimSpace(localID)
	if localID == "" {
		return ""
	}
	return r.ID + separator + localID
}

func (r Registry) LocalID(scopedID string) (string, error) {
	scopedID = strings.TrimSpace(scopedID)
	if scopedID == "" {
		return "", errors.New("id is required")
	}

	if nodeID, localID, ok := strings.Cut(scopedID, separator); ok {
		localID = strings.TrimSpace(localID)
		if localID == "" {
			return "", fmt.Errorf("invalid id %q", scopedID)
		}
		_ = nodeID
		return localID, nil
	}

	return scopedID, nil
}
