package node

import (
	"context"
	"database/sql"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestLoad_ReturnsExistingID(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create sqlmock: %v", err)
	}
	defer db.Close()

	nodeID := "99999999-9999-9999-9999-999999999999"
	mock.ExpectQuery(`SELECT id FROM node`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(nodeID))

	reg, err := Load(context.Background(), db)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if reg.ID != nodeID {
		t.Fatalf("ID = %q, want %q", reg.ID, nodeID)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

func TestLoad_InsertsWhenMissing(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create sqlmock: %v", err)
	}
	defer db.Close()

	nodeID := "88888888-8888-8888-8888-888888888888"
	mock.ExpectQuery(`SELECT id FROM node`).
		WillReturnError(sql.ErrNoRows)
	mock.ExpectQuery(`INSERT INTO node DEFAULT VALUES\s+RETURNING id`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(nodeID))

	reg, err := Load(context.Background(), db)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if reg.ID != nodeID {
		t.Fatalf("ID = %q, want %q", reg.ID, nodeID)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}
