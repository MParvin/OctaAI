package memory

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/mparvin/octaai/pkg/storage"
)

// Entry is a single memory record for a goal.
type Entry struct {
	Key       string    `json:"key"`
	Value     string    `json:"value"`
	Source    string    `json:"source"`
	CreatedAt time.Time `json:"created_at"`
}

// Manager stores task memory and execution history summaries.
type Manager struct {
	store storage.Storage
	mu    sync.RWMutex
	cache map[string][]Entry
}

// NewManager creates a memory manager.
func NewManager(store storage.Storage) *Manager {
	return &Manager{
		store: store,
		cache: make(map[string][]Entry),
	}
}

// Remember stores a key-value fact for a goal.
func (m *Manager) Remember(goalID, key, value, source string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	entry := Entry{
		Key:       key,
		Value:     value,
		Source:    source,
		CreatedAt: time.Now(),
	}
	m.cache[goalID] = append(m.cache[goalID], entry)

	data, _ := json.Marshal(entry)
	if m.store != nil {
		_ = m.store.CreateLog(&storage.ExecutionLog{
			ID:        fmt.Sprintf("mem_%d", time.Now().UnixNano()),
			GoalID:    goalID,
			Level:     "memory",
			Message:   key,
			Data:      string(data),
			CreatedAt: time.Now(),
		})
	}
}

// Recall returns all memory entries for a goal.
func (m *Manager) Recall(goalID string) []Entry {
	m.mu.RLock()
	defer m.mu.RUnlock()
	entries := make([]Entry, len(m.cache[goalID]))
	copy(entries, m.cache[goalID])
	return entries
}

// SummarizeContext builds a compact context string for LLM prompts.
func (m *Manager) SummarizeContext(goalID string, maxEntries int) string {
	entries := m.Recall(goalID)
	if len(entries) == 0 {
		return ""
	}
	if len(entries) > maxEntries {
		entries = entries[len(entries)-maxEntries:]
	}
	var b strings.Builder
	for _, e := range entries {
		b.WriteString(fmt.Sprintf("- %s: %s\n", e.Key, e.Value))
	}
	return b.String()
}

// RecordStepOutcome stores execution history from a completed step.
func (m *Manager) RecordStepOutcome(goalID, stepID, description, output string, success bool) {
	status := "success"
	if !success {
		status = "failure"
	}
	m.Remember(goalID, fmt.Sprintf("step:%s", stepID), fmt.Sprintf("[%s] %s", status, description), "execution")
	if output != "" && len(output) < 500 {
		m.Remember(goalID, fmt.Sprintf("step_output:%s", stepID), output, "step_output")
	}
}

// LoadFromCheckpoint restores in-memory cache from checkpoint payload.
func (m *Manager) LoadFromCheckpoint(goalID string, data map[string]string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for k, v := range data {
		m.cache[goalID] = append(m.cache[goalID], Entry{
			Key:       k,
			Value:     v,
			Source:    "checkpoint",
			CreatedAt: time.Now(),
		})
	}
}

// Export returns memory as a map for checkpoint serialization.
func (m *Manager) Export(goalID string) map[string]string {
	entries := m.Recall(goalID)
	out := make(map[string]string, len(entries))
	for _, e := range entries {
		out[e.Key] = e.Value
	}
	return out
}
