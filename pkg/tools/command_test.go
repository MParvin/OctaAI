package tools

import (
	"testing"

	"github.com/mparvin/octaai/pkg/config"
)

func testCommandConfig() *config.Config {
	cfg := config.DefaultConfig()
	cfg.ProjectsRoot = "/home/user/Projects"
	cfg.Safety.AllowPaths = []string{"/home/user/Projects"}
	cfg.Safety.DenyCommands = []string{"rm -rf /"}
	return cfg
}

func TestCommandToolDenyBypass(t *testing.T) {
	tool := NewCommandTool(testCommandConfig())
	if !tool.isCommandDenied("RM  -rf  /") {
		t.Fatal("expected denied command with whitespace/case bypass attempt")
	}
}

func TestCommandToolAllowsSafeCommand(t *testing.T) {
	tool := NewCommandTool(testCommandConfig())
	if tool.isCommandDenied("ls -la") {
		t.Fatal("expected ls -la to be allowed")
	}
}
