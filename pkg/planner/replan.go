package planner

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/mparvin/octaai/pkg/llm"
	"github.com/mparvin/octaai/pkg/storage"
)

// Replan generates additional tasks when execution needs correction.
func (p *LLMPlanner) replan(ctx context.Context, goal *storage.Goal, reason, memoryContext string, existing []storage.Task) ([]*storage.Task, error) {
	var completed, pending strings.Builder
	for _, t := range existing {
		line := fmt.Sprintf("- [%s] %s\n", t.Status, t.Description)
		if t.Status == "completed" {
			completed.WriteString(line)
		} else {
			pending.WriteString(line)
		}
	}

	prompt := fmt.Sprintf(`Goal: %s

Execution needs replanning because: %s

Completed tasks:
%s
Remaining/failed tasks:
%s
%s

Suggest ONE specific next task to move toward the goal.
Respond with ONLY the task description (one line, no markdown).`, goal.Description, reason, completed.String(), pending.String(), memoryContext)

	resp, err := p.llm.Complete(ctx, prompt, &llm.Options{Temperature: 0.2, MaxTokens: 200})
	if err != nil {
		return nil, err
	}

	desc := strings.TrimSpace(resp.Content)
	if desc == "" {
		return nil, fmt.Errorf("replan produced empty task")
	}

	deps := []string{}
	for _, t := range existing {
		if t.Status == "completed" {
			deps = append(deps, t.ID)
		}
	}

	now := time.Now()
	task := &storage.Task{
		ID:           fmt.Sprintf("task_%s_replan_%d", goal.ID, time.Now().Unix()),
		GoalID:       goal.ID,
		Description:  desc,
		Status:       "pending",
		CreatedAt:    now,
		UpdatedAt:    now,
		MaxAttempts:  3,
		Dependencies: deps,
	}

	p.memory.Remember(goal.ID, "replan", desc, "planner")
	return []*storage.Task{task}, nil
}
