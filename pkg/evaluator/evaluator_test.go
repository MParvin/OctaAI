package evaluator

import (
	"context"
	"testing"

	"github.com/mparvin/octaai/pkg/execution"
)

func TestToolResultEvaluatorSuccessfulStep(t *testing.T) {
	ev := &ToolResultEvaluator{}
	step := &execution.StepView{
		Success:  true,
		Output:   "Wrote 20 bytes to: /tmp/foo.py",
		ToolName: "filesystem",
	}

	verdict, err := ev.Evaluate(context.Background(), step)
	if err != nil {
		t.Fatal(err)
	}
	if verdict.Result != Success {
		t.Fatalf("expected success, got %q (%s)", verdict.Result, verdict.Detail)
	}
}

func TestToolResultEvaluatorEmptyUnsuccessfulStep(t *testing.T) {
	ev := &ToolResultEvaluator{}
	step := &execution.StepView{
		Success: false,
	}

	verdict, err := ev.Evaluate(context.Background(), step)
	if err != nil {
		t.Fatal(err)
	}
	if verdict.Result != RetryRequired || verdict.Detail != "no output recorded" {
		t.Fatalf("expected retry for empty step, got %q (%s)", verdict.Result, verdict.Detail)
	}
}
