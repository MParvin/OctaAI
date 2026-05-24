package engine

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/mparvin/octaai/pkg/config"
	"github.com/mparvin/octaai/pkg/isolation"
	"github.com/mparvin/octaai/pkg/observability"
	"github.com/mparvin/octaai/pkg/permission"
	"github.com/mparvin/octaai/pkg/tools"
)

// Runner executes tools with permission checks, timeouts, and isolation hooks.
type Runner struct {
	cfg        *config.Config
	tools      *tools.Registry
	permission *permission.Manager
	logger     *observability.Logger
	docker     *isolation.Docker
}

// RunConfig controls tool execution constraints.
type RunConfig struct {
	Timeout time.Duration
	GoalID  string
	TaskID  string
	StepID  string
}

// NewRunner creates a tool runner.
func NewRunner(cfg *config.Config, registry *tools.Registry, perm *permission.Manager, logger *observability.Logger) *Runner {
	return &Runner{
		cfg:        cfg,
		tools:      registry,
		permission: perm,
		logger:     logger,
		docker:     isolation.NewDocker(cfg),
	}
}

// Run executes a tool after permission checks and optional Docker isolation.
func (r *Runner) Run(ctx context.Context, toolName string, args map[string]interface{}, cfg RunConfig) (*StepOutput, error) {
	span := r.logger.StartSpan(cfg.GoalID, fmt.Sprintf("tool:%s", toolName), observability.Fields{"step_id": cfg.StepID})

	execArgs := args
	if wrapped, ok, err := r.docker.WrapArgs(toolName, args); err != nil {
		r.logger.EndSpan(span)
		return nil, err
	} else if ok {
		execArgs = wrapped
	}

	check := r.permission.CheckTool(cfg.GoalID, cfg.TaskID, toolName, execArgs)
	switch check.Decision {
	case permission.DecisionDeny:
		r.logger.EndSpan(span)
		return &StepOutput{Success: false, Error: check.Reason}, nil
	case permission.DecisionRequireApproval:
		r.logger.EndSpan(span)
		return nil, fmt.Errorf("approval_required: %s", check.Reason)
	}

	tool, ok := r.tools.Get(toolName)
	if !ok {
		r.logger.EndSpan(span)
		return nil, fmt.Errorf("tool not found: %s", toolName)
	}

	runCtx := ctx
	if cfg.Timeout > 0 {
		var cancel context.CancelFunc
		runCtx, cancel = context.WithTimeout(ctx, cfg.Timeout)
		defer cancel()
	}

	result, err := tool.Execute(runCtx, execArgs)
	r.logger.EndSpan(span)

	if err != nil {
		return &StepOutput{Success: false, Error: err.Error()}, nil
	}

	out := &StepOutput{
		Success: result.Success,
		Output:  result.Output,
		Error:   result.Error,
	}
	if data, ok := result.Data.(map[string]interface{}); ok {
		out.Data = data
	}
	if execArgs["_isolated"] == true {
		if out.Data == nil {
			out.Data = map[string]interface{}{}
		}
		out.Data["isolated"] = true
	}

	return out, nil
}

// RunParallel executes multiple tool calls concurrently with a limit.
func (r *Runner) RunParallel(ctx context.Context, calls []parallelCall, maxParallel int) ([]parallelResult, error) {
	if maxParallel <= 0 {
		maxParallel = 1
	}
	sem := make(chan struct{}, maxParallel)
	results := make([]parallelResult, len(calls))
	var wg sync.WaitGroup

	for i, call := range calls {
		wg.Add(1)
		go func(idx int, c parallelCall) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			out, err := r.Run(ctx, c.ToolName, c.Args, RunConfig{
				Timeout: c.Timeout,
				GoalID:  c.GoalID,
				TaskID:  c.TaskID,
				StepID:  c.StepID,
			})
			results[idx] = parallelResult{Output: out, Err: err}
		}(i, call)
	}
	wg.Wait()
	return results, nil
}

type parallelCall struct {
	ToolName string
	Args     map[string]interface{}
	GoalID   string
	TaskID   string
	StepID   string
	Timeout  time.Duration
}

type parallelResult struct {
	Output *StepOutput
	Err    error
}
