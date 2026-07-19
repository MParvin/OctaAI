package tools

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/mparvin/octaai/pkg/config"
	"github.com/mparvin/octaai/pkg/permission"
)

func testFilesystemConfig(root string) *config.Config {
	cfg := config.DefaultConfig()
	cfg.ProjectsRoot = root
	cfg.Safety.AllowPaths = []string{root}
	return cfg
}

func TestFilesystemToolPathAllowance(t *testing.T) {
	root := t.TempDir()
	cfg := testFilesystemConfig(root)

	allowed := filepath.Join(root, "app")
	if err := os.MkdirAll(allowed, 0755); err != nil {
		t.Fatal(err)
	}
	if !permission.PathAllowed(cfg, allowed) {
		t.Fatal("expected path inside projects root to be allowed")
	}

	outside := filepath.Join(root, "..", "outside")
	absOutside, _ := filepath.Abs(outside)
	if permission.PathAllowed(cfg, absOutside) {
		t.Fatal("expected path outside projects root to be denied")
	}

	// Sibling prefix must not match (e.g. /tmp/proj vs /tmp/project)
	sibling := root + "evil"
	if permission.PathAllowed(cfg, sibling) {
		t.Fatal("expected sibling-prefix path to be denied")
	}
}

func TestFilesystemWriteAndRead(t *testing.T) {
	root := t.TempDir()
	tool := NewFilesystemTool(testFilesystemConfig(root))

	writeResult, err := tool.Execute(context.Background(), map[string]interface{}{
		"action":  "write_file",
		"path":    "hello.txt",
		"content": "hello world",
	})
	if err != nil {
		t.Fatalf("write failed: %v", err)
	}
	if !writeResult.Success {
		t.Fatalf("write unsuccessful: %s", writeResult.Error)
	}

	readResult, err := tool.Execute(context.Background(), map[string]interface{}{
		"action": "read_file",
		"path":   "hello.txt",
	})
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}
	if readResult.Output != "hello world" {
		t.Fatalf("unexpected content: %q", readResult.Output)
	}

	fullPath := filepath.Join(root, "hello.txt")
	if _, err := os.Stat(fullPath); err != nil {
		t.Fatalf("file not created on disk: %v", err)
	}
}
