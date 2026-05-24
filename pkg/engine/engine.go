package engine

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/mparvin/octaai/pkg/config"
	"github.com/mparvin/octaai/pkg/evaluator"
	"github.com/mparvin/octaai/pkg/execution"
	"github.com/mparvin/octaai/pkg/llm"
	"github.com/mparvin/octaai/pkg/memory"
	"github.com/mparvin/octaai/pkg/observability"
	"github.com/mparvin/octaai/pkg/permission"
	"github.com/mparvin/octaai/pkg/planner"
	"github.com/mparvin/octaai/pkg/storage"
	"github.com/mparvin/octaai/pkg/tools"
)

// EngineConfig holds engine limits and safety settings.
type EngineConfig struct {
	MaxLoops       int
	MaxRetries     int
	MaxParallel    int
	StepTimeout    time.Duration
	LoopPause      time.Duration
	EnableReplan   bool
	EnableParallel bool
}

// DefaultEngineConfig returns sensible defaults.
func DefaultEngineConfig() EngineConfig {
	return EngineConfig{
		MaxLoops:       50,
		MaxRetries:     3,
		MaxParallel:    3,
		StepTimeout:    5 * time.Minute,
		LoopPause:      500 * time.Millisecond,
		EnableReplan:   true,
		EnableParallel: true,
	}
}

// Engine is the deterministic execution engine with state machine semantics.
type Engine struct {
	cfg        *config.Config
	engineCfg  EngineConfig
	store      storage.Storage
	llm        llm.Provider
	tools      *tools.Registry
	planner    planner.Planner
	evaluator  *evaluator.CompositeEvaluator
	runner     *Runner
	memory     *memory.Manager
	permission *permission.Manager
	logger     *observability.Logger
	goalID     string
	loopCount  int
}

// NewEngine wires all engine components.
func NewEngine(
	cfg *config.Config,
	llmProvider llm.Provider,
	toolRegistry *tools.Registry,
	store storage.Storage,
) *Engine {
	mem := memory.NewManager(store)
	logger := observability.NewLogger(store)
	perm := permission.NewManager(cfg, store)
	pl := planner.NewLLMPlanner(llmProvider, toolRegistry, mem)

	ecfg := DefaultEngineConfig()
	if cfg.Engine.MaxLoops > 0 {
		ecfg.MaxLoops = cfg.Engine.MaxLoops
	}
	if cfg.Engine.MaxRetries > 0 {
		ecfg.MaxRetries = cfg.Engine.MaxRetries
	}
	ecfg.EnableReplan = cfg.Engine.EnableReplan
	ecfg.EnableParallel = cfg.Engine.EnableParallel
	if cfg.Isolation.MaxParallel > 0 {
		ecfg.MaxParallel = cfg.Isolation.MaxParallel
	}

	return &Engine{
		cfg:        cfg,
		engineCfg:  ecfg,
		store:      store,
		llm:        llmProvider,
		tools:      toolRegistry,
		planner:    pl,
		evaluator:  evaluator.DefaultEvaluator(),
		runner:     NewRunner(cfg, toolRegistry, perm, logger),
		memory:     mem,
		permission: perm,
		logger:     logger,
	}
}

// ProcessGoal runs a goal through the state machine until completion or failure.
func (e *Engine) ProcessGoal(ctx context.Context, goalID string) error {
	e.goalID = goalID
	e.loopCount = 0

	goal, err := e.store.GetGoal(goalID)
	if err != nil {
		return fmt.Errorf("failed to get goal: %w", err)
	}

	e.logger.RecordTimeline(observability.TimelineEvent{
		Timestamp: time.Now(),
		GoalID:    goalID,
		Event:     "goal_started",
		Detail:    goal.Description,
	})

	if err := e.transitionGoal(goal, StatePlanning); err != nil {
		return err
	}

	existingTasks, _ := e.store.GetTasksByGoal(goalID)
	if len(existingTasks) == 0 {
		memCtx := e.memory.SearchContext(goalID, goal.Description, 10)
		plan, err := e.planner.Plan(ctx, goal, memCtx)
		if err != nil {
			return e.failGoal(goal, err)
		}
		for _, task := range plan.Tasks {
			if err := e.store.CreateTask(task); err != nil {
				return e.failGoal(goal, err)
			}
		}
		e.logger.Info(goalID, fmt.Sprintf("Created %d task(s)", len(plan.Tasks)), nil)
	} else {
		e.logger.Info(goalID, fmt.Sprintf("Resuming with %d existing task(s)", len(existingTasks)), nil)
	}

	if err := e.transitionGoal(goal, StateExecuting); err != nil {
		return err
	}

executionLoop:
	for e.loopCount < e.engineCfg.MaxLoops {
		e.loopCount++

		tasks, err := e.store.GetTasksByGoal(goalID)
		if err != nil {
			return err
		}

		if e.detectStall(tasks) {
			return e.failGoal(goal, fmt.Errorf("no progress after %d iterations", e.loopCount))
		}

		allDone, hasFatal := e.summarizeTasks(tasks)
		if hasFatal {
			return e.failGoal(goal, fmt.Errorf("tasks failed after max attempts"))
		}
		if allDone {
			break
		}

		if e.engineCfg.EnableParallel {
			if err := e.executeReadyTasks(ctx, goal); err != nil {
				if isApprovalRequired(err) {
					return err
				}
				e.logger.Error(goalID, fmt.Sprintf("Task execution error: %v", err), nil)
			}
		} else if err := e.executeNextTask(ctx, goal); err != nil {
			if isApprovalRequired(err) {
				return err
			}
			e.logger.Error(goalID, fmt.Sprintf("Task execution error: %v", err), nil)
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(e.engineCfg.LoopPause):
		}
	}

	if e.loopCount >= e.engineCfg.MaxLoops {
		return e.failGoal(goal, fmt.Errorf("exceeded maximum loop count"))
	}

	if err := e.transitionGoal(goal, StateEvaluating); err != nil {
		return err
	}

	steps, _ := e.store.GetStepsByGoal(goalID)
	stepPtrs := make([]*execution.StepView, len(steps))
	for i := range steps {
		stepPtrs[i] = stepFromStorage(&steps[i]).View()
	}

	verdict, err := e.evaluator.EvaluateGoal(ctx, goal.Description, stepPtrs)
	if err != nil {
		return e.failGoal(goal, err)
	}

	switch verdict.Result {
	case evaluator.RetryRequired:
		if e.engineCfg.EnableReplan {
			if err := e.transitionGoal(goal, StateRetrying); err != nil {
				return err
			}
			tasks, _ := e.store.GetTasksByGoal(goalID)
			newTasks, replanErr := e.planner.Replan(ctx, goal, verdict.Detail, e.memory.SearchContext(goalID, goal.Description, 5), tasks)
			if replanErr == nil {
				for _, t := range newTasks {
					_ = e.store.CreateTask(t)
				}
				e.logger.Info(goalID, fmt.Sprintf("Replanned with %d new task(s)", len(newTasks)), nil)
				if err := e.transitionGoal(goal, StateExecuting); err != nil {
					return err
				}
				e.loopCount = 0
				goto executionLoop
			}
			e.logger.Warn(goalID, fmt.Sprintf("Replan failed: %v", replanErr), nil)
		}
		return e.failGoal(goal, fmt.Errorf("retry required: %s", verdict.Detail))
	case evaluator.Fatal:
		return e.failGoal(goal, fmt.Errorf("evaluation failed: %s", verdict.Detail))
	}

	now := time.Now()
	goal.State = StateCompleted
	goal.UpdatedAt = now
	goal.CompletedAt = &now
	goal.Result = "Goal completed successfully"
	e.logger.RecordTimeline(observability.TimelineEvent{
		Timestamp: now,
		GoalID:    goalID,
		Event:     "goal_completed",
		Detail:    goal.Result,
	})
	return e.store.UpdateGoal(goal)
}

func (e *Engine) transitionGoal(goal *storage.Goal, to GoalState) error {
	if !CanTransition(goal.State, to) {
		return fmt.Errorf("invalid state transition: %s -> %s", goal.State, to)
	}
	goal.State = to
	goal.UpdatedAt = time.Now()
	e.logger.RecordTimeline(observability.TimelineEvent{
		Timestamp: time.Now(),
		GoalID:    goal.ID,
		Event:     "state_transition",
		Detail:    fmt.Sprintf("%s", to),
	})
	return e.store.UpdateGoal(goal)
}

func (e *Engine) executeReadyTasks(ctx context.Context, goal *storage.Goal) error {
	tasks, err := e.store.GetTasksByGoal(goal.ID)
	if err != nil {
		return err
	}

	var ready []*storage.Task
	for i := range tasks {
		task := &tasks[i]
		if task.Status == "pending" && e.dependenciesMet(task) {
			ready = append(ready, task)
		}
	}
	if len(ready) == 0 {
		return nil
	}

	limit := e.engineCfg.MaxParallel
	if limit <= 0 {
		limit = 1
	}
	if len(ready) > limit {
		ready = ready[:limit]
	}

	if len(ready) == 1 {
		return e.runTask(ctx, goal, ready[0])
	}

	var wg sync.WaitGroup
	errCh := make(chan error, len(ready))
	for _, task := range ready {
		wg.Add(1)
		go func(t *storage.Task) {
			defer wg.Done()
			if err := e.runTask(ctx, goal, t); err != nil {
				errCh <- err
			}
		}(task)
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		if isApprovalRequired(err) {
			return err
		}
	}
	return nil
}

func (e *Engine) runTask(ctx context.Context, goal *storage.Goal, task *storage.Task) error {
	e.logger.Info(goal.ID, fmt.Sprintf("Executing task: %s", task.Description), observability.Fields{"task_id": task.ID})

	task.Status = "running"
	task.UpdatedAt = time.Now()
	task.Attempts++
	if err := e.store.UpdateTask(task); err != nil {
		return err
	}

	if task.ToolName != "" {
		return e.executeToolTask(ctx, goal, task)
	}
	return e.executeLLMTask(ctx, goal, task)
}

func (e *Engine) executeNextTask(ctx context.Context, goal *storage.Goal) error {
	tasks, err := e.store.GetTasksByGoal(goal.ID)
	if err != nil {
		return err
	}

	for i := range tasks {
		task := &tasks[i]
		if task.Status == "pending" && e.dependenciesMet(task) {
			return e.runTask(ctx, goal, task)
		}
	}
	return nil
}

func (e *Engine) handleApprovalRequired(goal *storage.Goal, task *storage.Task, toolName string, args map[string]interface{}, reason string) error {
	reason = strings.TrimPrefix(reason, "approval_required: ")
	_, err := e.permission.Approvals().CreatePending(goal.ID, task.ID, toolName, args, reason)
	if err != nil {
		return err
	}

	task.Status = "pending"
	task.Error = "awaiting approval: " + reason
	task.UpdatedAt = time.Now()
	_ = e.store.UpdateTask(task)

	if err := e.transitionGoal(goal, StateWaitingForApproval); err != nil {
		return err
	}
	e.logger.Warn(goal.ID, "Waiting for human approval", observability.Fields{"task_id": task.ID, "tool": toolName})
	return fmt.Errorf("approval_required: %s", reason)
}

func (e *Engine) executeToolTask(ctx context.Context, goal *storage.Goal, task *storage.Task) error {
	step := storageStepFromTask(task)
	step.MarkRunning()
	if err := e.store.CreateStep(stepToStorage(step)); err != nil {
		return err
	}

	output, err := e.runner.Run(ctx, task.ToolName, task.ToolArgs, RunConfig{
		Timeout: e.engineCfg.StepTimeout,
		GoalID:  goal.ID,
		TaskID:  task.ID,
		StepID:  step.ID,
	})
	if err != nil {
		if isApprovalRequired(err) {
			return e.handleApprovalRequired(goal, task, task.ToolName, task.ToolArgs, err.Error())
		}
		return e.handleStepFailure(task, step, output, err)
	}

	verdict, _ := e.evaluator.Evaluate(ctx, step.View())
	stepOutput := output
	if stepOutput == nil {
		stepOutput = &StepOutput{}
	}
	step.MarkCompleted(stepOutput, ValidationResult(verdict.Result), verdict.Detail)
	_ = e.store.UpdateStep(stepToStorage(step))

	e.memory.RecordStepOutcome(goal.ID, step.ID, task.Description, stepOutput.Output, stepOutput.Success)

	if verdict.Result == evaluator.RetryRequired && task.Attempts < task.MaxAttempts {
		task.Status = "pending"
		task.Error = verdict.Detail
		task.UpdatedAt = time.Now()
		return e.store.UpdateTask(task)
	}
	if verdict.Result == evaluator.Fatal || !stepOutput.Success {
		return e.failTask(task, fmt.Errorf("%s", verdict.Detail))
	}

	task.Status = "completed"
	task.Result = stepOutput.Output
	task.UpdatedAt = time.Now()
	return e.store.UpdateTask(task)
}

func (e *Engine) executeLLMTask(ctx context.Context, goal *storage.Goal, task *storage.Task) error {
	prompt := e.buildTaskExecutionPrompt(goal, task)

	resp, err := e.llm.Complete(ctx, prompt, &llm.Options{
		Temperature:  0.3,
		MaxTokens:    2048,
		SystemPrompt: e.getSystemPrompt(),
	})
	if err != nil {
		return e.failTask(task, fmt.Errorf("LLM execution failed: %w", err))
	}

	if resp.Usage != nil {
		e.logger.RecordTokenUsage(observability.TokenUsage{
			GoalID:           goal.ID,
			PromptTokens:     resp.Usage.PromptTokens,
			CompletionTokens: resp.Usage.CompletionTokens,
			TotalTokens:      resp.Usage.TotalTokens,
		})
	}

	toolCall, err := parseToolCall(resp.Content)
	if err != nil {
		if task.Attempts < task.MaxAttempts {
			task.Status = "pending"
			task.Error = err.Error()
			task.UpdatedAt = time.Now()
			return e.store.UpdateTask(task)
		}
		return e.failTask(task, err)
	}

	step := NewExecutionStep(
		fmt.Sprintf("step_%d", time.Now().UnixNano()),
		goal.ID,
		task.Description,
		StepInput{ToolName: toolCall.ToolName, ToolArgs: toolCall.Args},
	)
	step.TaskID = task.ID
	step.MarkRunning()
	_ = e.store.CreateStep(stepToStorage(step))

	output, err := e.runner.Run(ctx, toolCall.ToolName, toolCall.Args, RunConfig{
		Timeout: e.engineCfg.StepTimeout,
		GoalID:  goal.ID,
		TaskID:  task.ID,
		StepID:  step.ID,
	})
	if err != nil {
		if isApprovalRequired(err) {
			return e.handleApprovalRequired(goal, task, toolCall.ToolName, toolCall.Args, err.Error())
		}
		return e.handleStepFailure(task, step, output, err)
	}

	verdict, _ := e.evaluator.Evaluate(ctx, step.View())
	if output == nil {
		output = &StepOutput{}
	}
	step.MarkCompleted(output, ValidationResult(verdict.Result), verdict.Detail)
	_ = e.store.UpdateStep(stepToStorage(step))
	e.memory.RecordStepOutcome(goal.ID, step.ID, task.Description, output.Output, output.Success)

	if verdict.Result == evaluator.RetryRequired && task.Attempts < task.MaxAttempts {
		task.Status = "pending"
		task.Error = verdict.Detail
		task.UpdatedAt = time.Now()
		return e.store.UpdateTask(task)
	}
	if verdict.Result == evaluator.Fatal || !output.Success {
		return e.failTask(task, fmt.Errorf("%s", verdict.Detail))
	}

	task.Result = output.Output
	task.Status = "completed"
	task.UpdatedAt = time.Now()
	return e.store.UpdateTask(task)
}

func (e *Engine) handleStepFailure(task *storage.Task, step *ExecutionStep, output *StepOutput, err error) error {
	if output == nil {
		output = &StepOutput{Success: false, Error: err.Error()}
	}
	step.MarkFailed(output, ValidationFatal, err.Error())
	_ = e.store.UpdateStep(stepToStorage(step))

	if task.Attempts < task.MaxAttempts {
		task.Status = "pending"
		task.Error = err.Error()
		task.UpdatedAt = time.Now()
		return e.store.UpdateTask(task)
	}
	return e.failTask(task, err)
}

func (e *Engine) dependenciesMet(task *storage.Task) bool {
	for _, depID := range task.Dependencies {
		dep, err := e.store.GetTask(depID)
		if err != nil || dep.Status != "completed" {
			return false
		}
	}
	return true
}

func (e *Engine) summarizeTasks(tasks []storage.Task) (allDone, hasFatal bool) {
	allDone = true
	for _, task := range tasks {
		switch task.Status {
		case "completed":
		case "failed":
			if task.Attempts >= task.MaxAttempts {
				hasFatal = true
			} else {
				allDone = false
			}
		default:
			allDone = false
		}
	}
	return
}

func (e *Engine) detectStall(tasks []storage.Task) bool {
	completed := 0
	for _, t := range tasks {
		if t.Status == "completed" {
			completed++
		}
	}
	return e.loopCount > 5 && completed == 0
}

func (e *Engine) failGoal(goal *storage.Goal, err error) error {
	now := time.Now()
	goal.State = StateFailed
	goal.UpdatedAt = now
	goal.CompletedAt = &now
	goal.Error = err.Error()
	_ = e.store.UpdateGoal(goal)
	e.logger.RecordTimeline(observability.TimelineEvent{
		Timestamp: now,
		GoalID:    goal.ID,
		Event:     "goal_failed",
		Detail:    err.Error(),
	})
	return err
}

func (e *Engine) failTask(task *storage.Task, err error) error {
	task.Status = "failed"
	task.Error = err.Error()
	task.UpdatedAt = time.Now()
	return e.store.UpdateTask(task)
}

func isApprovalRequired(err error) bool {
	return err != nil && len(err.Error()) > 17 && err.Error()[:17] == "approval_required"
}

// ResumeFromCheckpoint restores goal execution from a saved checkpoint.
func (e *Engine) ResumeFromCheckpoint(ctx context.Context, checkpointID string) error {
	cp, err := e.store.GetCheckpoint(checkpointID)
	if err != nil {
		return err
	}
	payload, err := decodeCheckpointPayload(cp.Payload)
	if err != nil {
		return err
	}
	e.memory.LoadFromCheckpoint(cp.GoalID, payload.Memory)

	goal, err := e.store.GetGoal(cp.GoalID)
	if err != nil {
		return err
	}
	if err := e.transitionGoal(goal, payload.GoalState); err != nil {
		return err
	}
	return e.ProcessGoal(ctx, cp.GoalID)
}

// SaveCheckpoint persists current goal state for resume/replay.
func (e *Engine) SaveCheckpoint(goalID string, state GoalState, stepIndex int) (*Checkpoint, error) {
	payload := CheckpointPayload{
		GoalState: state,
		Memory:    e.memory.Export(goalID),
	}
	cp, err := NewCheckpoint(
		fmt.Sprintf("cp_%d", time.Now().UnixNano()),
		goalID,
		state,
		stepIndex,
		payload,
	)
	if err != nil {
		return nil, err
	}
	if err := e.store.CreateCheckpoint(checkpointFromEngine(cp)); err != nil {
		return nil, err
	}
	return cp, nil
}

func decodeCheckpointPayload(raw string) (*CheckpointPayload, error) {
	cp := &Checkpoint{Payload: raw}
	return cp.DecodePayload()
}
