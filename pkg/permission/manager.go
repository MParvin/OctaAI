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
	cfg      *config.Config
	store    storage.Storage
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
		return CheckResult{Decision: DecisionAllow}
	}
	for _, allowed := range m.cfg.Safety.AllowPaths {
		if strings.HasPrefix(path, allowed) || strings.HasPrefix(allowed, path) {
			return CheckResult{Decision: DecisionAllow}
		}
	}
	return CheckResult{
		Decision: DecisionDeny,
		Reason:   fmt.Sprintf("path %q is outside allowed paths", path),
	}
}

// CheckCommand verifies a shell command against deny/require lists.
func (m *Manager) CheckCommand(command string) CheckResult {
	for _, denied := range m.cfg.Safety.DenyCommands {
		if strings.Contains(command, denied) {
			return CheckResult{
				Decision: DecisionDeny,
				Reason:   fmt.Sprintf("command matches denied pattern: %q", denied),
			}
		}
	}
	for _, confirm := range m.cfg.Safety.RequireConfirmationFor {
		if strings.Contains(command, confirm) {
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
		return m.CheckCommand(cmd)
	case "filesystem":
		path, _ := args["path"].(string)
		if path != "" {
			return m.CheckPath(path)
		}
	case "ssh":
		return CheckResult{
			Decision: DecisionRequireApproval,
			Reason:   "remote SSH execution requires approval",
		}
	}
	return CheckResult{Decision: DecisionAllow}
}

// Approvals exposes the approval service.
func (m *Manager) Approvals() *approval.Service {
	return m.approvals
}
