package tools

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/mparvin/octaai/pkg/config"
)

func TestRegistryRegisterGetListSchemas(t *testing.T) {
	reg := NewRegistry()
	cfg := config.DefaultConfig()
	cfg.ProjectsRoot = t.TempDir()
	cfg.Safety.AllowPaths = []string{cfg.ProjectsRoot}
	fs := NewFilesystemTool(cfg)
	reg.Register(fs)

	got, ok := reg.Get("filesystem")
	if !ok || got.Name() != "filesystem" {
		t.Fatal("Get failed")
	}
	if len(reg.List()) != 1 {
		t.Fatalf("List=%d", len(reg.List()))
	}
	if len(reg.Schemas()) != 1 {
		t.Fatalf("Schemas=%d", len(reg.Schemas()))
	}
}

func TestFilesystemCreateListAppend(t *testing.T) {
	root := t.TempDir()
	cfg := config.DefaultConfig()
	cfg.ProjectsRoot = root
	cfg.Safety.AllowPaths = []string{root}
	tool := NewFilesystemTool(cfg)

	res, err := tool.Execute(context.Background(), map[string]interface{}{
		"action": "create_directory",
		"path":   "subdir",
	})
	if err != nil || !res.Success {
		t.Fatalf("mkdir: %+v err=%v", res, err)
	}

	res, err = tool.Execute(context.Background(), map[string]interface{}{
		"action":  "append_file",
		"path":    "subdir/a.txt",
		"content": "line1\n",
	})
	if err != nil || !res.Success {
		t.Fatalf("append: %+v err=%v", res, err)
	}

	res, err = tool.Execute(context.Background(), map[string]interface{}{
		"action": "list_files",
		"path":   "subdir",
	})
	if err != nil || !res.Success {
		t.Fatalf("list: %+v err=%v", res, err)
	}
	if _, err := os.Stat(filepath.Join(root, "subdir", "a.txt")); err != nil {
		t.Fatal(err)
	}
}

func TestGitInitPathDenied(t *testing.T) {
	root := t.TempDir()
	cfg := config.DefaultConfig()
	cfg.ProjectsRoot = root
	cfg.Safety.AllowPaths = []string{root}
	tool := NewGitTool(cfg)

	res, err := tool.Execute(context.Background(), map[string]interface{}{
		"action": "init",
		"path":   "/etc/passwd-repo",
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Success {
		t.Fatal("expected path deny")
	}
}

func TestGitInitAllowed(t *testing.T) {
	root := t.TempDir()
	cfg := config.DefaultConfig()
	cfg.ProjectsRoot = root
	cfg.Safety.AllowPaths = []string{root}
	tool := NewGitTool(cfg)

	res, err := tool.Execute(context.Background(), map[string]interface{}{
		"action": "init",
		"path":   "repo",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !res.Success {
		t.Fatalf("init failed: %s", res.Error)
	}
}
