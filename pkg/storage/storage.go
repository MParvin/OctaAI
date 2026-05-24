package storage

import (
	"time"
)

// State represents the agent's execution state
type State string

const (
	StateIdle               State = "IDLE"
	StatePlanning           State = "PLANNING"
	StateExecuting          State = "EXECUTING"
	StateEvaluating         State = "EVALUATING"
	StateRetrying           State = "RETRYING"
	StateWaitingForApproval State = "WAITING_FOR_APPROVAL"
	StateBlocked            State = "BLOCKED"
	StateCompleted          State = "COMPLETED"
	StateFailed             State = "FAILED"
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

// ExecutionStepRecord persists a deterministic execution step.
type ExecutionStepRecord struct {
	ID               string     `json:"id"`
	GoalID           string     `json:"goal_id"`
	TaskID           string     `json:"task_id,omitempty"`
	Description      string     `json:"description"`
	Status           string     `json:"status"`
	InputJSON        string     `json:"input_json"`
	OutputJSON       string     `json:"output_json,omitempty"`
	Validation       string     `json:"validation,omitempty"`
	ValidationDetail string     `json:"validation_detail,omitempty"`
	RetryCount       int        `json:"retry_count"`
	MaxRetries       int        `json:"max_retries"`
	StartedAt        *time.Time `json:"started_at,omitempty"`
	CompletedAt      *time.Time `json:"completed_at,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
	CheckpointID     string     `json:"checkpoint_id,omitempty"`
}

// CheckpointRecord persists a resumable execution snapshot.
type CheckpointRecord struct {
	ID        string    `json:"id"`
	GoalID    string    `json:"goal_id"`
	State     string    `json:"state"`
	StepIndex int       `json:"step_index"`
	Payload   string    `json:"payload"`
	CreatedAt time.Time `json:"created_at"`
}

// ApprovalRequest represents a pending human approval for a high-risk action.
type ApprovalRequest struct {
	ID           string     `json:"id"`
	GoalID       string     `json:"goal_id"`
	TaskID       string     `json:"task_id"`
	ToolName     string     `json:"tool_name"`
	ToolArgsJSON string     `json:"tool_args_json"`
	Fingerprint  string     `json:"fingerprint"`
	Reason       string     `json:"reason"`
	Status       string     `json:"status"` // pending, approved, denied
	CreatedAt    time.Time  `json:"created_at"`
	ResolvedAt   *time.Time `json:"resolved_at,omitempty"`
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

	// Execution step operations
	CreateStep(step *ExecutionStepRecord) error
	UpdateStep(step *ExecutionStepRecord) error
	GetStepsByGoal(goalID string) ([]ExecutionStepRecord, error)

	// Checkpoint operations
	CreateCheckpoint(cp *CheckpointRecord) error
	GetCheckpoint(id string) (*CheckpointRecord, error)
	GetCheckpointsByGoal(goalID string) ([]CheckpointRecord, error)

	// Approval operations
	CreateApproval(req *ApprovalRequest) error
	GetApproval(id string) (*ApprovalRequest, error)
	UpdateApproval(req *ApprovalRequest) error
	ListPendingApprovals() ([]ApprovalRequest, error)
	HasApprovedAction(goalID, fingerprint string) (bool, error)

	// Close closes the storage connection
	Close() error
}
