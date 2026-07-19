package executor

import (
	"context"
	"errors"
	"testing"

	"github.com/mparvin/octaai/pkg/storage"
)

func TestSchedulerRespectsParallelLimitAndDeps(t *testing.T) {
	var ran []string
	s := &Scheduler{
		MaxParallel: 2,
		DepsMet: func(task *storage.Task) bool {
			return len(task.Dependencies) == 0
		},
		Run: func(ctx context.Context, task *storage.Task) error {
			ran = append(ran, task.ID)
			return nil
		},
	}
	tasks := []storage.Task{
		{ID: "a", Status: "pending"},
		{ID: "b", Status: "pending"},
		{ID: "c", Status: "pending", Dependencies: []string{"a"}},
	}
	if err := s.RunReady(context.Background(), tasks); err != nil {
		t.Fatal(err)
	}
	if len(ran) != 2 {
		t.Fatalf("expected 2 ready tasks, got %v", ran)
	}
}

func TestSchedulerPropagatesError(t *testing.T) {
	s := &Scheduler{
		MaxParallel: 1,
		Run: func(ctx context.Context, task *storage.Task) error {
			return errors.New("boom")
		},
	}
	err := s.RunReady(context.Background(), []storage.Task{{ID: "a", Status: "pending"}})
	if err == nil {
		t.Fatal("expected error")
	}
}
