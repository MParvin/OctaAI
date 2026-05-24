package plugin

import (
	"context"

	"github.com/mparvin/octaai/pkg/config"
	"github.com/mparvin/octaai/pkg/tools"
)

// Capability describes what a plugin provides.
type Capability string

const (
	CapabilityCoding    Capability = "coding"
	CapabilityDevOps    Capability = "devops"
	CapabilityRedTeam   Capability = "redteam"
	CapabilityK8s       Capability = "kubernetes"
	CapabilityBrowser   Capability = "browser"
	CapabilitySSH       Capability = "ssh"
)

// Plugin registers tools and capabilities with the core engine.
type Plugin interface {
	Name() string
	Capabilities() []Capability
	Register(registry *tools.Registry, cfg *config.Config) error
	Shutdown(ctx context.Context) error
}

// Registry manages loaded plugins.
type Registry struct {
	plugins map[string]Plugin
}

// NewRegistry creates an empty plugin registry.
func NewRegistry() *Registry {
	return &Registry{plugins: make(map[string]Plugin)}
}

// Register adds a plugin.
func (r *Registry) Register(p Plugin) {
	r.plugins[p.Name()] = p
}

// LoadAll registers all plugin tools into the tool registry.
func (r *Registry) LoadAll(toolRegistry *tools.Registry, cfg *config.Config) error {
	for _, p := range r.plugins {
		if err := p.Register(toolRegistry, cfg); err != nil {
			return err
		}
	}
	return nil
}

// Shutdown stops all plugins.
func (r *Registry) Shutdown(ctx context.Context) error {
	for _, p := range r.plugins {
		_ = p.Shutdown(ctx)
	}
	return nil
}

// List returns registered plugins.
func (r *Registry) List() []Plugin {
	out := make([]Plugin, 0, len(r.plugins))
	for _, p := range r.plugins {
		out = append(out, p)
	}
	return out
}

// HasCapability reports whether any plugin provides a capability.
func (r *Registry) HasCapability(cap Capability) bool {
	for _, p := range r.plugins {
		for _, c := range p.Capabilities() {
			if c == cap {
				return true
			}
		}
	}
	return false
}
