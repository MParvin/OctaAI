package executor

import (
	"context"
	"sync"

	"github.com/mparvin/octaai/pkg/storage"
)

// TaskRunner executes a single storage task.
type TaskRunner func(ctx context.Context, task *storage.Task) error

// Scheduler runs pending tasks in dependency-aware parallel batches (DAG executor).
type Scheduler struct {
	MaxParallel int
	Run         TaskRunner
	DepsMet     func(task *storage.Task) bool
}

// RunReady executes up to MaxParallel ready pending tasks.
func (s *Scheduler) RunReady(ctx context.Context, tasks []storage.Task) error {
	if s.Run == nil {
		return nil
	}
	if s.MaxParallel <= 0 {
		s.MaxParallel = 1
	}
	depsMet := s.DepsMet
	if depsMet == nil {
		depsMet = func(*storage.Task) bool { return true }
	}

	var ready []*storage.Task
	for i := range tasks {
		t := &tasks[i]
		if t.Status == "pending" && depsMet(t) {
			ready = append(ready, t)
		}
	}
	if len(ready) == 0 {
		return nil
	}
	if len(ready) > s.MaxParallel {
		ready = ready[:s.MaxParallel]
	}
	if len(ready) == 1 {
		return s.Run(ctx, ready[0])
	}

	var wg sync.WaitGroup
	errCh := make(chan error, len(ready))
	sem := make(chan struct{}, s.MaxParallel)
	for _, task := range ready {
		wg.Add(1)
		go func(t *storage.Task) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			if err := s.Run(ctx, t); err != nil {
				errCh <- err
			}
		}(task)
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		if err != nil {
			return err
		}
	}
	return nil
}
