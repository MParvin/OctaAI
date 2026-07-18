package permission

import (
	"fmt"
	"strings"

	"github.com/mparvin/octaai/pkg/approval"
	"github.com/mparvin/octaai/pkg/config"
	"github.com/mparvin/octaai/pkg/storage"
)

// Decision is the outcome of a permission check.
type Decision string

const (
	DecisionAllow           Decision = "allow"
	DecisionDeny            Decision = "deny"
	DecisionRequireApproval Decision = "require_approval"
)

// CheckResult describes a permission evaluation.
type CheckResult struct {
	Decision Decision `json:"decision"`
	Reason   string   `json:"reason,omitempty"`
}

// Manager enforces safety policies before tool execution.
type Manager struct {
	cfg       *config.Config
	store     storage.Storage
	approvals *approval.Service
}

// NewManager creates a permission manager from config.
func NewManager(cfg *config.Config, store storage.Storage) *Manager {
	return &Manager{
		cfg:       cfg,
		store:     store,
		approvals: approval.NewService(store),
	}
}

// CheckPath verifies a filesystem path is within allowed roots.
func (m *Manager) CheckPath(path string) CheckResult {
	if len(m.cfg.Safety.AllowPaths) == 0 {
		return CheckResult{
			Decision: DecisionDeny,
			Reason:   "no allow_paths configured; refusing filesystem access",
		}
	}

	absPath, err := ResolveSafePath(m.cfg, path)
	if err != nil {
		return CheckResult{
			Decision: DecisionDeny,
			Reason:   fmt.Sprintf("invalid path %q: %v", path, err),
		}
	}

	if PathAllowed(m.cfg, absPath) {
		return CheckResult{Decision: DecisionAllow}
	}
	return CheckResult{
		Decision: DecisionDeny,
		Reason:   fmt.Sprintf("path %q is outside allowed paths", path),
	}
}

// CheckCommand verifies a shell command against deny/require lists.
func (m *Manager) CheckCommand(command string) CheckResult {
	normalized := NormalizeCommand(command)

	for _, denied := range m.cfg.Safety.DenyCommands {
		if strings.Contains(normalized, NormalizeCommand(denied)) {
			return CheckResult{
				Decision: DecisionDeny,
				Reason:   fmt.Sprintf("command matches denied pattern: %q", denied),
			}
		}
	}
	for _, confirm := range m.cfg.Safety.RequireConfirmationFor {
		if strings.Contains(normalized, NormalizeCommand(confirm)) {
			return CheckResult{
				Decision: DecisionRequireApproval,
				Reason:   fmt.Sprintf("command requires approval: %q", confirm),
			}
		}
	}
	return CheckResult{Decision: DecisionAllow}
}

// CheckTool evaluates permission for a tool invocation.
func (m *Manager) CheckTool(goalID, taskID, toolName string, args map[string]interface{}) CheckResult {
	if goalID != "" && m.store != nil {
		if ok, err := m.approvals.IsApproved(goalID, toolName, args); err == nil && ok {
			return CheckResult{Decision: DecisionAllow}
		}
	}

	switch toolName {
	case "command":
		cmd, _ := args["command"].(string)
		if result := m.CheckCommand(cmd); result.Decision != DecisionAllow {
			return result
		}
		if cwd, ok := args["cwd"].(string); ok && cwd != "" {
			return m.CheckPath(cwd)
		}
		return CheckResult{Decision: DecisionAllow}

	case "filesystem":
		path, _ := args["path"].(string)
		if path != "" {
			return m.CheckPath(path)
		}

	case "git":
		path, _ := args["path"].(string)
		if path == "" {
			return CheckResult{Decision: DecisionDeny, Reason: "git path is required"}
		}
		if result := m.CheckPath(path); result.Decision != DecisionAllow {
			return result
		}
		action, _ := args["action"].(string)
		if action == "push" {
			return CheckResult{
				Decision: DecisionRequireApproval,
				Reason:   "git push requires approval",
			}
		}
		if action == "clone" {
			url, _ := args["url"].(string)
			if strings.HasPrefix(strings.TrimSpace(url), "-") {
				return CheckResult{
					Decision: DecisionDeny,
					Reason:   "git clone URL cannot start with '-'",
				}
			}
			if result := m.CheckURL(url); result.Decision != DecisionAllow {
				return result
			}
		}
		return CheckResult{Decision: DecisionAllow}

	case "http":
		url, _ := args["url"].(string)
		if url == "" {
			return CheckResult{Decision: DecisionDeny, Reason: "http url is required"}
		}
		return m.CheckURL(url)

	case "browser":
		action, _ := args["action"].(string)
		if action == "execute" {
			return CheckResult{
				Decision: DecisionRequireApproval,
				Reason:   "browser JavaScript execution requires approval",
			}
		}
		if action == "screenshot" {
			if outputPath, ok := args["output_path"].(string); ok && outputPath != "" {
				return m.CheckPath(outputPath)
			}
		}
		if action == "navigate" {
			url, _ := args["url"].(string)
			if url == "" {
				return CheckResult{Decision: DecisionDeny, Reason: "browser navigate requires url"}
			}
			return m.CheckBrowserDomain(url)
		}
		return CheckResult{Decision: DecisionAllow}

	case "ssh":
		if localPath, ok := args["local_path"].(string); ok && localPath != "" {
			if result := m.CheckPath(localPath); result.Decision != DecisionAllow {
				return result
			}
		}
		return CheckResult{
			Decision: DecisionRequireApproval,
			Reason:   "remote SSH execution requires approval",
		}
	}

	return CheckResult{Decision: DecisionDeny, Reason: fmt.Sprintf("unknown tool: %s", toolName)}
}

// Approvals exposes the approval service.
func (m *Manager) Approvals() *approval.Service {
	return m.approvals
}
