package storage

import (
	"path/filepath"
	"testing"
	"time"
)

func TestSQLiteWALAndGoalRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.db")
	store, err := NewSQLiteStorage(path)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	var mode string
	if err := store.db.QueryRow(`PRAGMA journal_mode;`).Scan(&mode); err != nil {
		t.Fatal(err)
	}
	if mode != "wal" && mode != "WAL" {
		t.Fatalf("expected WAL journal mode, got %q", mode)
	}

	now := time.Now().UTC()
	goal := &Goal{
		ID:          "g1",
		Description: "test goal",
		State:       StateIdle,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := store.CreateGoal(goal); err != nil {
		t.Fatal(err)
	}
	got, err := store.GetGoal("g1")
	if err != nil {
		t.Fatal(err)
	}
	if got.Description != "test goal" {
		t.Fatalf("unexpected goal: %+v", got)
	}
}
