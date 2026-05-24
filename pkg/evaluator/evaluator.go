package evaluator

import (
	"context"
	"fmt"
	"strings"

	"github.com/mparvin/octaai/pkg/execution"
)

// Verdict is the evaluator outcome for a step or goal.
type Verdict struct {
	Result execution.ValidationResult `json:"result"`
	Detail string                     `json:"detail"`
}

const (
	Success       = execution.ValidationSuccess
	Partial       = execution.ValidationPartial
	RetryRequired = execution.ValidationRetryRequired
	Fatal         = execution.ValidationFatal
)

// Evaluator determines whether execution output satisfies expectations.
type Evaluator interface {
	Evaluate(ctx context.Context, step *execution.StepView) (*Verdict, error)
}

// CompositeEvaluator runs multiple evaluators in sequence.
type CompositeEvaluator struct {
	evaluators      []Evaluator
	goalEvaluators  []GoalEvaluator
}

// NewComposite creates an evaluator chain.
func NewComposite(evaluators ...Evaluator) *CompositeEvaluator {
	c := &CompositeEvaluator{evaluators: evaluators}
	for _, ev := range evaluators {
		if ge, ok := ev.(GoalEvaluator); ok {
			c.goalEvaluators = append(c.goalEvaluators, ge)
		}
	}
	return c
}

// Evaluate runs all evaluators; first non-success wins.
func (c *CompositeEvaluator) Evaluate(ctx context.Context, step *execution.StepView) (*Verdict, error) {
	for _, ev := range c.evaluators {
		v, err := ev.Evaluate(ctx, step)
		if err != nil {
			return nil, err
		}
		if v.Result != Success {
			return v, nil
		}
	}
	return &Verdict{Result: Success, Detail: "all checks passed"}, nil
}

// EvaluateGoal delegates to goal-level evaluators.
func (c *CompositeEvaluator) EvaluateGoal(ctx context.Context, goalDescription string, steps []*execution.StepView) (*Verdict, error) {
	for _, ev := range c.goalEvaluators {
		v, err := ev.EvaluateGoal(ctx, goalDescription, steps)
		if err != nil {
			return nil, err
		}
		if v.Result != Success {
			return v, nil
		}
	}
	return &Verdict{Result: Success, Detail: "goal satisfied"}, nil
}

// GoalEvaluator can evaluate full goal completion.
type GoalEvaluator interface {
	EvaluateGoal(ctx context.Context, goalDescription string, steps []*execution.StepView) (*Verdict, error)
}

// ToolResultEvaluator checks basic tool execution success.
type ToolResultEvaluator struct{}

// Evaluate checks step output for tool-level success.
func (e *ToolResultEvaluator) Evaluate(_ context.Context, step *execution.StepView) (*Verdict, error) {
	if !step.Success && step.Error == "" && step.Output == "" {
		return &Verdict{Result: RetryRequired, Detail: "no output recorded"}, nil
	}
	if !step.Success {
		if step.CanRetry() {
			return &Verdict{Result: RetryRequired, Detail: step.Error}, nil
		}
		return &Verdict{Result: Fatal, Detail: step.Error}, nil
	}
	return &Verdict{Result: Success}, nil
}

// BuildResultEvaluator inspects command output for build failures.
type BuildResultEvaluator struct{}

// Evaluate detects build/compile errors in command output.
func (e *BuildResultEvaluator) Evaluate(_ context.Context, step *execution.StepView) (*Verdict, error) {
	if step.ToolName != "command" {
		return &Verdict{Result: Success}, nil
	}
	out := strings.ToLower(step.Output + step.Error)
	failPatterns := []string{"build failed", "compilation error", "syntax error", "cannot find module"}
	for _, p := range failPatterns {
		if strings.Contains(out, p) {
			if step.CanRetry() {
				return &Verdict{Result: RetryRequired, Detail: "build failure detected: " + p}, nil
			}
			return &Verdict{Result: Fatal, Detail: "build failure: " + p}, nil
		}
	}
	return &Verdict{Result: Success}, nil
}

// TestResultEvaluator inspects test command output.
type TestResultEvaluator struct{}

// Evaluate detects test failures.
func (e *TestResultEvaluator) Evaluate(_ context.Context, step *execution.StepView) (*Verdict, error) {
	if step.ToolName != "command" {
		return &Verdict{Result: Success}, nil
	}
	cmd, _ := step.ToolArgs["command"].(string)
	if !strings.Contains(strings.ToLower(cmd), "test") {
		return &Verdict{Result: Success}, nil
	}
	out := strings.ToLower(step.Output)
	if strings.Contains(out, "fail") || strings.Contains(out, "error") {
		if step.CanRetry() {
			return &Verdict{Result: RetryRequired, Detail: "test failures detected"}, nil
		}
		return &Verdict{Result: Fatal, Detail: "tests failed after max retries"}, nil
	}
	return &Verdict{Result: Success}, nil
}

// GoalCompletionEvaluator verifies all steps completed successfully.
type GoalCompletionEvaluator struct{}

func (e *GoalCompletionEvaluator) Evaluate(_ context.Context, _ *execution.StepView) (*Verdict, error) {
	return &Verdict{Result: Success}, nil
}

func (e *GoalCompletionEvaluator) EvaluateGoal(_ context.Context, _ string, steps []*execution.StepView) (*Verdict, error) {
	if len(steps) == 0 {
		return &Verdict{Result: Success, Detail: "no steps to evaluate"}, nil
	}
	for _, step := range steps {
		if step.Status != execution.StepCompleted {
			return &Verdict{Result: RetryRequired, Detail: fmt.Sprintf("step %s not completed", step.ID)}, nil
		}
		if step.Validation == Fatal {
			return &Verdict{Result: Fatal, Detail: step.ValidationDetail}, nil
		}
	}
	return &Verdict{Result: Success, Detail: "all steps completed"}, nil
}

// DefaultEvaluator returns the standard evaluator chain.
func DefaultEvaluator() *CompositeEvaluator {
	return NewComposite(
		&ToolResultEvaluator{},
		&BuildResultEvaluator{},
		&TestResultEvaluator{},
		&GoalCompletionEvaluator{},
	)
}
