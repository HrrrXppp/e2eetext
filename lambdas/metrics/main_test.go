package main

import "testing"

func TestNormalizeDatabaseURLTrimsSpace(t *testing.T) {
	dsn, err := normalizeDatabaseURL("  postgres://messenger:secret@db.example:5432/messenger?sslmode=require\n")
	if err != nil {
		t.Fatal(err)
	}
	const want = "postgres://messenger:secret@db.example:5432/messenger?sslmode=require"
	if dsn != want {
		t.Fatalf("dsn = %q, want %q", dsn, want)
	}
}

func TestNormalizeDatabaseURLRejectsEmpty(t *testing.T) {
	if _, err := normalizeDatabaseURL(" \n"); err == nil {
		t.Fatal("expected error for a blank secret")
	}
}
