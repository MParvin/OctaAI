package workflow_test

import (
	"context"
	"testing"

	"github.com/mparvin/octaai/pkg/workflow"
)

func TestParallelExecutor(t *testing.T) {
	wf := &workflow.Workflow{
		Nodes: []workflow.Node{
			{ID: "a", Description: "a", Parallel: true},
			{ID: "b", Description: "b", Parallel: true},
			{ID: "c", Description: "c", Dependencies: []string{"a", "b"}},
		},
	}

	order := make(chan string, 3)
	exec := &workflow.Executor{
		MaxParallel: 2,
		RunNode: func(_ context.Context, node workflow.Node) error {
			order <- node.ID
			return nil
		},
	}

	if err := exec.Execute(context.Background(), wf); err != nil {
		t.Fatal(err)
	}
	close(order)

	seen := map[string]bool{}
	for id := range order {
		seen[id] = true
	}
	if !seen["a"] || !seen["b"] || !seen["c"] {
		t.Fatalf("expected all nodes executed, got %v", seen)
	}
}
