package workflow

import (
	"encoding/json"
	"fmt"
)

// Node represents a step in an execution graph.
type Node struct {
	ID           string                 `json:"id"`
	Description  string                 `json:"description"`
	ToolName     string                 `json:"tool_name,omitempty"`
	ToolArgs     map[string]interface{} `json:"tool_args,omitempty"`
	Dependencies []string               `json:"dependencies"`
	Condition    string                 `json:"condition,omitempty"`
	Parallel     bool                   `json:"parallel,omitempty"`
}

// Workflow is a validated execution graph.
type Workflow struct {
	Version int    `json:"version"`
	GoalID  string `json:"goal_id,omitempty"`
	Nodes   []Node `json:"nodes"`
}

// ValidationError describes a schema validation failure.
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// Validate checks workflow structure before execution.
func Validate(data []byte) (*Workflow, error) {
	var wf Workflow
	if err := json.Unmarshal(data, &wf); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}
	if err := validateWorkflow(&wf); err != nil {
		return nil, err
	}
	return &wf, nil
}

func validateWorkflow(wf *Workflow) error {
	if len(wf.Nodes) == 0 {
		return ValidationError{Field: "nodes", Message: "workflow must contain at least one node"}
	}

	ids := make(map[string]bool, len(wf.Nodes))
	for _, n := range wf.Nodes {
		if n.ID == "" {
			return ValidationError{Field: "id", Message: "node id is required"}
		}
		if ids[n.ID] {
			return ValidationError{Field: "id", Message: fmt.Sprintf("duplicate node id: %s", n.ID)}
		}
		ids[n.ID] = true
		if n.Description == "" {
			return ValidationError{Field: "description", Message: fmt.Sprintf("node %s missing description", n.ID)}
		}
	}

	for _, n := range wf.Nodes {
		for _, dep := range n.Dependencies {
			if !ids[dep] {
				return ValidationError{
					Field:   "dependencies",
					Message: fmt.Sprintf("node %s references unknown dependency %s", n.ID, dep),
				}
			}
		}
	}

	if hasCycle(wf.Nodes) {
		return ValidationError{Field: "dependencies", Message: "workflow contains a dependency cycle"}
	}

	return nil
}

func hasCycle(nodes []Node) bool {
	graph := make(map[string][]string)
	for _, n := range nodes {
		graph[n.ID] = n.Dependencies
	}

	visited := make(map[string]bool)
	inStack := make(map[string]bool)

	var visit func(id string) bool
	visit = func(id string) bool {
		if inStack[id] {
			return true
		}
		if visited[id] {
			return false
		}
		visited[id] = true
		inStack[id] = true
		for _, dep := range graph[id] {
			if visit(dep) {
				return true
			}
		}
		inStack[id] = false
		return false
	}

	for id := range graph {
		if visit(id) {
			return true
		}
	}
	return false
}

// ReadyNodes returns nodes whose dependencies are satisfied.
func ReadyNodes(wf *Workflow, completed map[string]bool) []Node {
	var ready []Node
	for _, n := range wf.Nodes {
		if completed[n.ID] {
			continue
		}
		depsMet := true
		for _, dep := range n.Dependencies {
			if !completed[dep] {
				depsMet = false
				break
			}
		}
		if depsMet {
			ready = append(ready, n)
		}
	}
	return ready
}
