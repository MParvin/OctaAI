package planner

import (
	"context"
	"fmt"

	"github.com/mparvin/octaai/pkg/core"
	"github.com/mparvin/octaai/pkg/llm"
)

// GraphPlanner interface defines the contract for graph-based planning systems
type GraphPlanner interface {
	// Plan generates an execution graph from a goal
	Plan(ctx context.Context, goal *core.Goal, capabilities []*core.Capability) (*ExecutionGraph, error)

	// Replan modifies an execution graph based on a failure
	Replan(ctx context.Context, graph *ExecutionGraph, failure *FailureReport) (*ExecutionGraph, error)

	// EstimateCost estimates the cost of executing a graph
	EstimateCost(graph *ExecutionGraph) (*CostEstimate, error)
}

// HTNPlanner implements Hierarchical Task Network planning
type HTNPlanner struct {
	llm                llm.Provider
	capabilityRegistry *core.CapabilityRegistry
	decomposer         *GoalDecomposer
	toolSelector       *ToolSelector
	costEstimator      *CostEstimator
}

// NewHTNPlanner creates a new HTN planner
func NewHTNPlanner(
	provider llm.Provider,
	capabilityRegistry *core.CapabilityRegistry,
) *HTNPlanner {
	return &HTNPlanner{
		llm:                provider,
		capabilityRegistry: capabilityRegistry,
		decomposer:         NewGoalDecomposer(provider),
		toolSelector:       NewToolSelector(provider),
		costEstimator:      NewCostEstimator(),
	}
}

// Plan generates an execution graph from a goal
func (p *HTNPlanner) Plan(ctx context.Context, goal *core.Goal, capabilities []*core.Capability) (*ExecutionGraph, error) {
	// Step 1: Decompose goal into abstract tasks
	abstractTasks, err := p.decomposer.Decompose(ctx, goal)
	if err != nil {
		return nil, fmt.Errorf("goal decomposition failed: %w", err)
	}

	// Step 2: Build execution graph
	graph := &ExecutionGraph{
		ID:     generateGraphID(goal.ID),
		GoalID: goal.ID,
		Nodes:  make([]*TaskNode, 0, len(abstractTasks)),
		Edges:  make([]*Dependency, 0),
		Metadata: &GraphMetadata{
			Version: "2.0",
		},
	}

	// Step 3: Create nodes for each abstract task
	for i, task := range abstractTasks {
		node, err := p.createNodeFromTask(ctx, task, capabilities)
		if err != nil {
			return nil, fmt.Errorf("failed to create node for task %d: %w", i, err)
		}
		graph.Nodes = append(graph.Nodes, node)

		// Add sequential dependencies for now (can be optimized later)
		if i > 0 {
			graph.Edges = append(graph.Edges, &Dependency{
				From: graph.Nodes[i-1].ID,
				To:   node.ID,
				Type: DependencyTypeSequential,
			})
		}
	}

	// Step 4: Estimate costs
	if err := p.estimateGraphCost(graph); err != nil {
		return nil, fmt.Errorf("cost estimation failed: %w", err)
	}

	// Step 5: Validate the graph
	if err := graph.Validate(); err != nil {
		return nil, fmt.Errorf("graph validation failed: %w", err)
	}

	return graph, nil
}

// createNodeFromTask creates a TaskNode from an abstract task description
func (p *HTNPlanner) createNodeFromTask(
	ctx context.Context,
	task *AbstractTask,
	capabilities []*core.Capability,
) (*TaskNode, error) {
	// Resolve capability
	capability, err := p.resolveCapability(ctx, task.Description, capabilities)
	if err != nil {
		return nil, err
	}

	// Select tools for this task
	toolSequence, err := p.toolSelector.SelectTools(ctx, task, capability)
	if err != nil {
		return nil, err
	}

	// Create validators based on capability
	validators := p.createValidators(capability)

	node := &TaskNode{
		ID:           task.ID,
		Description:  task.Description,
		Type:         task.Type,
		Capability:   capability.ID,
		ToolSequence: toolSequence,
		Validators:   validators,
		Status:       core.TaskStatusPending,
		MaxRetries:   3,
	}

	return node, nil
}

// resolveCapability maps a task description to a capability
func (p *HTNPlanner) resolveCapability(
	ctx context.Context,
	taskDescription string,
	capabilities []*core.Capability,
) (*core.Capability, error) {
	// For now, use simple keyword matching
	// TODO: Use LLM-based resolution
	for _, cap := range capabilities {
		if contains(taskDescription, "file") || contains(taskDescription, "directory") {
			if cap.ID == "coding.filesystem" {
				return cap, nil
			}
		}
		if contains(taskDescription, "git") || contains(taskDescription, "repository") {
			if cap.ID == "coding.git" {
				return cap, nil
			}
		}
		if contains(taskDescription, "command") || contains(taskDescription, "run") {
			if cap.ID == "coding.command" {
				return cap, nil
			}
		}
	}

	// Default to filesystem capability
	return &core.Capability{
		ID:            "coding.filesystem",
		Name:          "Filesystem Operations",
		RequiredTools: []string{"filesystem.*"},
		Cost:          core.CostTierFree,
	}, nil
}

// createValidators generates validators based on capability
func (p *HTNPlanner) createValidators(capability *core.Capability) []*Validator {
	var validators []*Validator

	// Add build validator for coding capabilities
	if contains(capability.ID, "coding") {
		validators = append(validators, &Validator{
			Type: ValidatorTypeBuild,
		})
	}

	return validators
}

// estimateGraphCost estimates costs for all nodes in the graph
func (p *HTNPlanner) estimateGraphCost(graph *ExecutionGraph) error {
	var totalTokens int64
	var totalCost float64

	for _, node := range graph.Nodes {
		estimate := p.costEstimator.EstimateNode(node)
		node.Estimate = estimate
		totalTokens += estimate.Tokens
		totalCost += estimate.Cost
	}

	graph.Metadata.TotalEstimatedTokens = totalTokens
	graph.Metadata.TotalEstimatedCost = totalCost

	return nil
}

// Replan inserts a corrective node before the failed node and revalidates the graph.
// This is a minimal adaptive strategy (not LLM-driven) until Phase 7 wires HTN into the engine.
func (p *HTNPlanner) Replan(ctx context.Context, graph *ExecutionGraph, failure *FailureReport) (*ExecutionGraph, error) {
	_ = ctx
	if graph == nil {
		return nil, fmt.Errorf("graph is required")
	}
	if failure == nil || failure.FailedNode == nil {
		return nil, fmt.Errorf("failure report with FailedNode is required")
	}

	failedID := failure.FailedNode.ID
	details := failure.Analysis
	if details == "" && len(failure.Suggestions) > 0 {
		details = failure.Suggestions[0].Details
	}
	if details == "" {
		details = fmt.Sprintf("corrective step after failure of %s (cause=%d)", failedID, failure.RootCause)
	}

	corrective := &TaskNode{
		ID:          fmt.Sprintf("%s_corrective_%d", failedID, len(graph.Nodes)+1),
		Description: details,
		Type:        NodeTypeSequential,
		Capability:  failure.FailedNode.Capability,
		Status:      core.TaskStatusPending,
		ToolSequence: []*ToolCall{
			{
				Tool: "command",
				Args: map[string]interface{}{
					"command": "true",
					"cwd":     ".",
				},
			},
		},
	}
	if failure.FailedNode.Capability != "" {
		corrective.Capability = failure.FailedNode.Capability
	}

	if err := graph.InsertBefore(failedID, corrective); err != nil {
		return nil, fmt.Errorf("insert corrective node: %w", err)
	}
	if err := p.estimateGraphCost(graph); err != nil {
		return nil, fmt.Errorf("cost estimation failed: %w", err)
	}
	if err := graph.Validate(); err != nil {
		return nil, fmt.Errorf("replanned graph invalid: %w", err)
	}
	return graph, nil
}

// EstimateCost estimates the cost of executing a graph
func (p *HTNPlanner) EstimateCost(graph *ExecutionGraph) (*CostEstimate, error) {
	return &CostEstimate{
		TotalTokens: graph.Metadata.TotalEstimatedTokens,
		TotalCost:   graph.Metadata.TotalEstimatedCost,
	}, nil
}

// CostEstimate represents the estimated cost of execution
type CostEstimate struct {
	TotalTokens int64
	TotalCost   float64
}

// FailureReport contains information about a failed node
type FailureReport struct {
	FailedNode  *TaskNode
	RootCause   RootCauseType
	Analysis    string
	Suggestions []*CorrectiveAction
}

// RootCauseType categorizes failure causes
type RootCauseType int

const (
	RootCauseMissingDependency RootCauseType = iota
	RootCauseIncorrectArguments
	RootCauseEnvironmentIssue
	RootCauseLogicError
	RootCauseTimeoutExceeded
	RootCausePermissionDenied
	RootCauseResourceExhausted
)

// CorrectiveAction suggests how to fix a failure
type CorrectiveAction struct {
	Type    string
	Details string
}

// Helper function to generate graph IDs
func generateGraphID(goalID string) string {
	return fmt.Sprintf("graph_%s", goalID)
}

// Helper function for substring search
func contains(s, substr string) bool {
	return len(s) >= len(substr) && indexOf(s, substr) >= 0
}

func indexOf(s, substr string) int {
	if len(substr) == 0 {
		return 0
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
