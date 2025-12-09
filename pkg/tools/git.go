package tools

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/mparvin/octaai/pkg/config"
)

// GitTool provides git operations
type GitTool struct {
	cfg *config.Config
}

// NewGitTool creates a new git tool
func NewGitTool(cfg *config.Config) *GitTool {
	return &GitTool{cfg: cfg}
}

// Name implements Tool.Name
func (t *GitTool) Name() string {
	return "git"
}

// Schema implements Tool.Schema
func (t *GitTool) Schema() ToolSchema {
	return ToolSchema{
		Name:        "git",
		Description: "Git repository operations (clone, commit, push)",
		Parameters: map[string]ParamSchema{
			"action": {
				Type:        "string",
				Description: "Git action to perform",
				Required:    true,
				Enum:        []string{"clone", "init", "commit_all", "push"},
			},
			"url": {
				Type:        "string",
				Description: "Repository URL (for clone)",
				Required:    false,
			},
			"path": {
				Type:        "string",
				Description: "Local repository path",
				Required:    true,
			},
			"message": {
				Type:        "string",
				Description: "Commit message (for commit_all)",
				Required:    false,
			},
			"remote": {
				Type:        "string",
				Description: "Remote name (default: origin)",
				Required:    false,
			},
			"branch": {
				Type:        "string",
				Description: "Branch name (default: main)",
				Required:    false,
			},
		},
	}
}

// Execute implements Tool.Execute
func (t *GitTool) Execute(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
	action, ok := args["action"].(string)
	if !ok {
		return nil, fmt.Errorf("action is required")
	}

	path, ok := args["path"].(string)
	if !ok {
		return nil, fmt.Errorf("path is required")
	}

	switch action {
	case "clone":
		url, _ := args["url"].(string)
		return t.clone(ctx, url, path)
	case "init":
		return t.init(ctx, path)
	case "commit_all":
		message, _ := args["message"].(string)
		return t.commitAll(ctx, path, message)
	case "push":
		remote, _ := args["remote"].(string)
		if remote == "" {
			remote = "origin"
		}
		branch, _ := args["branch"].(string)
		if branch == "" {
			branch = "main"
		}
		return t.push(ctx, path, remote, branch)
	default:
		return nil, fmt.Errorf("unknown action: %s", action)
	}
}

func (t *GitTool) clone(ctx context.Context, url, destPath string) (*ToolResult, error) {
	if url == "" {
		return nil, fmt.Errorf("url is required for clone")
	}

	// Ensure parent directory exists
	parentDir := filepath.Dir(destPath)
	if err := os.MkdirAll(parentDir, 0755); err != nil {
		return &ToolResult{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	cmd := exec.CommandContext(ctx, "git", "clone", url, destPath)
	output, err := cmd.CombinedOutput()

	if err != nil {
		return &ToolResult{
			Success: false,
			Error:   string(output),
		}, nil
	}

	return &ToolResult{
		Success: true,
		Output:  fmt.Sprintf("Cloned repository to: %s", destPath),
	}, nil
}

func (t *GitTool) init(ctx context.Context, path string) (*ToolResult, error) {
	if err := os.MkdirAll(path, 0755); err != nil {
		return &ToolResult{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	cmd := exec.CommandContext(ctx, "git", "init")
	cmd.Dir = path
	output, err := cmd.CombinedOutput()

	if err != nil {
		return &ToolResult{
			Success: false,
			Error:   string(output),
		}, nil
	}

	return &ToolResult{
		Success: true,
		Output:  fmt.Sprintf("Initialized git repository at: %s", path),
	}, nil
}

func (t *GitTool) commitAll(ctx context.Context, path, message string) (*ToolResult, error) {
	if message == "" {
		message = "Automated commit by OctaAI"
	}

	// Add all files
	cmdAdd := exec.CommandContext(ctx, "git", "add", ".")
	cmdAdd.Dir = path
	if output, err := cmdAdd.CombinedOutput(); err != nil {
		return &ToolResult{
			Success: false,
			Error:   string(output),
		}, nil
	}

	// Commit
	cmdCommit := exec.CommandContext(ctx, "git", "commit", "-m", message)
	cmdCommit.Dir = path
	output, err := cmdCommit.CombinedOutput()

	if err != nil {
		return &ToolResult{
			Success: false,
			Error:   string(output),
		}, nil
	}

	return &ToolResult{
		Success: true,
		Output:  fmt.Sprintf("Committed changes: %s", message),
	}, nil
}

func (t *GitTool) push(ctx context.Context, path, remote, branch string) (*ToolResult, error) {
	cmd := exec.CommandContext(ctx, "git", "push", remote, branch)
	cmd.Dir = path
	output, err := cmd.CombinedOutput()

	if err != nil {
		return &ToolResult{
			Success: false,
			Error:   string(output),
		}, nil
	}

	return &ToolResult{
		Success: true,
		Output:  fmt.Sprintf("Pushed to %s/%s", remote, branch),
	}, nil
}
