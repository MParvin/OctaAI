package planner

import (
	"time"

	"github.com/mparvin/octaai/pkg/core"
)

// CostEstimator estimates execution costs for nodes
type CostEstimator struct {
	// Token cost per tool call (rough estimates)
	baseTokensPerTool    int64
	tokensPerComplexity  int64
	secondsPerComplexity int64
	costPerMillionTokens float64
}

// NewCostEstimator creates a new cost estimator
func NewCostEstimator() *CostEstimator {
	return &CostEstimator{
		baseTokensPerTool:    500,
		tokensPerComplexity:  200,
		secondsPerComplexity: 5,
		costPerMillionTokens: 0.0, // Free for local models
	}
}

// EstimateNode estimates costs for a task node
func (e *CostEstimator) EstimateNode(node *TaskNode) *core.NodeEstimate {
	// Base estimate
	tokens := e.baseTokensPerTool * int64(len(node.ToolSequence))

	// Add complexity multiplier (we don't have it in TaskNode, use tool count as proxy)
	complexity := len(node.ToolSequence)
	tokens += e.tokensPerComplexity * int64(complexity)

	// Estimate duration
	durationSeconds := e.secondsPerComplexity * int64(complexity)

	// Calculate cost
	cost := float64(tokens) / 1_000_000.0 * e.costPerMillionTokens

	return &core.NodeEstimate{
		Tokens:          tokens,
		DurationSeconds: durationSeconds,
		Cost:            cost,
		Confidence:      0.7, // Medium confidence in estimates
	}
}

// EstimateGraph calculates total costs for an execution graph
func (e *CostEstimator) EstimateGraph(graph *ExecutionGraph) {
	var totalTokens int64
	var totalDuration int64
	var totalCost float64

	for _, node := range graph.Nodes {
		if node.Estimate == nil {
			node.Estimate = e.EstimateNode(node)
		}
		totalTokens += node.Estimate.Tokens
		totalDuration += node.Estimate.DurationSeconds
		totalCost += node.Estimate.Cost
	}

	if graph.Metadata == nil {
		graph.Metadata = &GraphMetadata{}
	}

	graph.Metadata.TotalEstimatedTokens = totalTokens
	graph.Metadata.TotalEstimatedDuration = time.Duration(totalDuration) * time.Second
	graph.Metadata.TotalEstimatedCost = totalCost
}
