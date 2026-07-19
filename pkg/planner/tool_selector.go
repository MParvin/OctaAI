package planner

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mparvin/octaai/pkg/core"
	"github.com/mparvin/octaai/pkg/llm"
)

// ToolSelector selects appropriate tools for abstract tasks
type ToolSelector struct {
	llm llm.Provider
}

// NewToolSelector creates a new tool selector
func NewToolSelector(provider llm.Provider) *ToolSelector {
	return &ToolSelector{
		llm: provider,
	}
}

// SelectTools chooses tools for a task based on capability
func (t *ToolSelector) SelectTools(
	ctx context.Context,
	task *AbstractTask,
	capability *core.Capability,
) ([]*ToolCall, error) {
	prompt := t.buildToolSelectionPrompt(task, capability)

	resp, err := t.llm.Complete(ctx, prompt, &llm.Options{
		Temperature: 0.2,
		MaxTokens:   1000,
	})
	if err != nil {
		return nil, fmt.Errorf("LLM completion failed: %w", err)
	}

	tools, err := t.parseToolSelectionResponse(resp.Content)
	if err != nil {
		return nil, fmt.Errorf("failed to parse tool selection: %w", err)
	}

	return tools, nil
}

// buildToolSelectionPrompt creates the prompt for tool selection
func (t *ToolSelector) buildToolSelectionPrompt(task *AbstractTask, capability *core.Capability) string {
	return fmt.Sprintf(`Select the appropriate tools to accomplish this task.

Task: %s
Capability: %s
Available Tools: %v

Choose 1-3 tools in the order they should be executed. Provide specific arguments.

Respond in JSON format:
{
  "tools": [
    {
      "tool": "filesystem.create_directory",
      "args": {"path": "~/octaai-projects/my-project"}
    },
    {
      "tool": "filesystem.write_file",
      "args": {"path": "~/octaai-projects/my-project/main.py", "content": "..."}
    }
  ]
}

Respond with ONLY the JSON, no markdown or explanation.`,
		task.Description,
		capability.ID,
		capability.RequiredTools,
	)
}

// parseToolSelectionResponse extracts tool calls from LLM response
func (t *ToolSelector) parseToolSelectionResponse(content string) ([]*ToolCall, error) {
	jsonContent := extractJSON(content)

	var response struct {
		Tools []struct {
			Tool string                 `json:"tool"`
			Args map[string]interface{} `json:"args"`
		} `json:"tools"`
	}

	if err := json.Unmarshal([]byte(jsonContent), &response); err != nil {
		return nil, fmt.Errorf("JSON parsing failed: %w", err)
	}

	if len(response.Tools) == 0 {
		return nil, fmt.Errorf("no tools found in response")
	}

	tools := make([]*ToolCall, 0, len(response.Tools))
	for _, tool := range response.Tools {
		toolCall := &ToolCall{
			Tool: tool.Tool,
			Args: tool.Args,
		}
		tools = append(tools, toolCall)
	}

	return tools, nil
}
