package engine_test

import (
	"testing"

	"github.com/mparvin/octaai/pkg/engine"
	"github.com/mparvin/octaai/pkg/storage"
)

func TestCanTransition(t *testing.T) {
	cases := []struct {
		from, to storage.State
		want     bool
	}{
		{storage.StateIdle, storage.StatePlanning, true},
		{storage.StatePlanning, storage.StateExecuting, true},
		{storage.StateExecuting, storage.StateEvaluating, true},
		{storage.StateEvaluating, storage.StateCompleted, true},
		{storage.StateEvaluating, storage.StateRetrying, true},
		{storage.StateExecuting, storage.StateWaitingForApproval, true},
		{storage.StateIdle, storage.StateCompleted, false},
		{storage.StateCompleted, storage.StatePlanning, false},
	}

	for _, tc := range cases {
		got := engine.CanTransition(tc.from, tc.to)
		if got != tc.want {
			t.Errorf("CanTransition(%s, %s) = %v, want %v", tc.from, tc.to, got, tc.want)
		}
	}
}

func TestExecutionStepRetry(t *testing.T) {
	step := engine.NewExecutionStep("s1", "g1", "test", engine.StepInput{})
	step.MaxRetries = 3

	if !step.CanRetry() {
		t.Fatal("expected CanRetry true initially")
	}

	step.IncrementRetry()
	if step.RetryCount != 1 {
		t.Fatalf("expected retry count 1, got %d", step.RetryCount)
	}
	if step.Status != engine.StepPending {
		t.Fatalf("expected pending after retry, got %s", step.Status)
	}
}
