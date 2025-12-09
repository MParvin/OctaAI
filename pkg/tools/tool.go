package tools

import (
	"context"
)

// Tool defines the interface for all tools
type Tool interface {
	// Name returns the tool name
	Name() string

	// Schema returns the tool schema for LLM
	Schema() ToolSchema

	// Execute runs the tool with given arguments
	Execute(ctx context.Context, args map[string]interface{}) (*ToolResult, error)
}

// ToolSchema describes the tool for the LLM
type ToolSchema struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]ParamSchema `json:"parameters"`
}

// ParamSchema describes a parameter
type ParamSchema struct {
	Type        string   `json:"type"` // "string", "number", "boolean", "array", "object"
	Description string   `json:"description"`
	Required    bool     `json:"required"`
	Enum        []string `json:"enum,omitempty"`
}

// ToolResult contains the result of tool execution
type ToolResult struct {
	Success bool        `json:"success"`
	Output  string      `json:"output"`
	Error   string      `json:"error,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

// Registry manages available tools
type Registry struct {
	tools map[string]Tool
}

// NewRegistry creates a new tool registry
func NewRegistry() *Registry {
	return &Registry{
		tools: make(map[string]Tool),
	}
}

// Register adds a tool to the registry
func (r *Registry) Register(tool Tool) {
	r.tools[tool.Name()] = tool
}

// Get retrieves a tool by name
func (r *Registry) Get(name string) (Tool, bool) {
	tool, ok := r.tools[name]
	return tool, ok
}

// List returns all registered tools
func (r *Registry) List() []Tool {
	tools := make([]Tool, 0, len(r.tools))
	for _, tool := range r.tools {
		tools = append(tools, tool)
	}
	return tools
}

// Schemas returns schemas for all tools
func (r *Registry) Schemas() []ToolSchema {
	schemas := make([]ToolSchema, 0, len(r.tools))
	for _, tool := range r.tools {
		schemas = append(schemas, tool.Schema())
	}
	return schemas
}
