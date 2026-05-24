package plugin

import (
	"context"

	"github.com/mparvin/octaai/pkg/browser"
	"github.com/mparvin/octaai/pkg/config"
	"github.com/mparvin/octaai/pkg/tools"
)

// CodingPlugin provides filesystem, command, and git tools.
type CodingPlugin struct{}

func (p *CodingPlugin) Name() string { return "coding" }

func (p *CodingPlugin) Capabilities() []Capability {
	return []Capability{CapabilityCoding}
}

func (p *CodingPlugin) Register(registry *tools.Registry, cfg *config.Config) error {
	registry.Register(tools.NewFilesystemTool(cfg))
	registry.Register(tools.NewCommandTool(cfg))
	registry.Register(tools.NewGitTool(cfg))
	return nil
}

func (p *CodingPlugin) Shutdown(_ context.Context) error { return nil }

// DevOpsPlugin provides HTTP and SSH tools.
type DevOpsPlugin struct{}

func (p *DevOpsPlugin) Name() string { return "devops" }

func (p *DevOpsPlugin) Capabilities() []Capability {
	return []Capability{CapabilityDevOps, CapabilitySSH}
}

func (p *DevOpsPlugin) Register(registry *tools.Registry, cfg *config.Config) error {
	registry.Register(tools.NewSSHTool(cfg))
	registry.Register(tools.NewHTTPTool())
	return nil
}

func (p *DevOpsPlugin) Shutdown(_ context.Context) error { return nil }

// BrowserPlugin provides browser automation when enabled.
type BrowserPlugin struct {
	Server *browser.Server
}

func (p *BrowserPlugin) Name() string { return "browser" }

func (p *BrowserPlugin) Capabilities() []Capability {
	return []Capability{CapabilityBrowser}
}

func (p *BrowserPlugin) Register(registry *tools.Registry, _ *config.Config) error {
	if p.Server != nil {
		registry.Register(tools.NewBrowserTool(p.Server))
	}
	return nil
}

func (p *BrowserPlugin) Shutdown(_ context.Context) error { return nil }

// DefaultPlugins returns the standard plugin set.
func DefaultPlugins(browserServer *browser.Server) []Plugin {
	plugins := []Plugin{&CodingPlugin{}, &DevOpsPlugin{}}
	if browserServer != nil {
		plugins = append(plugins, &BrowserPlugin{Server: browserServer})
	}
	return plugins
}
