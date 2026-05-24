package execution

// ValidationResult captures evaluator output for a step.
type ValidationResult string

const (
	ValidationSuccess       ValidationResult = "success"
	ValidationPartial       ValidationResult = "partial_success"
	ValidationRetryRequired ValidationResult = "retry_required"
	ValidationFatal         ValidationResult = "fatal_failure"
)

// StepStatus represents the lifecycle of a single execution step.
type StepStatus string

const (
	StepPending    StepStatus = "pending"
	StepRunning    StepStatus = "running"
	StepCompleted  StepStatus = "completed"
	StepFailed     StepStatus = "failed"
	StepSkipped    StepStatus = "skipped"
	StepRolledBack StepStatus = "rolled_back"
)

// StepView is a read-only step snapshot for evaluation.
type StepView struct {
	ID               string
	GoalID           string
	TaskID           string
	Description      string
	Status           StepStatus
	ToolName         string
	ToolArgs         map[string]interface{}
	Output           string
	Error            string
	Success          bool
	Validation       ValidationResult
	ValidationDetail string
	RetryCount       int
	MaxRetries       int
}

// CanRetry reports whether the step may be retried.
func (s *StepView) CanRetry() bool {
	return s.RetryCount < s.MaxRetries
}
