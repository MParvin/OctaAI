package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mparvin/octaai/pkg/config"
)

// FilesystemTool provides file and directory operations
type FilesystemTool struct {
	cfg *config.Config
}

// NewFilesystemTool creates a new filesystem tool
func NewFilesystemTool(cfg *config.Config) *FilesystemTool {
	return &FilesystemTool{cfg: cfg}
}

// Name implements Tool.Name
func (t *FilesystemTool) Name() string {
	return "filesystem"
}

// Schema implements Tool.Schema
func (t *FilesystemTool) Schema() ToolSchema {
	return ToolSchema{
		Name:        "filesystem",
		Description: "Create, read, write, and list files and directories",
		Parameters: map[string]ParamSchema{
			"action": {
				Type:        "string",
				Description: "Action to perform",
				Required:    true,
				Enum:        []string{"create_directory", "write_file", "read_file", "list_files", "append_file"},
			},
			"path": {
				Type:        "string",
				Description: "File or directory path (relative to projects_root)",
				Required:    true,
			},
			"content": {
				Type:        "string",
				Description: "Content for write/append operations",
				Required:    false,
			},
		},
	}
}

// Execute implements Tool.Execute
func (t *FilesystemTool) Execute(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
	action, ok := args["action"].(string)
	if !ok {
		return nil, fmt.Errorf("action is required")
	}

	pathStr, ok := args["path"].(string)
	if !ok {
		return nil, fmt.Errorf("path is required")
	}

	// Resolve path
	fullPath := t.resolvePath(pathStr)

	// Check if path is allowed
	if !t.isPathAllowed(fullPath) {
		return &ToolResult{
			Success: false,
			Error:   fmt.Sprintf("path not allowed: %s", fullPath),
		}, nil
	}

	switch action {
	case "create_directory":
		return t.createDirectory(fullPath)
	case "write_file":
		content, _ := args["content"].(string)
		return t.writeFile(fullPath, content)
	case "read_file":
		return t.readFile(fullPath)
	case "list_files":
		return t.listFiles(fullPath)
	case "append_file":
		content, _ := args["content"].(string)
		return t.appendFile(fullPath, content)
	default:
		return nil, fmt.Errorf("unknown action: %s", action)
	}
}

func (t *FilesystemTool) resolvePath(path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(t.cfg.ProjectsRoot, path)
}

func (t *FilesystemTool) isPathAllowed(path string) bool {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return false
	}

	for _, allowedPath := range t.cfg.Safety.AllowPaths {
		allowedAbs, err := filepath.Abs(allowedPath)
		if err != nil {
			continue
		}
		if strings.HasPrefix(absPath, allowedAbs) {
			return true
		}
	}
	return false
}

func (t *FilesystemTool) createDirectory(path string) (*ToolResult, error) {
	if err := os.MkdirAll(path, 0755); err != nil {
		return &ToolResult{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	return &ToolResult{
		Success: true,
		Output:  fmt.Sprintf("Created directory: %s", path),
	}, nil
}

func (t *FilesystemTool) writeFile(path, content string) (*ToolResult, error) {
	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return &ToolResult{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return &ToolResult{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	return &ToolResult{
		Success: true,
		Output:  fmt.Sprintf("Wrote %d bytes to: %s", len(content), path),
	}, nil
}

func (t *FilesystemTool) readFile(path string) (*ToolResult, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return &ToolResult{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	return &ToolResult{
		Success: true,
		Output:  string(content),
		Data:    map[string]interface{}{"size": len(content)},
	}, nil
}

func (t *FilesystemTool) listFiles(path string) (*ToolResult, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return &ToolResult{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	var output strings.Builder
	files := []string{}

	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() {
			name += "/"
		}
		files = append(files, name)
		output.WriteString(name)
		output.WriteString("\n")
	}

	return &ToolResult{
		Success: true,
		Output:  output.String(),
		Data:    map[string]interface{}{"files": files, "count": len(files)},
	}, nil
}

func (t *FilesystemTool) appendFile(path, content string) (*ToolResult, error) {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return &ToolResult{
			Success: false,
			Error:   err.Error(),
		}, nil
	}
	defer f.Close()

	if _, err := f.WriteString(content); err != nil {
		return &ToolResult{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	return &ToolResult{
		Success: true,
		Output:  fmt.Sprintf("Appended %d bytes to: %s", len(content), path),
	}, nil
}
