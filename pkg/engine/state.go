package engine

import (
	"github.com/mparvin/octaai/pkg/execution"
	"github.com/mparvin/octaai/pkg/storage"
)

// GoalState mirrors storage goal states used by the execution engine.
type GoalState = storage.State

const (
	StateIdle               GoalState = storage.StateIdle
	StatePlanning           GoalState = storage.StatePlanning
	StateExecuting          GoalState = storage.StateExecuting
	StateEvaluating         GoalState = storage.StateEvaluating
	StateRetrying           GoalState = storage.StateRetrying
	StateWaitingForApproval GoalState = storage.StateWaitingForApproval
	StateBlocked            GoalState = storage.StateBlocked
	StateCompleted          GoalState = storage.StateCompleted
	StateFailed             GoalState = storage.StateFailed
)

// StepStatus and ValidationResult alias shared execution types.
type StepStatus = execution.StepStatus
type ValidationResult = execution.ValidationResult

const (
	StepPending    = execution.StepPending
	StepRunning    = execution.StepRunning
	StepCompleted  = execution.StepCompleted
	StepFailed     = execution.StepFailed
	StepSkipped    = execution.StepSkipped
	StepRolledBack = execution.StepRolledBack

	ValidationSuccess       = execution.ValidationSuccess
	ValidationPartial       = execution.ValidationPartial
	ValidationRetryRequired = execution.ValidationRetryRequired
	ValidationFatal         = execution.ValidationFatal
)

// Transition defines a valid state machine edge.
type Transition struct {
	From GoalState
	To   GoalState
}

// allowedTransitions encodes the goal-level state machine.
var allowedTransitions = []Transition{
	{StateIdle, StatePlanning},
	{StatePlanning, StateExecuting},
	{StatePlanning, StateFailed},
	{StateExecuting, StateEvaluating},
	{StateExecuting, StateWaitingForApproval},
	{StateExecuting, StateFailed},
	{StateEvaluating, StateCompleted},
	{StateEvaluating, StateRetrying},
	{StateEvaluating, StateFailed},
	{StateRetrying, StateExecuting},
	{StateRetrying, StateFailed},
	{StateWaitingForApproval, StateExecuting},
	{StateWaitingForApproval, StateFailed},
	{StateBlocked, StateExecuting},
	{StateBlocked, StateFailed},
}

// CanTransition reports whether moving from one goal state to another is allowed.
func CanTransition(from, to GoalState) bool {
	if from == to {
		return true
	}
	for _, t := range allowedTransitions {
		if t.From == from && t.To == to {
			return true
		}
	}
	return false
}
