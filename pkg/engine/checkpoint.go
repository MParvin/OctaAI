package engine

import (
	"encoding/json"
	"time"
)

// Checkpoint captures a resumable snapshot of goal execution.
type Checkpoint struct {
	ID        string    `json:"id"`
	GoalID    string    `json:"goal_id"`
	State     GoalState `json:"state"`
	StepIndex int       `json:"step_index"`
	Payload   string    `json:"payload"`
	CreatedAt time.Time `json:"created_at"`
}

// CheckpointPayload holds serialized execution context.
type CheckpointPayload struct {
	GoalState   GoalState        `json:"goal_state"`
	StepIDs     []string         `json:"step_ids"`
	Memory      map[string]string `json:"memory,omitempty"`
	PlanVersion int              `json:"plan_version"`
}

// NewCheckpoint creates a checkpoint from current engine state.
func NewCheckpoint(id, goalID string, state GoalState, stepIndex int, payload CheckpointPayload) (*Checkpoint, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return &Checkpoint{
		ID:        id,
		GoalID:    goalID,
		State:     state,
		StepIndex: stepIndex,
		Payload:   string(data),
		CreatedAt: time.Now(),
	}, nil
}

// DecodePayload parses the checkpoint payload.
func (c *Checkpoint) DecodePayload() (*CheckpointPayload, error) {
	var payload CheckpointPayload
	if err := json.Unmarshal([]byte(c.Payload), &payload); err != nil {
		return nil, err
	}
	return &payload, nil
}
