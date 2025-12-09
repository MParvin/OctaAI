package storage

import (
	"time"
)

// State represents the agent's execution state
type State string

const (
	StateIdle      State = "IDLE"
	StatePlanning  State = "PLANNING"
	StateExecuting State = "EXECUTING"
	StateBlocked   State = "BLOCKED"
	StateCompleted State = "COMPLETED"
	StateFailed    State = "FAILED"
)

// Goal represents a high-level objective
type Goal struct {
	ID          string     `json:"id"`
	Description string     `json:"description"`
	State       State      `json:"state"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	Result      string     `json:"result,omitempty"`
	Error       string     `json:"error,omitempty"`
}

// Task represents a specific actionable step
type Task struct {
	ID           string                 `json:"id"`
	GoalID       string                 `json:"goal_id"`
	Description  string                 `json:"description"`
	Status       string                 `json:"status"` // "pending", "running", "completed", "failed"
	Dependencies []string               `json:"dependencies"`
	ToolName     string                 `json:"tool_name,omitempty"`
	ToolArgs     map[string]interface{} `json:"tool_args,omitempty"`
	Result       string                 `json:"result,omitempty"`
	Error        string                 `json:"error,omitempty"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
	Attempts     int                    `json:"attempts"`
	MaxAttempts  int                    `json:"max_attempts"`
}

// ExecutionLog represents a log entry for debugging
type ExecutionLog struct {
	ID        string    `json:"id"`
	GoalID    string    `json:"goal_id"`
	TaskID    string    `json:"task_id,omitempty"`
	Level     string    `json:"level"` // "info", "warning", "error", "debug"
	Message   string    `json:"message"`
	Data      string    `json:"data,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// Storage defines the interface for persisting agent state
type Storage interface {
	// Goal operations
	CreateGoal(goal *Goal) error
	GetGoal(id string) (*Goal, error)
	UpdateGoal(goal *Goal) error
	ListGoals() ([]*Goal, error)

	// Task operations
	CreateTask(task *Task) error
	GetTask(id string) (*Task, error)
	UpdateTask(task *Task) error
	GetTasksByGoal(goalID string) ([]Task, error)

	// Log operations
	CreateLog(log *ExecutionLog) error
	GetLogsByGoal(goalID string) ([]*ExecutionLog, error)

	// Close closes the storage connection
	Close() error
}
