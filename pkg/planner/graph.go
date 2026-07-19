package planner

import (
	"fmt"
	"time"

	"github.com/mparvin/octaai/pkg/core"
)

// ExecutionGraph represents a DAG of tasks for goal execution
type ExecutionGraph struct {
	ID         string
	GoalID     string
	Nodes      []*TaskNode
	Edges      []*Dependency
	Strategies []*FallbackStrategy
	Metadata   *GraphMetadata
	CreatedAt  time.Time
}

// TaskNode represents a node in the execution graph
type TaskNode struct {
	ID           string
	Description  string
	Type         NodeType
	Capability   string // e.g., "coding.filesystem"
	ToolSequence []*ToolCall
	Validators   []*Validator
	Fallback     *TaskNode
	Estimate     *core.NodeEstimate
	DependsOn    []string
	Status       core.TaskStatus
	RetryCount   int
	MaxRetries   int
}

// NodeType defines the execution behavior of a node
type NodeType int

const (
	NodeTypeSequential         NodeType = iota // Execute tools in order
	NodeTypeParallel                           // Execute tools concurrently
	NodeTypeConditional                        // If-then-else branching
	NodeTypeLoop                               // Repeat until condition
	NodeTypeApproval                           // Human gate
	NodeTypeAgentCollaboration                 // Multi-agent sub-task
)

func (nt NodeType) String() string {
	switch nt {
	case NodeTypeSequential:
		return "Sequential"
	case NodeTypeParallel:
		return "Parallel"
	case NodeTypeConditional:
		return "Conditional"
	case NodeTypeLoop:
		return "Loop"
	case NodeTypeApproval:
		return "Approval"
	case NodeTypeAgentCollaboration:
		return "AgentCollaboration"
	default:
		return "Unknown"
	}
}

// ToolCall represents a specific tool invocation
type ToolCall struct {
	Tool      string
	Args      map[string]interface{}
	Condition *Condition // Optional: execute if condition met
}

// Condition defines when a tool call should execute
type Condition struct {
	Type  ConditionType
	Field string
	Value interface{}
}

// ConditionType defines the type of condition
type ConditionType int

const (
	ConditionTypeEquals ConditionType = iota
	ConditionTypeNotEquals
	ConditionTypeContains
	ConditionTypeGreaterThan
	ConditionTypeLessThan
)

// Validator defines a validation step after task execution
type Validator struct {
	Type           ValidatorType
	Tool           string // Tool to run for validation (e.g., "command.run_tests")
	Args           map[string]interface{}
	ExpectedOutput string
}

// ValidatorType defines the kind of validation
type ValidatorType int

const (
	ValidatorTypeBuild ValidatorType = iota
	ValidatorTypeTest
	ValidatorTypeLint
	ValidatorTypeCustom
)

// Dependency represents an edge between nodes
type Dependency struct {
	From string // Node ID
	To   string // Node ID
	Type DependencyType
}

// DependencyType defines the relationship between nodes
type DependencyType int

const (
	DependencyTypeSequential  DependencyType = iota // To executes after From
	DependencyTypeConditional                       // To executes if From succeeds
	DependencyTypeParallel                          // To can execute concurrently with From
)

// FallbackStrategy defines alternative approaches when a node fails
type FallbackStrategy struct {
	NodeID      string
	Alternative *TaskNode
	Condition   string // When to trigger fallback
}

// GraphMetadata contains metadata about the execution graph
type GraphMetadata struct {
	TotalEstimatedTokens   int64
	TotalEstimatedDuration time.Duration
	TotalEstimatedCost     float64
	CreatedBy              string
	Version                string
}

// Validate checks if the execution graph is valid
func (g *ExecutionGraph) Validate() error {
	if len(g.Nodes) == 0 {
		return fmt.Errorf("execution graph must have at least one node")
	}

	// Check for duplicate node IDs
	nodeIDs := make(map[string]bool)
	for _, node := range g.Nodes {
		if nodeIDs[node.ID] {
			return fmt.Errorf("duplicate node ID: %s", node.ID)
		}
		nodeIDs[node.ID] = true
	}

	// Validate dependencies
	for _, edge := range g.Edges {
		if !nodeIDs[edge.From] {
			return fmt.Errorf("dependency references non-existent node: %s", edge.From)
		}
		if !nodeIDs[edge.To] {
			return fmt.Errorf("dependency references non-existent node: %s", edge.To)
		}
	}

	// Check for cycles
	if err := g.detectCycles(); err != nil {
		return err
	}

	return nil
}

// detectCycles uses DFS to detect cycles in the graph
func (g *ExecutionGraph) detectCycles() error {
	visited := make(map[string]bool)
	recStack := make(map[string]bool)

	for _, node := range g.Nodes {
		if !visited[node.ID] {
			if g.hasCycleDFS(node.ID, visited, recStack) {
				return fmt.Errorf("cycle detected in execution graph")
			}
		}
	}

	return nil
}

func (g *ExecutionGraph) hasCycleDFS(nodeID string, visited, recStack map[string]bool) bool {
	visited[nodeID] = true
	recStack[nodeID] = true

	// Get all nodes that depend on this node
	for _, edge := range g.Edges {
		if edge.From == nodeID {
			if !visited[edge.To] {
				if g.hasCycleDFS(edge.To, visited, recStack) {
					return true
				}
			} else if recStack[edge.To] {
				return true
			}
		}
	}

	recStack[nodeID] = false
	return false
}

// GetNode retrieves a node by ID
func (g *ExecutionGraph) GetNode(id string) (*TaskNode, error) {
	for _, node := range g.Nodes {
		if node.ID == id {
			return node, nil
		}
	}
	return nil, fmt.Errorf("node not found: %s", id)
}

// GetDependencies returns all nodes that a given node depends on
func (g *ExecutionGraph) GetDependencies(nodeID string) []*TaskNode {
	var deps []*TaskNode
	for _, edge := range g.Edges {
		if edge.To == nodeID {
			if node, err := g.GetNode(edge.From); err == nil {
				deps = append(deps, node)
			}
		}
	}
	return deps
}

// GetReadyNodes returns nodes that have all dependencies satisfied
func (g *ExecutionGraph) GetReadyNodes() []*TaskNode {
	var ready []*TaskNode

	for _, node := range g.Nodes {
		if node.Status == core.TaskStatusPending {
			deps := g.GetDependencies(node.ID)
			allComplete := true
			for _, dep := range deps {
				if dep.Status != core.TaskStatusCompleted {
					allComplete = false
					break
				}
			}
			if allComplete {
				ready = append(ready, node)
			}
		}
	}

	return ready
}

// InsertBefore inserts a new node before an existing node in the graph
func (g *ExecutionGraph) InsertBefore(existingNodeID string, newNode *TaskNode) error {
	// Find the existing node
	existingNode, err := g.GetNode(existingNodeID)
	if err != nil {
		return err
	}

	// Add the new node
	g.Nodes = append(g.Nodes, newNode)

	// Find all edges that point to the existing node
	var edgesToUpdate []*Dependency
	for _, edge := range g.Edges {
		if edge.To == existingNodeID {
			edgesToUpdate = append(edgesToUpdate, edge)
		}
	}

	// Redirect those edges to the new node
	for _, edge := range edgesToUpdate {
		edge.To = newNode.ID
	}

	// Create an edge from new node to existing node
	g.Edges = append(g.Edges, &Dependency{
		From: newNode.ID,
		To:   existingNode.ID,
		Type: DependencyTypeSequential,
	})

	return nil
}
