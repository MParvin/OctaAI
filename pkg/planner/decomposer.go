package planner

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mparvin/octaai/pkg/core"
	"github.com/mparvin/octaai/pkg/llm"
)

// GoalDecomposer breaks down goals into abstract tasks
type GoalDecomposer struct {
	llm llm.Provider
}

// NewGoalDecomposer creates a new goal decomposer
func NewGoalDecomposer(provider llm.Provider) *GoalDecomposer {
	return &GoalDecomposer{
		llm: provider,
	}
}

// AbstractTask represents a high-level task before tool selection
type AbstractTask struct {
	ID                  string
	Description         string
	Type                NodeType
	EstimatedComplexity int // 1-10
}

// Decompose breaks a goal into abstract tasks
func (d *GoalDecomposer) Decompose(ctx context.Context, goal *core.Goal) ([]*AbstractTask, error) {
	prompt := d.buildDecompositionPrompt(goal)

	resp, err := d.llm.Complete(ctx, prompt, &llm.Options{
		Temperature: 0.3,
		MaxTokens:   2000,
	})
	if err != nil {
		return nil, fmt.Errorf("LLM completion failed: %w", err)
	}

	tasks, err := d.parseDecompositionResponse(resp.Content)
	if err != nil {
		return nil, fmt.Errorf("failed to parse decomposition response: %w", err)
	}

	return tasks, nil
}

// buildDecompositionPrompt creates the LLM prompt for goal decomposition
func (d *GoalDecomposer) buildDecompositionPrompt(goal *core.Goal) string {
	return fmt.Sprintf(`You are a task planner for an AI agent system. Break down this goal into 3-7 concrete, actionable tasks.

Goal: %s

Guidelines:
1. Create sequential steps that build toward the goal
2. Each task should be specific and measurable
3. Tasks should be independent where possible (for parallel execution)
4. Keep tasks at a high level (tool selection happens later)
5. Estimate complexity for each task (1-10 scale)

Respond in JSON format:
{
  "tasks": [
    {
      "id": "task_1",
      "description": "Setup project structure",
      "type": "sequential",
      "complexity": 3
    },
    {
      "id": "task_2",
      "description": "Implement core functionality",
      "type": "sequential",
      "complexity": 8
    }
  ]
}

Respond with ONLY the JSON, no markdown or explanation.`, goal.Description)
}

// parseDecompositionResponse extracts tasks from LLM response
func (d *GoalDecomposer) parseDecompositionResponse(content string) ([]*AbstractTask, error) {
	// Try to extract JSON from markdown code blocks
	jsonContent := extractJSON(content)

	var response struct {
		Tasks []struct {
			ID          string `json:"id"`
			Description string `json:"description"`
			Type        string `json:"type"`
			Complexity  int    `json:"complexity"`
		} `json:"tasks"`
	}

	if err := json.Unmarshal([]byte(jsonContent), &response); err != nil {
		return nil, fmt.Errorf("JSON parsing failed: %w (content: %s)", err, jsonContent)
	}

	if len(response.Tasks) == 0 {
		return nil, fmt.Errorf("no tasks found in response")
	}

	tasks := make([]*AbstractTask, 0, len(response.Tasks))
	for _, t := range response.Tasks {
		task := &AbstractTask{
			ID:                  t.ID,
			Description:         t.Description,
			Type:                parseNodeType(t.Type),
			EstimatedComplexity: t.Complexity,
		}
		tasks = append(tasks, task)
	}

	return tasks, nil
}

// extractJSON attempts to extract JSON from markdown code blocks
func extractJSON(content string) string {
	// Try to find JSON in ```json blocks
	start := -1
	end := -1

	// Look for ```json or ```
	for i := 0; i < len(content)-3; i++ {
		if content[i:i+3] == "```" {
			if start == -1 {
				// Found opening ```
				start = i + 3
				// Skip "json" if present
				if start < len(content)-4 && content[start:start+4] == "json" {
					start += 4
				}
				// Skip newline
				if start < len(content) && content[start] == '\n' {
					start++
				}
			} else {
				// Found closing ```
				end = i
				break
			}
		}
	}

	if start != -1 && end != -1 {
		return content[start:end]
	}

	// If no code blocks found, try to find JSON object
	for i := 0; i < len(content); i++ {
		if content[i] == '{' {
			// Found start of JSON
			return content[i:]
		}
	}

	return content
}

// parseNodeType converts string type to NodeType
func parseNodeType(typeStr string) NodeType {
	switch typeStr {
	case "sequential":
		return NodeTypeSequential
	case "parallel":
		return NodeTypeParallel
	case "conditional":
		return NodeTypeConditional
	case "loop":
		return NodeTypeLoop
	case "approval":
		return NodeTypeApproval
	default:
		return NodeTypeSequential
	}
}
