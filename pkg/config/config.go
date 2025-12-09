package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config represents the main configuration structure
type Config struct {
	ProjectsRoot string        `yaml:"projects_root"`
	LLM          LLMConfig     `yaml:"llm"`
	Safety       SafetyConfig  `yaml:"safety"`
	SSH          SSHConfig     `yaml:"ssh"`
	Storage      StorageConfig `yaml:"storage"`
}

// LLMConfig holds LLM provider configuration
type LLMConfig struct {
	Provider    string  `yaml:"provider"` // "openai", "claude", "ollama"
	Model       string  `yaml:"model"`    // e.g., "qwen2.5:32b", "gpt-4"
	BaseURL     string  `yaml:"base_url"` // For Ollama or custom endpoints
	APIKey      string  `yaml:"api_key"`  // For OpenAI/Claude
	Temperature float64 `yaml:"temperature"`
	MaxTokens   int     `yaml:"max_tokens"`
}

// SafetyConfig defines safety constraints
type SafetyConfig struct {
	AllowPaths             []string `yaml:"allow_paths"`
	DenyCommands           []string `yaml:"deny_commands"`
	RequireConfirmationFor []string `yaml:"require_confirmation_for"`
}

// SSHConfig holds SSH-related configuration
type SSHConfig struct {
	DefaultPort    int    `yaml:"default_port"`
	KnownHostsFile string `yaml:"known_hosts_file"`
	DefaultKeyPath string `yaml:"default_key_path"`
}

// StorageConfig defines where state is stored
type StorageConfig struct {
	Type string `yaml:"type"` // "sqlite", "postgres"
	Path string `yaml:"path"` // For sqlite
	DSN  string `yaml:"dsn"`  // For postgres
}

// DefaultConfig returns a configuration with sensible defaults
func DefaultConfig() *Config {
	homeDir, _ := os.UserHomeDir()

	return &Config{
		ProjectsRoot: filepath.Join(homeDir, "Projects"),
		LLM: LLMConfig{
			Provider:    "ollama",
			Model:       "qwen2.5:32b",
			BaseURL:     "http://localhost:11434",
			Temperature: 0.3,
			MaxTokens:   4096,
		},
		Safety: SafetyConfig{
			AllowPaths: []string{
				filepath.Join(homeDir, "Projects"),
			},
			DenyCommands: []string{
				"rm -rf /",
				":(){ :|:& };:",
				"mkfs",
				"dd if=/dev/zero",
			},
			RequireConfirmationFor: []string{
				"apt upgrade",
				"dnf upgrade",
				"yum upgrade",
			},
		},
		SSH: SSHConfig{
			DefaultPort:    22,
			KnownHostsFile: filepath.Join(homeDir, ".ssh", "known_hosts"),
			DefaultKeyPath: filepath.Join(homeDir, ".ssh", "id_rsa"),
		},
		Storage: StorageConfig{
			Type: "sqlite",
			Path: filepath.Join(homeDir, ".config", "octaai", "state.db"),
		},
	}
}

// LoadConfig loads configuration from a YAML file
func LoadConfig(path string) (*Config, error) {
	// Start with defaults
	cfg := DefaultConfig()

	// If file doesn't exist, return defaults
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return cfg, nil
	}

	// Read file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse YAML
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Expand environment variables in paths
	cfg.ProjectsRoot = os.ExpandEnv(cfg.ProjectsRoot)
	cfg.SSH.KnownHostsFile = os.ExpandEnv(cfg.SSH.KnownHostsFile)
	cfg.SSH.DefaultKeyPath = os.ExpandEnv(cfg.SSH.DefaultKeyPath)
	cfg.Storage.Path = os.ExpandEnv(cfg.Storage.Path)

	// Read API key from environment if not in config
	if cfg.LLM.APIKey == "" {
		if cfg.LLM.Provider == "openai" {
			cfg.LLM.APIKey = os.Getenv("OPENAI_API_KEY")
		} else if cfg.LLM.Provider == "claude" {
			cfg.LLM.APIKey = os.Getenv("ANTHROPIC_API_KEY")
		}
	}

	return cfg, nil
}

// SaveConfig saves the configuration to a YAML file
func SaveConfig(cfg *Config, path string) error {
	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Marshal to YAML
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write file
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// ConfigPath returns the default configuration file path
func ConfigPath() string {
	homeDir, _ := os.UserHomeDir()
	return filepath.Join(homeDir, ".config", "octaai", "config.yaml")
}
