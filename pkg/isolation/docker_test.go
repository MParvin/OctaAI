package isolation

import (
	"testing"

	"github.com/mparvin/octaai/pkg/config"
)

func TestWrapCommand(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Isolation.Enabled = true
	cfg.Isolation.Docker.Enabled = true
	cfg.ProjectsRoot = "/home/user/Projects"

	d := NewDocker(cfg)
	parts, err := d.WrapCommand("myapp", "go test ./...")
	if err != nil {
		t.Fatal(err)
	}
	joined := ""
	for i, p := range parts {
		if i > 0 {
			joined += " "
		}
		joined += p
	}
	if joined[:6] != "docker" {
		t.Fatalf("expected docker command, got %q", joined)
	}
	if !contains(joined, "go test ./...") {
		t.Fatalf("expected wrapped command in docker args, got %q", joined)
	}
}

func TestShouldIsolate(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Isolation.Enabled = true
	cfg.Isolation.Docker.Enabled = true

	d := NewDocker(cfg)
	if !d.ShouldIsolate("command") {
		t.Fatal("expected command tool to be isolated")
	}
	if d.ShouldIsolate("filesystem") {
		t.Fatal("filesystem should not require docker")
	}
}

func TestWrapArgsPreservesArgv(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Isolation.Enabled = true
	cfg.Isolation.Docker.Enabled = true

	d := NewDocker(cfg)
	args := map[string]interface{}{
		"cwd":     "myapp",
		"command": "go test ./...",
	}
	wrapped, ok, err := d.WrapArgs("command", args)
	if err != nil || !ok {
		t.Fatalf("expected wrapped args, got ok=%v err=%v", ok, err)
	}
	argv, exists := wrapped["_docker_argv"].([]string)
	if !exists || len(argv) == 0 || argv[0] != "docker" {
		t.Fatalf("expected docker argv slice, got %v", wrapped["_docker_argv"])
	}
	rewritten, ok := wrapped["command"].(string)
	if !ok || rewritten != "go test ./..." {
		t.Fatalf("expected original command preserved, got %v", wrapped["command"])
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 || indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
