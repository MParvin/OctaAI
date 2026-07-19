package approval

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/mparvin/octaai/pkg/storage"
)

func TestFingerprintStable(t *testing.T) {
	a := Fingerprint("command", map[string]interface{}{"cwd": "x", "command": "ls"})
	b := Fingerprint("command", map[string]interface{}{"command": "ls", "cwd": "x", "_isolated": true})
	if a == "" || a != b {
		t.Fatalf("fingerprint mismatch: %q vs %q", a, b)
	}
}

func seedGoal(t *testing.T, store *storage.SQLiteStorage, id string) {
	t.Helper()
	now := time.Now().UTC()
	if err := store.CreateGoal(&storage.Goal{
		ID:          id,
		Description: "approval test",
		State:       storage.StateIdle,
		CreatedAt:   now,
		UpdatedAt:   now,
	}); err != nil {
		t.Fatal(err)
	}
}

func TestApproveDenyRoundTrip(t *testing.T) {
	store, err := storage.NewSQLiteStorage(filepath.Join(t.TempDir(), "state.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	seedGoal(t, store, "g1")
	seedGoal(t, store, "g2")

	svc := NewService(store)
	req, err := svc.CreatePending("g1", "t1", "ssh", map[string]interface{}{"host": "h"}, "remote SSH")
	if err != nil {
		t.Fatal(err)
	}
	if req.Status != "pending" {
		t.Fatalf("status %s", req.Status)
	}

	approved, err := svc.Approve(req.ID)
	if err != nil {
		t.Fatal(err)
	}
	if approved.Status != "approved" {
		t.Fatalf("expected approved, got %s", approved.Status)
	}

	ok, err := svc.IsApproved("g1", "ssh", map[string]interface{}{"host": "h"})
	if err != nil || !ok {
		t.Fatalf("IsApproved: ok=%v err=%v", ok, err)
	}

	req2, err := svc.CreatePending("g2", "t2", "ssh", map[string]interface{}{"host": "h2"}, "remote SSH")
	if err != nil {
		t.Fatal(err)
	}
	denied, err := svc.Deny(req2.ID)
	if err != nil {
		t.Fatal(err)
	}
	if denied.Status != "denied" {
		t.Fatalf("expected denied, got %s", denied.Status)
	}
}
