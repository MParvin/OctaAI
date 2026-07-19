package core

import (
	"context"
	"fmt"
	"sync"
)

// CapabilityRegistry manages the available capabilities in the system
type CapabilityRegistry struct {
	capabilities map[string]*Capability
	mu           sync.RWMutex
}

// NewCapabilityRegistry creates a new capability registry
func NewCapabilityRegistry() *CapabilityRegistry {
	return &CapabilityRegistry{
		capabilities: make(map[string]*Capability),
	}
}

// Register adds a capability to the registry
func (r *CapabilityRegistry) Register(capability *Capability) error {
	if capability.ID == "" {
		return fmt.Errorf("capability ID cannot be empty")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.capabilities[capability.ID]; exists {
		return fmt.Errorf("capability %s already registered", capability.ID)
	}

	r.capabilities[capability.ID] = capability
	return nil
}

// Get retrieves a capability by ID
func (r *CapabilityRegistry) Get(id string) (*Capability, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	capability, exists := r.capabilities[id]
	if !exists {
		return nil, fmt.Errorf("capability %s not found", id)
	}

	return capability, nil
}

// List returns all registered capabilities
func (r *CapabilityRegistry) List() []*Capability {
	r.mu.RLock()
	defer r.mu.RUnlock()

	capabilities := make([]*Capability, 0, len(r.capabilities))
	for _, cap := range r.capabilities {
		capabilities = append(capabilities, cap)
	}

	return capabilities
}

// Search finds capabilities matching a query
func (r *CapabilityRegistry) Search(query string) []*Capability {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Simple substring matching for now
	// TODO: Implement more sophisticated matching (fuzzy, semantic)
	var results []*Capability
	for _, cap := range r.capabilities {
		if containsIgnoreCase(cap.Name, query) ||
			containsIgnoreCase(cap.Description, query) ||
			containsIgnoreCase(cap.ID, query) {
			results = append(results, cap)
		}
	}

	return results
}

// CapabilityResolver resolves tasks to capabilities
type CapabilityResolver interface {
	Resolve(ctx context.Context, taskDescription string) (*Capability, error)
}

// Helper function for case-insensitive substring matching
func containsIgnoreCase(s, substr string) bool {
	s = toLower(s)
	substr = toLower(substr)
	return contains(s, substr)
}

func toLower(s string) string {
	// Simple ASCII lowercase conversion
	result := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 32
		}
		result[i] = c
	}
	return string(result)
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && indexOf(s, substr) >= 0
}

func indexOf(s, substr string) int {
	if len(substr) == 0 {
		return 0
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
