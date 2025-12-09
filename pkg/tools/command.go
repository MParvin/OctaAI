package tools

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/mparvin/octaai/pkg/config"
)

// CommandTool executes shell commands
type CommandTool struct {
	cfg *config.Config
}

// NewCommandTool creates a new command tool
func NewCommandTool(cfg *config.Config) *CommandTool {
	return &CommandTool{cfg: cfg}
}

// Name implements Tool.Name
func (t *CommandTool) Name() string {
	return "command"
}

// Schema implements Tool.Schema
func (t *CommandTool) Schema() ToolSchema {
	return ToolSchema{
		Name:        "command",
		Description: "Execute shell commands and programs",
		Parameters: map[string]ParamSchema{
			"cwd": {
				Type:        "string",
				Description: "Working directory (relative to projects_root)",
				Required:    true,
			},
			"command": {
				Type:        "string",
				Description: "Command and arguments as array",
				Required:    true,
			},
			"timeout": {
				Type:        "number",
				Description: "Timeout in seconds (default: 300)",
				Required:    false,
			},
		},
	}
}

// Execute implements Tool.Execute
func (t *CommandTool) Execute(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
	cwd, ok := args["cwd"].(string)
	if !ok {
		return nil, fmt.Errorf("cwd is required")
	}

	cmdStr, ok := args["command"].(string)
	if !ok {
		return nil, fmt.Errorf("command is required")
	}

	// Check if command is denied
	if t.isCommandDenied(cmdStr) {
		return &ToolResult{
			Success: false,
			Error:   "command is denied by safety policy",
		}, nil
	}

	// Parse timeout
	timeout := 300 * time.Second
	if timeoutVal, ok := args["timeout"].(float64); ok {
		timeout = time.Duration(timeoutVal) * time.Second
	}

	// Create context with timeout
	cmdCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Parse command
	parts := strings.Fields(cmdStr)
	if len(parts) == 0 {
		return nil, fmt.Errorf("empty command")
	}

	// Create command
	cmd := exec.CommandContext(cmdCtx, parts[0], parts[1:]...)
	cmd.Dir = t.resolvePath(cwd)

	// Execute command
	output, err := cmd.CombinedOutput()

	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			return &ToolResult{
				Success: false,
				Error:   err.Error(),
			}, nil
		}
	}

	success := exitCode == 0

	return &ToolResult{
		Success: success,
		Output:  string(output),
		Data: map[string]interface{}{
			"exit_code": exitCode,
			"command":   cmdStr,
			"cwd":       cmd.Dir,
		},
	}, nil
}

func (t *CommandTool) resolvePath(path string) string {
	// Similar to filesystem tool
	return path
}

func (t *CommandTool) isCommandDenied(cmdStr string) bool {
	for _, denied := range t.cfg.Safety.DenyCommands {
		if strings.Contains(cmdStr, denied) {
			return true
		}
	}
	return false
}
