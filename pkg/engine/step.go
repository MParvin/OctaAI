package engine

import (
	"encoding/json"
	"time"

	"github.com/mparvin/octaai/pkg/execution"
)

// StepInput captures everything needed to run a deterministic step.
type StepInput struct {
	ToolName string                 `json:"tool_name"`
	ToolArgs map[string]interface{} `json:"tool_args"`
	Metadata map[string]string      `json:"metadata,omitempty"`
}

// StepOutput captures the result of a step execution.
type StepOutput struct {
	Success bool                   `json:"success"`
	Output  string                 `json:"output"`
	Error   string                 `json:"error,omitempty"`
	Data    map[string]interface{} `json:"data,omitempty"`
}

// ExecutionStep is the atomic unit of work tracked by the engine.
type ExecutionStep struct {
	ID               string           `json:"id"`
	GoalID           string           `json:"goal_id"`
	TaskID           string           `json:"task_id,omitempty"`
	Description      string           `json:"description"`
	Status           StepStatus       `json:"status"`
	Input            StepInput        `json:"input"`
	Output           *StepOutput      `json:"output,omitempty"`
	Validation       ValidationResult `json:"validation,omitempty"`
	ValidationDetail string           `json:"validation_detail,omitempty"`
	RetryCount       int              `json:"retry_count"`
	MaxRetries       int              `json:"max_retries"`
	StartedAt        *time.Time       `json:"started_at,omitempty"`
	CompletedAt      *time.Time       `json:"completed_at,omitempty"`
	CreatedAt        time.Time        `json:"created_at"`
	UpdatedAt        time.Time        `json:"updated_at"`
	CheckpointID     string           `json:"checkpoint_id,omitempty"`
}

// NewExecutionStep creates a step with defaults.
func NewExecutionStep(id, goalID, description string, input StepInput) *ExecutionStep {
	now := time.Now()
	return &ExecutionStep{
		ID:          id,
		GoalID:      goalID,
		Description: description,
		Status:      StepPending,
		Input:       input,
		MaxRetries:  3,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

// MarkRunning transitions the step to running.
func (s *ExecutionStep) MarkRunning() {
	now := time.Now()
	s.Status = StepRunning
	s.StartedAt = &now
	s.UpdatedAt = now
}

// MarkCompleted records a successful step result.
func (s *ExecutionStep) MarkCompleted(output *StepOutput, validation ValidationResult, detail string) {
	now := time.Now()
	s.Status = StepCompleted
	s.Output = output
	s.Validation = validation
	s.ValidationDetail = detail
	s.CompletedAt = &now
	s.UpdatedAt = now
}

// MarkFailed records a failed step.
func (s *ExecutionStep) MarkFailed(output *StepOutput, validation ValidationResult, detail string) {
	now := time.Now()
	s.Status = StepFailed
	s.Output = output
	s.Validation = validation
	s.ValidationDetail = detail
	s.CompletedAt = &now
	s.UpdatedAt = now
}

// CanRetry reports whether the step may be retried.
func (s *ExecutionStep) CanRetry() bool {
	return s.RetryCount < s.MaxRetries
}

// IncrementRetry bumps retry count and resets status to pending.
func (s *ExecutionStep) IncrementRetry() {
	s.RetryCount++
	s.Status = StepPending
	s.StartedAt = nil
	s.CompletedAt = nil
	s.UpdatedAt = time.Now()
}

// Snapshot serializes step state for checkpoints.
func (s *ExecutionStep) Snapshot() ([]byte, error) {
	return json.Marshal(s)
}

// View returns a read-only snapshot for evaluators.
func (s *ExecutionStep) View() *execution.StepView {
	view := &execution.StepView{
		ID:               s.ID,
		GoalID:           s.GoalID,
		TaskID:           s.TaskID,
		Description:      s.Description,
		Status:           s.Status,
		ToolName:         s.Input.ToolName,
		ToolArgs:         s.Input.ToolArgs,
		Validation:       s.Validation,
		ValidationDetail: s.ValidationDetail,
		RetryCount:       s.RetryCount,
		MaxRetries:       s.MaxRetries,
	}
	if s.Output != nil {
		view.Output = s.Output.Output
		view.Error = s.Output.Error
		view.Success = s.Output.Success
	}
	return view
}
