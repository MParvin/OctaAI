package engine

import (
	"context"
	"testing"
	"time"

	"github.com/mparvin/octaai/pkg/config"
	"github.com/mparvin/octaai/pkg/observability"
	"github.com/mparvin/octaai/pkg/permission"
	"github.com/mparvin/octaai/pkg/tools"
)

func TestRunnerDeniesUnknownTool(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.ProjectsRoot = t.TempDir()
	cfg.Safety.AllowPaths = []string{cfg.ProjectsRoot}
	reg := tools.NewRegistry()
	perm := permission.NewManager(cfg, nil)
	logger := observability.NewLogger(nil)
	r := NewRunner(cfg, reg, perm, logger)

	out, err := r.Run(context.Background(), "nope", map[string]interface{}{}, RunConfig{Timeout: time.Second})
	if err != nil {
		// unknown tool denied by permission before execute
		return
	}
	if out != nil && out.Success {
		t.Fatal("expected deny")
	}
}

func TestRunnerFilesystemAllowed(t *testing.T) {
	root := t.TempDir()
	cfg := config.DefaultConfig()
	cfg.ProjectsRoot = root
	cfg.Safety.AllowPaths = []string{root}
	reg := tools.NewRegistry()
	reg.Register(tools.NewFilesystemTool(cfg))
	perm := permission.NewManager(cfg, nil)
	logger := observability.NewLogger(nil)
	r := NewRunner(cfg, reg, perm, logger)

	out, err := r.Run(context.Background(), "filesystem", map[string]interface{}{
		"action":  "write_file",
		"path":    "x.txt",
		"content": "hi",
	}, RunConfig{Timeout: time.Second})
	if err != nil {
		t.Fatal(err)
	}
	if out == nil || !out.Success {
		t.Fatalf("unexpected: %+v", out)
	}
}
