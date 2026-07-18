package permission

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mparvin/octaai/pkg/config"
)

func TestPathAllowedSiblingPrefix(t *testing.T) {
	root := t.TempDir()
	cfg := config.DefaultConfig()
	cfg.ProjectsRoot = root
	cfg.Safety.AllowPaths = []string{root}

	inside := filepath.Join(root, "app")
	if err := os.MkdirAll(inside, 0755); err != nil {
		t.Fatal(err)
	}
	if !PathAllowed(cfg, inside) {
		t.Fatal("expected path under allow_paths to be allowed")
	}

	sibling := root + "evil"
	if PathAllowed(cfg, sibling) {
		t.Fatalf("sibling-prefix path must be denied: %s", sibling)
	}
}

func TestPathAllowedRejectsEmptyAllowPaths(t *testing.T) {
	root := t.TempDir()
	cfg := config.DefaultConfig()
	cfg.ProjectsRoot = root
	cfg.Safety.AllowPaths = nil

	if PathAllowed(cfg, filepath.Join(root, "x")) {
		t.Fatal("empty allow_paths must deny all paths")
	}

	m := NewManager(cfg, nil)
	result := m.CheckPath(filepath.Join(root, "x"))
	if result.Decision != DecisionDeny {
		t.Fatalf("expected deny when allow_paths empty, got %s", result.Decision)
	}
}

func TestResolveSafePathSymlinkEscape(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()

	secret := filepath.Join(outside, "secret.txt")
	if err := os.WriteFile(secret, []byte("nope"), 0600); err != nil {
		t.Fatal(err)
	}

	link := filepath.Join(root, "escape")
	if err := os.Symlink(outside, link); err != nil {
		t.Fatal(err)
	}

	cfg := config.DefaultConfig()
	cfg.ProjectsRoot = root
	cfg.Safety.AllowPaths = []string{root}

	resolved, err := ResolveSafePath(cfg, "escape/secret.txt")
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if PathAllowed(cfg, resolved) {
		t.Fatalf("symlink escape into %s must be denied (resolved=%s)", outside, resolved)
	}

	m := NewManager(cfg, nil)
	result := m.CheckPath("escape/secret.txt")
	if result.Decision != DecisionDeny {
		t.Fatalf("expected deny for symlink escape, got %s (%s)", result.Decision, result.Reason)
	}
}

func TestResolveSafePathInsideSymlink(t *testing.T) {
	root := t.TempDir()
	realDir := filepath.Join(root, "real")
	if err := os.MkdirAll(realDir, 0755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "via-link")
	if err := os.Symlink(realDir, link); err != nil {
		t.Fatal(err)
	}

	cfg := config.DefaultConfig()
	cfg.ProjectsRoot = root
	cfg.Safety.AllowPaths = []string{root}

	resolved, err := ResolveSafePath(cfg, "via-link")
	if err != nil {
		t.Fatal(err)
	}
	if !PathAllowed(cfg, resolved) {
		t.Fatalf("symlink staying inside allow_paths should be allowed: %s", resolved)
	}
}
