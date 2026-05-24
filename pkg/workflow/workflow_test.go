package workflow_test

import (
	"testing"

	"github.com/mparvin/octaai/pkg/workflow"
)

func TestValidateWorkflow(t *testing.T) {
	valid := []byte(`{
		"version": 1,
		"nodes": [
			{"id": "a", "description": "step a"},
			{"id": "b", "description": "step b", "dependencies": ["a"]}
		]
	}`)
	wf, err := workflow.Validate(valid)
	if err != nil {
		t.Fatalf("expected valid workflow, got %v", err)
	}
	if len(wf.Nodes) != 2 {
		t.Fatalf("expected 2 nodes, got %d", len(wf.Nodes))
	}
}

func TestValidateCycle(t *testing.T) {
	cyclic := []byte(`{
		"version": 1,
		"nodes": [
			{"id": "a", "description": "a", "dependencies": ["b"]},
			{"id": "b", "description": "b", "dependencies": ["a"]}
		]
	}`)
	_, err := workflow.Validate(cyclic)
	if err == nil {
		t.Fatal("expected cycle detection, got nil")
	}
}

func TestReadyNodes(t *testing.T) {
	wf := &workflow.Workflow{
		Nodes: []workflow.Node{
			{ID: "a", Description: "a"},
			{ID: "b", Description: "b", Dependencies: []string{"a"}},
		},
	}
	ready := workflow.ReadyNodes(wf, map[string]bool{})
	if len(ready) != 1 || ready[0].ID != "a" {
		t.Fatalf("expected node a ready, got %+v", ready)
	}
}
