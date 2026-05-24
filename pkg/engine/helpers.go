package engine

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/mparvin/octaai/pkg/storage"
)

func storageStepFromTask(task *storage.Task) *ExecutionStep {
	step := NewExecutionStep(
		fmt.Sprintf("step_%s_%d", task.ID, time.Now().UnixNano()),
		task.GoalID,
		task.Description,
		StepInput{ToolName: task.ToolName, ToolArgs: task.ToolArgs},
	)
	step.TaskID = task.ID
	return step
}

func stepToStorage(step *ExecutionStep) *storage.ExecutionStepRecord {
	var outputJSON, inputJSON string
	if b, err := json.Marshal(step.Input); err == nil {
		inputJSON = string(b)
	}
	if step.Output != nil {
		if b, err := json.Marshal(step.Output); err == nil {
			outputJSON = string(b)
		}
	}
	return &storage.ExecutionStepRecord{
		ID:               step.ID,
		GoalID:           step.GoalID,
		TaskID:           step.TaskID,
		Description:      step.Description,
		Status:           string(step.Status),
		InputJSON:        inputJSON,
		OutputJSON:       outputJSON,
		Validation:       string(step.Validation),
		ValidationDetail: step.ValidationDetail,
		RetryCount:       step.RetryCount,
		MaxRetries:       step.MaxRetries,
		StartedAt:        step.StartedAt,
		CompletedAt:      step.CompletedAt,
		CreatedAt:        step.CreatedAt,
		UpdatedAt:        step.UpdatedAt,
		CheckpointID:     step.CheckpointID,
	}
}

func checkpointFromEngine(cp *Checkpoint) *storage.CheckpointRecord {
	return &storage.CheckpointRecord{
		ID:        cp.ID,
		GoalID:    cp.GoalID,
		State:     string(cp.State),
		StepIndex: cp.StepIndex,
		Payload:   cp.Payload,
		CreatedAt: cp.CreatedAt,
	}
}

type toolCall struct {
	ToolName string
	Args     map[string]interface{}
}

func parseToolCall(content string) (*toolCall, error) {
	content = strings.ReplaceAll(content, "```json", "")
	content = strings.ReplaceAll(content, "```", "")
	content = strings.TrimSpace(content)

	start := strings.Index(content, "{")
	end := strings.LastIndex(content, "}")
	if start == -1 || end == -1 {
		return nil, fmt.Errorf("no JSON object found in response")
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(content[start:end+1]), &parsed); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	toolName := ""
	for _, key := range []string{"tool", "tool_name", "toolName", "name"} {
		if name, ok := parsed[key].(string); ok && name != "" {
			toolName = name
			break
		}
	}
	if toolName == "" {
		return nil, fmt.Errorf("tool name not found in response")
	}

	var args map[string]interface{}
	for _, key := range []string{"args", "arguments", "params", "parameters", "tool_args"} {
		if a, ok := parsed[key].(map[string]interface{}); ok {
			args = a
			break
		}
	}
	if args == nil {
		return nil, fmt.Errorf("args not found in response")
	}

	return &toolCall{ToolName: toolName, Args: args}, nil
}

func stepFromStorage(rec *storage.ExecutionStepRecord) *ExecutionStep {
	step := &ExecutionStep{
		ID:               rec.ID,
		GoalID:           rec.GoalID,
		TaskID:           rec.TaskID,
		Description:      rec.Description,
		Status:           StepStatus(rec.Status),
		Validation:       ValidationResult(rec.Validation),
		ValidationDetail: rec.ValidationDetail,
		RetryCount:       rec.RetryCount,
		MaxRetries:       rec.MaxRetries,
		StartedAt:        rec.StartedAt,
		CompletedAt:      rec.CompletedAt,
		CreatedAt:        rec.CreatedAt,
		UpdatedAt:        rec.UpdatedAt,
		CheckpointID:     rec.CheckpointID,
	}
	_ = json.Unmarshal([]byte(rec.InputJSON), &step.Input)
	if rec.OutputJSON != "" {
		var out StepOutput
		_ = json.Unmarshal([]byte(rec.OutputJSON), &out)
		step.Output = &out
	}
	return step
}
