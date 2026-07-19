package planner

import (
	"context"
	"testing"

	"github.com/mparvin/octaai/pkg/core"
	"github.com/mparvin/octaai/pkg/llm"
)

func TestExecutionGraphValidation(t *testing.T) {
	tests := []struct {
		name    string
		graph   *ExecutionGraph
		wantErr bool
	}{
		{
			name: "valid graph with sequential nodes",
			graph: &ExecutionGraph{
				ID:     "graph_test_1",
				GoalID: "goal_test_1",
				Nodes: []*TaskNode{
					{
						ID:          "node_1",
						Description: "Setup project",
						Type:        NodeTypeSequential,
						Status:      core.TaskStatusPending,
					},
					{
						ID:          "node_2",
						Description: "Implement feature",
						Type:        NodeTypeSequential,
						Status:      core.TaskStatusPending,
					},
				},
				Edges: []*Dependency{
					{
						From: "node_1",
						To:   "node_2",
						Type: DependencyTypeSequential,
					},
				},
				Metadata: &GraphMetadata{},
			},
			wantErr: false,
		},
		{
			name: "empty graph",
			graph: &ExecutionGraph{
				ID:       "graph_test_2",
				GoalID:   "goal_test_2",
				Nodes:    []*TaskNode{},
				Edges:    []*Dependency{},
				Metadata: &GraphMetadata{},
			},
			wantErr: true,
		},
		{
			name: "duplicate node IDs",
			graph: &ExecutionGraph{
				ID:     "graph_test_3",
				GoalID: "goal_test_3",
				Nodes: []*TaskNode{
					{ID: "node_1", Status: core.TaskStatusPending},
					{ID: "node_1", Status: core.TaskStatusPending},
				},
				Metadata: &GraphMetadata{},
			},
			wantErr: true,
		},
		{
			name: "invalid dependency - non-existent node",
			graph: &ExecutionGraph{
				ID:     "graph_test_4",
				GoalID: "goal_test_4",
				Nodes: []*TaskNode{
					{ID: "node_1", Status: core.TaskStatusPending},
				},
				Edges: []*Dependency{
					{From: "node_1", To: "node_nonexistent"},
				},
				Metadata: &GraphMetadata{},
			},
			wantErr: true,
		},
		{
			name: "cycle detection",
			graph: &ExecutionGraph{
				ID:     "graph_test_5",
				GoalID: "goal_test_5",
				Nodes: []*TaskNode{
					{ID: "node_1", Status: core.TaskStatusPending},
					{ID: "node_2", Status: core.TaskStatusPending},
					{ID: "node_3", Status: core.TaskStatusPending},
				},
				Edges: []*Dependency{
					{From: "node_1", To: "node_2"},
					{From: "node_2", To: "node_3"},
					{From: "node_3", To: "node_1"}, // Creates cycle
				},
				Metadata: &GraphMetadata{},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.graph.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGetReadyNodes(t *testing.T) {
	graph := &ExecutionGraph{
		ID:     "graph_test",
		GoalID: "goal_test",
		Nodes: []*TaskNode{
			{ID: "node_1", Status: core.TaskStatusPending},
			{ID: "node_2", Status: core.TaskStatusPending},
			{ID: "node_3", Status: core.TaskStatusPending},
			{ID: "node_4", Status: core.TaskStatusCompleted},
		},
		Edges: []*Dependency{
			{From: "node_1", To: "node_2"},
			{From: "node_2", To: "node_3"},
			{From: "node_4", To: "node_1"},
		},
		Metadata: &GraphMetadata{},
	}

	// Initially, only node_1 should be ready (depends on completed node_4)
	ready := graph.GetReadyNodes()
	if len(ready) != 1 || ready[0].ID != "node_1" {
		t.Errorf("GetReadyNodes() = %v, want [node_1]", nodeIDs(ready))
	}

	// Complete node_1, node_2 should become ready
	graph.Nodes[0].Status = core.TaskStatusCompleted
	ready = graph.GetReadyNodes()
	if len(ready) != 1 || ready[0].ID != "node_2" {
		t.Errorf("GetReadyNodes() = %v, want [node_2]", nodeIDs(ready))
	}

	// Complete node_2, node_3 should become ready
	graph.Nodes[1].Status = core.TaskStatusCompleted
	ready = graph.GetReadyNodes()
	if len(ready) != 1 || ready[0].ID != "node_3" {
		t.Errorf("GetReadyNodes() = %v, want [node_3]", nodeIDs(ready))
	}
}

func TestHTNReplanInsertsCorrectiveNode(t *testing.T) {
	p := &HTNPlanner{costEstimator: NewCostEstimator()}
	graph := &ExecutionGraph{
		ID:     "graph_replan",
		GoalID: "goal_replan",
		Nodes: []*TaskNode{
			{ID: "node_1", Description: "setup", Type: NodeTypeSequential, Status: core.TaskStatusCompleted},
			{ID: "node_2", Description: "build", Type: NodeTypeSequential, Status: core.TaskStatusFailed, Capability: "coding.command"},
		},
		Edges: []*Dependency{
			{From: "node_1", To: "node_2", Type: DependencyTypeSequential},
		},
		Metadata: &GraphMetadata{Version: "2.0"},
	}
	out, err := p.Replan(context.Background(), graph, &FailureReport{
		FailedNode: graph.Nodes[1],
		RootCause:  RootCauseEnvironmentIssue,
		Analysis:   "install missing build deps",
	})
	if err != nil {
		t.Fatalf("Replan: %v", err)
	}
	if len(out.Nodes) != 3 {
		t.Fatalf("expected 3 nodes after replan, got %d", len(out.Nodes))
	}
	found := false
	for _, n := range out.Nodes {
		if n.Description == "install missing build deps" {
			found = true
		}
	}
	if !found {
		t.Fatal("expected corrective node with failure analysis description")
	}
}

func TestInsertBefore(t *testing.T) {
	graph := &ExecutionGraph{
		ID:     "graph_test",
		GoalID: "goal_test",
		Nodes: []*TaskNode{
			{ID: "node_1", Status: core.TaskStatusPending},
			{ID: "node_2", Status: core.TaskStatusPending},
			{ID: "node_3", Status: core.TaskStatusPending},
		},
		Edges: []*Dependency{
			{From: "node_1", To: "node_2"},
			{From: "node_2", To: "node_3"},
		},
		Metadata: &GraphMetadata{},
	}

	newNode := &TaskNode{
		ID:          "node_1_5",
		Description: "Intermediate step",
		Status:      core.TaskStatusPending,
	}

	// Insert new node between node_1 and node_2
	err := graph.InsertBefore("node_2", newNode)
	if err != nil {
		t.Fatalf("InsertBefore() error = %v", err)
	}

	// Verify graph structure
	if len(graph.Nodes) != 4 {
		t.Errorf("Expected 4 nodes, got %d", len(graph.Nodes))
	}

	// Verify edges: node_1 -> node_1_5 -> node_2
	expectedEdges := map[string]string{
		"node_1":   "node_1_5",
		"node_1_5": "node_2",
		"node_2":   "node_3",
	}

	for _, edge := range graph.Edges {
		if expectedTo, exists := expectedEdges[edge.From]; exists {
			if edge.To != expectedTo {
				t.Errorf("Edge from %s should point to %s, got %s", edge.From, expectedTo, edge.To)
			}
		}
	}
}

// Helper function to extract node IDs
func nodeIDs(nodes []*TaskNode) []string {
	ids := make([]string, len(nodes))
	for i, node := range nodes {
		ids[i] = node.ID
	}
	return ids
}

// Mock LLM provider for testing
type mockLLMProvider struct{}

func (m *mockLLMProvider) Complete(ctx context.Context, prompt string, opts *llm.Options) (*llm.Response, error) {
	return &llm.Response{
		Content: `{"tasks": [{"id": "task_1", "description": "Setup", "type": "sequential", "complexity": 3}]}`,
		Usage: &llm.Usage{
			PromptTokens:     100,
			CompletionTokens: 50,
		},
	}, nil
}

func (m *mockLLMProvider) Name() string {
	return "mock"
}

func (m *mockLLMProvider) Model() string {
	return "mock-model"
}

func TestGoalDecomposer(t *testing.T) {
	decomposer := NewGoalDecomposer(&mockLLMProvider{})

	goal := &core.Goal{
		ID:          "goal_test",
		Description: "Create a simple Flask application",
	}

	tasks, err := decomposer.Decompose(context.Background(), goal)
	if err != nil {
		t.Fatalf("Decompose() error = %v", err)
	}

	if len(tasks) == 0 {
		t.Error("Expected at least one task")
	}

	if tasks[0].ID == "" {
		t.Error("Task ID should not be empty")
	}
}
