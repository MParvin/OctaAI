package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExpandPath(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("no home directory")
	}

	tests := []struct {
		in, want string
	}{
		{"~", home},
		{"~/Projects", filepath.Join(home, "Projects")},
		{"/abs/path", "/abs/path"},
	}

	for _, tc := range tests {
		got := ExpandPath(tc.in)
		if got != tc.want {
			t.Errorf("ExpandPath(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestResolveProjectPath(t *testing.T) {
	root := "/home/user/projects"
	cfg := &Config{ProjectsRoot: root}

	got := ResolveProjectPath(cfg, "myapp/main.py")
	want := filepath.Join(root, "myapp/main.py")
	if got != want {
		t.Errorf("ResolveProjectPath() = %q, want %q", got, want)
	}

	got = ResolveProjectPath(cfg, "/abs/outside.py")
	if got != "/abs/outside.py" {
		t.Errorf("absolute path = %q, want /abs/outside.py", got)
	}
}

func TestLoadConfigExpandsTildePaths(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("no home directory")
	}

	cfg := DefaultConfig()
	cfg.Storage.Path = "~/.config/octaai/state.db"
	cfg.Safety.AllowPaths = []string{"~/Projects"}

	cfg.ProjectsRoot = ExpandPath(cfg.ProjectsRoot)
	for i, p := range cfg.Safety.AllowPaths {
		cfg.Safety.AllowPaths[i] = ExpandPath(p)
	}
	cfg.Storage.Path = ExpandPath(cfg.Storage.Path)

	wantStorage := filepath.Join(home, ".config", "octaai", "state.db")
	if cfg.Storage.Path != wantStorage {
		t.Errorf("storage path = %q, want %q", cfg.Storage.Path, wantStorage)
	}
	if cfg.Safety.AllowPaths[0] != filepath.Join(home, "Projects") {
		t.Errorf("allow path = %q", cfg.Safety.AllowPaths[0])
	}
}

func TestRelativePathAllowedUnderProjectsRoot(t *testing.T) {
	cfg := &Config{
		ProjectsRoot: "/home/user/MadProjects",
		Safety: SafetyConfig{
			AllowPaths: []string{"/home/user/MadProjects"},
		},
	}

	full := ResolveProjectPath(cfg, "project/app.py")
	absPath, err := filepath.Abs(full)
	if err != nil {
		t.Fatal(err)
	}

	allowedAbs, _ := filepath.Abs(cfg.Safety.AllowPaths[0])
	allowed := absPath == allowedAbs || strings.HasPrefix(absPath, allowedAbs+string(filepath.Separator))
	if !allowed {
		t.Errorf("project/app.py should be allowed under projects root, got %q", absPath)
	}
}
