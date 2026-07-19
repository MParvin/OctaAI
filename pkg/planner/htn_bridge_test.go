package planner

import (
	"testing"

	"github.com/mparvin/octaai/pkg/core"
	"github.com/mparvin/octaai/pkg/storage"
)

func TestGraphToPlanAndSplitTool(t *testing.T) {
	graph := &ExecutionGraph{
		ID:     "g",
		GoalID: "goal1",
		Nodes: []*TaskNode{
			{
				ID:          "n1",
				Description: "mkdir",
				Type:        NodeTypeSequential,
				Status:      core.TaskStatusPending,
				ToolSequence: []*ToolCall{
					{Tool: "filesystem.create_directory", Args: map[string]interface{}{"path": "app"}},
				},
			},
			{
				ID:          "n2",
				Description: "write",
				Type:        NodeTypeSequential,
				Status:      core.TaskStatusPending,
				DependsOn:   []string{"n1"},
				ToolSequence: []*ToolCall{
					{Tool: "filesystem", Args: map[string]interface{}{"action": "write_file", "path": "app/main.go"}},
				},
			},
		},
		Edges: []*Dependency{
			{From: "n1", To: "n2", Type: DependencyTypeSequential},
		},
		Metadata: &GraphMetadata{},
	}
	goal := &storage.Goal{ID: "goal1", ProjectName: "app"}
	plan := GraphToPlan(graph, goal, 3)
	if len(plan.Tasks) != 2 {
		t.Fatalf("tasks=%d", len(plan.Tasks))
	}
	if plan.Tasks[0].ToolName != "filesystem" {
		t.Fatalf("tool=%s", plan.Tasks[0].ToolName)
	}
	if plan.Tasks[0].ToolArgs["action"] != "create_directory" {
		t.Fatalf("args=%v", plan.Tasks[0].ToolArgs)
	}
	if len(plan.Tasks[1].Dependencies) == 0 {
		t.Fatal("expected dependency on first task")
	}
}

func TestSplitHTNTool(t *testing.T) {
	tool, args := splitHTNTool(&ToolCall{Tool: "command.execute", Args: map[string]interface{}{"command": "ls"}})
	if tool != "command" {
		t.Fatalf("tool=%s", tool)
	}
	if args["action"] != "execute" {
		t.Fatalf("args=%v", args)
	}
}
