package planner

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/mparvin/octaai/pkg/core"
	"github.com/mparvin/octaai/pkg/storage"
)

// HTNBridge adapts the HTN GraphPlanner to the v1 Planner interface used by the engine.
// On HTN failure it falls back to the template/LLM planner so goals still make progress.
type HTNBridge struct {
	htn         *HTNPlanner
	fallback    Planner
	caps        *core.CapabilityRegistry
	maxAttempts int
}

// NewHTNBridge creates a Planner backed by HTN with a v1 fallback.
func NewHTNBridge(htn *HTNPlanner, fallback Planner, caps *core.CapabilityRegistry, maxAttempts int) *HTNBridge {
	if maxAttempts <= 0 {
		maxAttempts = 3
	}
	return &HTNBridge{htn: htn, fallback: fallback, caps: caps, maxAttempts: maxAttempts}
}

// Plan uses HTN when possible, otherwise the fallback planner.
func (b *HTNBridge) Plan(ctx context.Context, goal *storage.Goal, memoryContext string) (*Plan, error) {
	coreGoal := &core.Goal{
		ID:          goal.ID,
		Description: goal.Description,
		ProjectName: goal.ProjectName,
		State:       core.GoalStatePlanning,
	}
	var caps []*core.Capability
	if b.caps != nil {
		caps = b.caps.List()
	}
	graph, err := b.htn.Plan(ctx, coreGoal, caps)
	if err != nil {
		if b.fallback != nil {
			return b.fallback.Plan(ctx, goal, memoryContext)
		}
		return nil, err
	}
	return GraphToPlan(graph, goal, b.maxAttempts), nil
}

// Replan prefers HTN corrective insertion when a failed task can be mapped; otherwise fallback.
func (b *HTNBridge) Replan(ctx context.Context, goal *storage.Goal, reason, memoryContext string, existing []storage.Task) ([]*storage.Task, error) {
	if b.fallback != nil {
		// Keep corrective tasks compatible with the v1 engine task loop.
		return b.fallback.Replan(ctx, goal, reason, memoryContext, existing)
	}
	return nil, fmt.Errorf("htn replan requires fallback planner")
}

// GraphToPlan converts an HTN execution graph into v1 storage tasks.
func GraphToPlan(graph *ExecutionGraph, goal *storage.Goal, maxAttempts int) *Plan {
	now := time.Now()
	plan := &Plan{
		ProjectName:  goal.ProjectName,
		NeedsProject: goal.ProjectName != "",
	}
	if maxAttempts <= 0 {
		maxAttempts = 3
	}

	idMap := make(map[string]string, len(graph.Nodes))
	for i, node := range graph.Nodes {
		taskID := fmt.Sprintf("task_%s_%d", goal.ID, i+1)
		idMap[node.ID] = taskID
	}

	depsByNode := make(map[string][]string)
	for _, edge := range graph.Edges {
		depsByNode[edge.To] = append(depsByNode[edge.To], edge.From)
	}
	for _, node := range graph.Nodes {
		depsByNode[node.ID] = append(depsByNode[node.ID], node.DependsOn...)
	}

	for i, node := range graph.Nodes {
		taskID := idMap[node.ID]
		var deps []string
		for _, from := range depsByNode[node.ID] {
			if mapped, ok := idMap[from]; ok {
				deps = append(deps, mapped)
			}
		}
		task := &storage.Task{
			ID:           taskID,
			GoalID:       goal.ID,
			Description:  node.Description,
			Status:       "pending",
			Dependencies: deps,
			CreatedAt:    now,
			UpdatedAt:    now,
			MaxAttempts:  maxAttempts,
		}
		if len(node.ToolSequence) == 1 {
			tool, args := splitHTNTool(node.ToolSequence[0])
			task.ToolName = tool
			task.ToolArgs = args
		} else if len(node.ToolSequence) > 1 {
			// First concrete tool; remaining work happens via LLM/engine if needed.
			tool, args := splitHTNTool(node.ToolSequence[0])
			task.ToolName = tool
			task.ToolArgs = args
			_ = i
		}
		plan.Tasks = append(plan.Tasks, task)
	}
	return plan
}

func splitHTNTool(tc *ToolCall) (string, map[string]interface{}) {
	args := make(map[string]interface{})
	if tc == nil {
		return "", args
	}
	for k, v := range tc.Args {
		args[k] = v
	}
	name := strings.TrimSpace(tc.Tool)
	if i := strings.Index(name, "."); i > 0 {
		tool := name[:i]
		action := name[i+1:]
		if _, ok := args["action"]; !ok && action != "" && !strings.Contains(action, ".") {
			args["action"] = action
		}
		return tool, args
	}
	return name, args
}
