package workflow

import (
	"context"
	"sync"
)

// Executor runs workflow nodes with dependency-aware parallelism.
type Executor struct {
	MaxParallel int
	RunNode     func(ctx context.Context, node Node) error
}

// Execute runs all nodes respecting dependencies and parallel flags.
func (e *Executor) Execute(ctx context.Context, wf *Workflow) error {
	if e.MaxParallel <= 0 {
		e.MaxParallel = 3
	}
	if e.RunNode == nil {
		return nil
	}

	completed := make(map[string]bool)
	var completedMu sync.Mutex
	var execErr error
	var errMu sync.Mutex

	for len(completed) < len(wf.Nodes) {
		if execErr != nil {
			return execErr
		}

		ready := ReadyNodes(wf, completed)
		if len(ready) == 0 {
			break
		}

		var batch []Node
		for _, n := range ready {
			batch = append(batch, n)
			if !n.Parallel || len(batch) >= e.MaxParallel {
				break
			}
		}

		if len(batch) == 1 || !anyParallel(batch) {
			for _, n := range batch {
				if err := e.RunNode(ctx, n); err != nil {
					return err
				}
				completedMu.Lock()
				completed[n.ID] = true
				completedMu.Unlock()
			}
			continue
		}

		sem := make(chan struct{}, e.MaxParallel)
		var wg sync.WaitGroup
		for _, n := range batch {
			wg.Add(1)
			go func(node Node) {
				defer wg.Done()
				sem <- struct{}{}
				defer func() { <-sem }()

				if err := e.RunNode(ctx, node); err != nil {
					errMu.Lock()
					if execErr == nil {
						execErr = err
					}
					errMu.Unlock()
					return
				}
				completedMu.Lock()
				completed[node.ID] = true
				completedMu.Unlock()
			}(n)
		}
		wg.Wait()
	}

	if execErr != nil {
		return execErr
	}
	if len(completed) < len(wf.Nodes) {
		return ErrIncompleteExecution
	}
	return nil
}

func anyParallel(nodes []Node) bool {
	for _, n := range nodes {
		if n.Parallel {
			return true
		}
	}
	return false
}
