package permission

import (
	"testing"

	"github.com/mparvin/octaai/pkg/config"
)

func testConfig() *config.Config {
	cfg := config.DefaultConfig()
	cfg.ProjectsRoot = "/home/user/Projects"
	cfg.Safety.AllowPaths = []string{"/home/user/Projects"}
	cfg.Safety.DenyCommands = []string{"rm -rf /"}
	cfg.Safety.RequireConfirmationFor = []string{"apt upgrade"}
	cfg.Browser.BrowserDomains = []string{"example.com"}
	return cfg
}

func TestNormalizeCommand(t *testing.T) {
	if got := NormalizeCommand("  RM   -rf  / "); got != "rm -rf /" {
		t.Fatalf("expected normalized command, got %q", got)
	}
}

func TestCheckCommandDenyBypass(t *testing.T) {
	m := NewManager(testConfig(), nil)
	result := m.CheckCommand("RM  -rf  /")
	if result.Decision != DecisionDeny {
		t.Fatalf("expected deny, got %s", result.Decision)
	}
}

func TestCheckCommandCwd(t *testing.T) {
	m := NewManager(testConfig(), nil)
	result := m.CheckTool("", "", "command", map[string]interface{}{
		"command": "ls",
		"cwd":     "/etc",
	})
	if result.Decision != DecisionDeny {
		t.Fatalf("expected deny for cwd outside allow_paths, got %s (%s)", result.Decision, result.Reason)
	}
}

func TestCheckGitPath(t *testing.T) {
	m := NewManager(testConfig(), nil)
	result := m.CheckTool("", "", "git", map[string]interface{}{
		"action": "init",
		"path":   "/tmp/repo",
	})
	if result.Decision != DecisionDeny {
		t.Fatalf("expected deny for git path outside allow_paths, got %s", result.Decision)
	}
}

func TestCheckGitPushRequiresApproval(t *testing.T) {
	m := NewManager(testConfig(), nil)
	result := m.CheckTool("", "", "git", map[string]interface{}{
		"action": "push",
		"path":   "myapp",
	})
	if result.Decision != DecisionRequireApproval {
		t.Fatalf("expected require_approval, got %s", result.Decision)
	}
}

func TestCheckHTTPBlocksLocalhost(t *testing.T) {
	m := NewManager(testConfig(), nil)
	result := m.CheckTool("", "", "http", map[string]interface{}{
		"method": "GET",
		"url":    "http://localhost:8080/admin",
	})
	if result.Decision != DecisionDeny {
		t.Fatalf("expected deny for localhost SSRF, got %s", result.Decision)
	}
}

func TestCheckBrowserExecuteRequiresApproval(t *testing.T) {
	m := NewManager(testConfig(), nil)
	result := m.CheckTool("", "", "browser", map[string]interface{}{
		"action": "execute",
		"script": "alert(1)",
	})
	if result.Decision != DecisionRequireApproval {
		t.Fatalf("expected require_approval, got %s", result.Decision)
	}
}

func TestCheckBrowserDomain(t *testing.T) {
	m := NewManager(testConfig(), nil)
	result := m.CheckTool("", "", "browser", map[string]interface{}{
		"action": "navigate",
		"url":    "https://evil.com/page",
	})
	if result.Decision != DecisionDeny {
		t.Fatalf("expected deny for disallowed browser domain, got %s", result.Decision)
	}

	allowed := m.CheckTool("", "", "browser", map[string]interface{}{
		"action": "navigate",
		"url":    "https://www.example.com/page",
	})
	if allowed.Decision != DecisionAllow {
		t.Fatalf("expected allow for example.com, got %s (%s)", allowed.Decision, allowed.Reason)
	}
}

func TestUnknownToolDenied(t *testing.T) {
	m := NewManager(testConfig(), nil)
	result := m.CheckTool("", "", "unknown_tool", nil)
	if result.Decision != DecisionDeny {
		t.Fatalf("expected deny for unknown tool, got %s", result.Decision)
	}
}
