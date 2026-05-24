package observability

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/mparvin/octaai/pkg/storage"
)

// Level is a structured log level.
type Level string

const (
	LevelDebug Level = "debug"
	LevelInfo  Level = "info"
	LevelWarn  Level = "warn"
	LevelError Level = "error"
)

// Fields holds structured log metadata.
type Fields map[string]interface{}

// Span represents a trace span for one operation.
type Span struct {
	ID        string     `json:"id"`
	GoalID    string     `json:"goal_id"`
	TaskID    string     `json:"task_id,omitempty"`
	StepID    string     `json:"step_id,omitempty"`
	Name      string     `json:"name"`
	StartedAt time.Time  `json:"started_at"`
	EndedAt   *time.Time `json:"ended_at,omitempty"`
	Duration  int64      `json:"duration_ms,omitempty"`
	Metadata  Fields     `json:"metadata,omitempty"`
}

// TokenUsage tracks LLM token consumption for a goal.
type TokenUsage struct {
	GoalID           string `json:"goal_id"`
	PromptTokens     int    `json:"prompt_tokens"`
	CompletionTokens int    `json:"completion_tokens"`
	TotalTokens      int    `json:"total_tokens"`
}

// TimelineEvent is a point on the execution timeline.
type TimelineEvent struct {
	Timestamp time.Time `json:"timestamp"`
	GoalID    string    `json:"goal_id"`
	Event     string    `json:"event"`
	Detail    string    `json:"detail,omitempty"`
	Fields    Fields    `json:"fields,omitempty"`
}

// Logger persists structured logs and timeline events.
type Logger struct {
	store storage.Storage
}

// NewLogger creates a structured logger backed by storage.
func NewLogger(store storage.Storage) *Logger {
	return &Logger{store: store}
}

// Log writes a structured log entry.
func (l *Logger) Log(goalID, taskID, level, message string, fields Fields) {
	data := ""
	if len(fields) > 0 {
		if b, err := json.Marshal(fields); err == nil {
			data = string(b)
		}
	}
	_ = l.store.CreateLog(&storage.ExecutionLog{
		ID:        fmt.Sprintf("log_%d", time.Now().UnixNano()),
		GoalID:    goalID,
		TaskID:    taskID,
		Level:     level,
		Message:   message,
		Data:      data,
		CreatedAt: time.Now(),
	})
}

// Info logs at info level.
func (l *Logger) Info(goalID, message string, fields Fields) {
	l.Log(goalID, "", string(LevelInfo), message, fields)
}

// Warn logs at warn level.
func (l *Logger) Warn(goalID, message string, fields Fields) {
	l.Log(goalID, "", string(LevelWarn), message, fields)
}

// Error logs at error level.
func (l *Logger) Error(goalID, message string, fields Fields) {
	l.Log(goalID, "", string(LevelError), message, fields)
}

// Debug logs at debug level.
func (l *Logger) Debug(goalID, message string, fields Fields) {
	l.Log(goalID, "", string(LevelDebug), message, fields)
}

// RecordTimeline stores a timeline event as a structured log.
func (l *Logger) RecordTimeline(event TimelineEvent) {
	fields := Fields{
		"event":     event.Event,
		"timeline":  true,
		"timestamp": event.Timestamp.Format(time.RFC3339Nano),
	}
	for k, v := range event.Fields {
		fields[k] = v
	}
	l.Log(event.GoalID, "", string(LevelInfo), event.Detail, fields)
}

// RecordTokenUsage logs token consumption for cost tracking.
func (l *Logger) RecordTokenUsage(usage TokenUsage) {
	l.Log(usage.GoalID, "", string(LevelInfo), "token usage", Fields{
		"prompt_tokens":     usage.PromptTokens,
		"completion_tokens": usage.CompletionTokens,
		"total_tokens":      usage.TotalTokens,
		"token_usage":       true,
	})
}

// StartSpan begins a trace span (in-memory; ended spans are logged).
func (l *Logger) StartSpan(goalID, name string, fields Fields) *Span {
	span := &Span{
		ID:        fmt.Sprintf("span_%d", time.Now().UnixNano()),
		GoalID:    goalID,
		Name:      name,
		StartedAt: time.Now(),
		Metadata:  fields,
	}
	l.Log(goalID, "", string(LevelDebug), fmt.Sprintf("span start: %s", name), Fields{
		"span_id": span.ID,
		"span":    true,
	})
	return span
}

// EndSpan completes a span and persists duration.
func (l *Logger) EndSpan(span *Span) {
	now := time.Now()
	span.EndedAt = &now
	span.Duration = now.Sub(span.StartedAt).Milliseconds()
	l.Log(span.GoalID, span.TaskID, string(LevelDebug), fmt.Sprintf("span end: %s", span.Name), Fields{
		"span_id":     span.ID,
		"duration_ms": span.Duration,
		"span":        true,
	})
}
