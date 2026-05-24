package planner

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/mparvin/octaai/pkg/llm"
	"github.com/mparvin/octaai/pkg/memory"
	"github.com/mparvin/octaai/pkg/storage"
	"github.com/mparvin/octaai/pkg/tools"
)

// Plan is the output of the planner.
type Plan struct {
	ProjectName string         `json:"project_name,omitempty"`
	NeedsProject bool          `json:"needs_project"`
	Tasks       []*storage.Task `json:"tasks"`
}

// Planner creates actionable task plans from goals.
type Planner interface {
	Plan(ctx context.Context, goal *storage.Goal, memoryContext string) (*Plan, error)
	Replan(ctx context.Context, goal *storage.Goal, reason, memoryContext string, existing []storage.Task) ([]*storage.Task, error)
}

// LLMPlanner uses an LLM to decompose goals into tasks.
type LLMPlanner struct {
	llm    llm.Provider
	tools  *tools.Registry
	memory *memory.Manager
}

// NewLLMPlanner creates an LLM-backed planner.
func NewLLMPlanner(provider llm.Provider, registry *tools.Registry, mem *memory.Manager) *LLMPlanner {
	return &LLMPlanner{llm: provider, tools: registry, memory: mem}
}

// Plan generates tasks for a goal.
func (p *LLMPlanner) Plan(ctx context.Context, goal *storage.Goal, memoryContext string) (*Plan, error) {
	projectName, needsProject := p.extractProjectInfo(ctx, goal)

	now := time.Now()
	plan := &Plan{
		ProjectName:  projectName,
		NeedsProject: needsProject,
	}

	if needsProject && projectName != "" {
		plan.Tasks = append(plan.Tasks, &storage.Task{
			ID:           fmt.Sprintf("task_%s_1", goal.ID),
			GoalID:       goal.ID,
			Description:  fmt.Sprintf("Create project directory: %s", projectName),
			Status:       "pending",
			ToolName:     "filesystem",
			ToolArgs:     map[string]interface{}{"action": "create_directory", "path": projectName},
			CreatedAt:    now,
			UpdatedAt:    now,
			MaxAttempts:  3,
			Dependencies: []string{},
		})
		plan.Tasks = append(plan.Tasks, &storage.Task{
			ID:           fmt.Sprintf("task_%s_2", goal.ID),
			GoalID:       goal.ID,
			Description:  fmt.Sprintf("Develop project '%s': %s", projectName, goal.Description),
			Status:       "pending",
			CreatedAt:    now,
			UpdatedAt:    now,
			MaxAttempts:  3,
			Dependencies: []string{fmt.Sprintf("task_%s_1", goal.ID)},
		})
	} else {
		plan.Tasks = append(plan.Tasks, &storage.Task{
			ID:           fmt.Sprintf("task_%s_1", goal.ID),
			GoalID:       goal.ID,
			Description:  fmt.Sprintf("Complete goal: %s", goal.Description),
			Status:       "pending",
			CreatedAt:    now,
			UpdatedAt:    now,
			MaxAttempts:  3,
			Dependencies: []string{},
		})
	}

	if memoryContext != "" {
		p.memory.Remember(goal.ID, "plan_context", memoryContext, "planner")
	}

	return plan, nil
}

// Replan delegates to LLM replanning (see replan.go).
func (p *LLMPlanner) Replan(ctx context.Context, goal *storage.Goal, reason, memoryContext string, existing []storage.Task) ([]*storage.Task, error) {
	return p.replan(ctx, goal, reason, memoryContext, existing)
}

func (p *LLMPlanner) extractProjectInfo(ctx context.Context, goal *storage.Goal) (string, bool) {
	prompt := fmt.Sprintf(`Analyze this development goal:
"%s"

Does this goal require creating a new project/application (not just a single file)?
If YES, suggest a short project directory name (lowercase, no spaces, alphanumeric with hyphens/underscores only).
If NO (just a simple file creation), respond with "NONE".

Respond in this format:
PROJECT_NAME: <name or NONE>`, goal.Description)

	resp, err := p.llm.Complete(ctx, prompt, &llm.Options{Temperature: 0.1, MaxTokens: 100})
	if err != nil {
		return "", false
	}

	content := strings.TrimSpace(resp.Content)
	if strings.Contains(strings.ToUpper(content), "PROJECT_NAME:") {
		parts := strings.Split(content, ":")
		if len(parts) >= 2 {
			name := strings.TrimSpace(parts[1])
			if name != "NONE" && name != "" {
				name = strings.ToLower(strings.ReplaceAll(name, " ", "-"))
				return name, true
			}
		}
	}
	return "", false
}

// PlanFromWorkflow parses a validated workflow JSON into a plan.
func PlanFromWorkflow(goalID string, workflowJSON []byte) (*Plan, error) {
	var tasks []*storage.Task
	if err := json.Unmarshal(workflowJSON, &tasks); err != nil {
		return nil, fmt.Errorf("invalid workflow tasks: %w", err)
	}
	for _, t := range tasks {
		t.GoalID = goalID
	}
	return &Plan{Tasks: tasks}, nil
}
