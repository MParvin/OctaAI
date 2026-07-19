package capability

import (
	"time"

	"github.com/mparvin/octaai/pkg/core"
)

// RegisterBuiltinCapabilities registers all built-in capabilities
func RegisterBuiltinCapabilities(registry *core.CapabilityRegistry) error {
	capabilities := []*core.Capability{
		// Coding capabilities
		{
			ID:             "coding.filesystem",
			Name:           "Filesystem Operations",
			Description:    "Create, read, write, and manage files and directories",
			RequiredTools:  []string{"filesystem.create_directory", "filesystem.write_file", "filesystem.read_file", "filesystem.list_files", "filesystem.append_file"},
			RequiredModels: []string{"any"},
			Cost:           core.CostTierFree,
			Constraints: &core.CapabilityConstraints{
				MaxDuration: 5 * time.Minute,
			},
		},
		{
			ID:               "coding.command",
			Name:             "Command Execution",
			Description:      "Execute shell commands and scripts",
			RequiredTools:    []string{"command.execute"},
			RequiredModels:   []string{"any"},
			Cost:             core.CostTierLow,
			RequiresApproval: true, // Commands can be dangerous
			Constraints: &core.CapabilityConstraints{
				MaxDuration: 10 * time.Minute,
			},
		},
		{
			ID:               "coding.git",
			Name:             "Git Operations",
			Description:      "Clone repositories, commit changes, and manage version control",
			RequiredTools:    []string{"git.clone", "git.init", "git.commit_all", "git.push"},
			RequiredModels:   []string{"any"},
			Cost:             core.CostTierFree,
			RequiresApproval: true, // Pushing to remote requires approval
			Constraints: &core.CapabilityConstraints{
				MaxDuration: 15 * time.Minute,
			},
		},
		{
			ID:             "coding.python",
			Name:           "Python Development",
			Description:    "Write, test, and debug Python code",
			RequiredTools:  []string{"filesystem.write_file", "command.execute"},
			RequiredModels: []string{"qwen2.5-coder:32b", "deepseek-coder-v2", "gpt-4"},
			Cost:           core.CostTierLow,
			Constraints: &core.CapabilityConstraints{
				MaxDuration: 20 * time.Minute,
				MaxTokens:   10000,
			},
		},
		{
			ID:             "coding.go",
			Name:           "Go Development",
			Description:    "Write, test, and debug Go code",
			RequiredTools:  []string{"filesystem.write_file", "command.execute"},
			RequiredModels: []string{"qwen2.5-coder:32b", "deepseek-coder-v2", "gpt-4"},
			Cost:           core.CostTierLow,
			Constraints: &core.CapabilityConstraints{
				MaxDuration: 20 * time.Minute,
				MaxTokens:   10000,
			},
		},

		// DevOps capabilities
		{
			ID:               "devops.ssh",
			Name:             "SSH Remote Execution",
			Description:      "Execute commands on remote servers via SSH",
			RequiredTools:    []string{"ssh.exec", "ssh.upload", "ssh.download"},
			RequiredModels:   []string{"any"},
			Cost:             core.CostTierLow,
			RequiresApproval: true, // Remote execution requires approval
			Constraints: &core.CapabilityConstraints{
				MaxDuration: 30 * time.Minute,
			},
		},
		{
			ID:             "devops.http",
			Name:           "HTTP Operations",
			Description:    "Make HTTP requests and interact with APIs",
			RequiredTools:  []string{"http.get", "http.post", "http.put", "http.delete"},
			RequiredModels: []string{"any"},
			Cost:           core.CostTierFree,
			Constraints: &core.CapabilityConstraints{
				MaxDuration:      5 * time.Minute,
				RequiresInternet: true,
			},
		},

		// Research capabilities
		{
			ID:             "research.web",
			Name:           "Web Research",
			Description:    "Search the web and extract information",
			RequiredTools:  []string{"http.get", "browser.navigate", "browser.extract"},
			RequiredModels: []string{"qwen2.5:32b", "gpt-4"},
			Cost:           core.CostTierMedium,
			Constraints: &core.CapabilityConstraints{
				MaxDuration:      15 * time.Minute,
				MaxTokens:        15000,
				RequiresInternet: true,
			},
		},

		// Browser automation
		{
			ID:             "browser.automation",
			Name:           "Browser Automation",
			Description:    "Automate browser interactions, fill forms, and extract data",
			RequiredTools:  []string{"browser.navigate", "browser.click", "browser.fill", "browser.extract", "browser.screenshot"},
			RequiredModels: []string{"any"},
			Cost:           core.CostTierMedium,
			Constraints: &core.CapabilityConstraints{
				MaxDuration:      30 * time.Minute,
				RequiresInternet: true,
			},
		},
	}

	for _, cap := range capabilities {
		if err := registry.Register(cap); err != nil {
			return err
		}
	}

	return nil
}
