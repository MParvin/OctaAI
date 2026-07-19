package core

import (
	"time"
)

// Goal represents a high-level user objective
type Goal struct {
	ID          string
	Description string
	ProjectName string
	State       GoalState
	CreatedAt   time.Time
	UpdatedAt   time.Time
	Metadata    map[string]interface{}
}

// GoalState represents the lifecycle state of a goal
type GoalState string

const (
	GoalStateIdle               GoalState = "IDLE"
	GoalStatePlanning           GoalState = "PLANNING"
	GoalStateExecuting          GoalState = "EXECUTING"
	GoalStateEvaluating         GoalState = "EVALUATING"
	GoalStateCompleted          GoalState = "COMPLETED"
	GoalStateFailed             GoalState = "FAILED"
	GoalStateRetrying           GoalState = "RETRYING"
	GoalStateWaitingForApproval GoalState = "WAITING_FOR_APPROVAL"
	GoalStateBlocked            GoalState = "BLOCKED"
)

// Task represents a unit of work in a goal
type Task struct {
	ID           string
	GoalID       string
	Description  string
	Status       TaskStatus
	Dependencies []string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// TaskStatus represents the execution state of a task
type TaskStatus string

const (
	TaskStatusPending   TaskStatus = "pending"
	TaskStatusRunning   TaskStatus = "running"
	TaskStatusCompleted TaskStatus = "completed"
	TaskStatusFailed    TaskStatus = "failed"
)

// Capability represents a first-class capability in the system
type Capability struct {
	ID               string
	Name             string
	Description      string
	RequiredTools    []string
	RequiredModels   []string
	Cost             CostTier
	RequiresApproval bool
	Constraints      *CapabilityConstraints
}

// CostTier represents the relative cost of a capability
type CostTier string

const (
	CostTierFree   CostTier = "free"
	CostTierLow    CostTier = "low"
	CostTierMedium CostTier = "medium"
	CostTierHigh   CostTier = "high"
)

// CapabilityConstraints defines resource and operational constraints
type CapabilityConstraints struct {
	MaxDuration      time.Duration
	MaxTokens        int
	RequiresInternet bool
	RequiresDocker   bool
}

// ExecutionResult represents the outcome of an execution
type ExecutionResult struct {
	Success          bool
	Status           string
	Duration         time.Duration
	TotalTokens      int64
	PromptTokens     int64
	CompletionTokens int64
	Cost             float64
	Error            error
	Metadata         map[string]interface{}
}

// NodeEstimate provides resource estimates for a node
type NodeEstimate struct {
	Tokens          int64
	DurationSeconds int64
	Cost            float64
	Confidence      float64 // 0.0 - 1.0
}

// ResourceRequirements defines what a tool or capability needs
type ResourceRequirements struct {
	Memory   int64         // Bytes
	CPU      float64       // Cores
	Disk     int64         // Bytes
	Network  bool          // Requires network access
	GPU      bool          // Requires GPU
	Duration time.Duration // Estimated duration
}
