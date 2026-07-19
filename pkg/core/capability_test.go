package core

import (
	"testing"
)

func TestCapabilityRegistry(t *testing.T) {
	registry := NewCapabilityRegistry()

	// Test registration
	cap1 := &Capability{
		ID:            "test.capability1",
		Name:          "Test Capability 1",
		Description:   "A test capability",
		RequiredTools: []string{"tool1", "tool2"},
		Cost:          CostTierLow,
	}

	err := registry.Register(cap1)
	if err != nil {
		t.Fatalf("Failed to register capability: %v", err)
	}

	// Test duplicate registration
	err = registry.Register(cap1)
	if err == nil {
		t.Error("Expected error when registering duplicate capability")
	}

	// Test retrieval
	retrieved, err := registry.Get("test.capability1")
	if err != nil {
		t.Fatalf("Failed to get capability: %v", err)
	}

	if retrieved.Name != cap1.Name {
		t.Errorf("Retrieved capability name = %s, want %s", retrieved.Name, cap1.Name)
	}

	// Test non-existent capability
	_, err = registry.Get("nonexistent")
	if err == nil {
		t.Error("Expected error when getting non-existent capability")
	}

	// Test list
	cap2 := &Capability{
		ID:   "test.capability2",
		Name: "Test Capability 2",
		Cost: CostTierFree,
	}
	if err := registry.Register(cap2); err != nil {
		t.Fatalf("Failed to register cap2: %v", err)
	}

	capabilities := registry.List()
	if len(capabilities) != 2 {
		t.Errorf("List() returned %d capabilities, want 2", len(capabilities))
	}

	// Test search
	results := registry.Search("capability1")
	if len(results) != 1 {
		t.Errorf("Search() returned %d results, want 1", len(results))
	}

	results = registry.Search("Test")
	if len(results) != 2 {
		t.Errorf("Search() for 'Test' returned %d results, want 2", len(results))
	}

	results = registry.Search("nonexistent")
	if len(results) != 0 {
		t.Errorf("Search() for 'nonexistent' returned %d results, want 0", len(results))
	}
}

func TestCapabilityConstraints(t *testing.T) {
	cap := &Capability{
		ID:   "test.constrained",
		Name: "Constrained Capability",
		Constraints: &CapabilityConstraints{
			MaxTokens:        1000,
			RequiresInternet: true,
			RequiresDocker:   false,
		},
	}

	if !cap.Constraints.RequiresInternet {
		t.Error("Expected capability to require internet")
	}

	if cap.Constraints.MaxTokens != 1000 {
		t.Errorf("MaxTokens = %d, want 1000", cap.Constraints.MaxTokens)
	}
}

func TestCostTier(t *testing.T) {
	tests := []struct {
		tier CostTier
		want string
	}{
		{CostTierFree, "free"},
		{CostTierLow, "low"},
		{CostTierMedium, "medium"},
		{CostTierHigh, "high"},
	}

	for _, tt := range tests {
		if string(tt.tier) != tt.want {
			t.Errorf("CostTier = %s, want %s", tt.tier, tt.want)
		}
	}
}
