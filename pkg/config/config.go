package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config represents the main configuration structure
type Config struct {
	ProjectsRoot string              `yaml:"projects_root"`
	LLM          LLMConfig           `yaml:"llm"`
	Safety       SafetyConfig        `yaml:"safety"`
	SSH          SSHConfig           `yaml:"ssh"`
	Storage      StorageConfig       `yaml:"storage"`
	Browser      BrowserConfig       `yaml:"browser"`
	Isolation    IsolationConfig     `yaml:"isolation"`
	Engine       EngineRuntimeConfig `yaml:"engine"`
}

// LLMConfig holds LLM provider configuration
type LLMConfig struct {
	Provider    string  `yaml:"provider"`
	Model       string  `yaml:"model"`
	BaseURL     string  `yaml:"base_url"`
	APIKey      string  `yaml:"api_key"`
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
	Type string `yaml:"type"`
	Path string `yaml:"path"`
	DSN  string `yaml:"dsn"`
}

// BrowserConfig holds browser automation configuration
type BrowserConfig struct {
	Enabled        bool     `yaml:"enabled"`
	Port           int      `yaml:"port"`
	Token          string   `yaml:"token"`
	AutoScreenshot bool     `yaml:"auto_screenshot"`
	BrowserDomains []string `yaml:"browser_domains"`
}

// IsolationConfig controls execution isolation for dangerous operations.
type IsolationConfig struct {
	Enabled          bool         `yaml:"enabled"`
	Docker           DockerConfig `yaml:"docker"`
	MaxParallel      int          `yaml:"max_parallel"`
	RequireDockerFor []string     `yaml:"require_docker_for"`
}

// DockerConfig holds Docker sandbox settings.
type DockerConfig struct {
	Enabled      bool     `yaml:"enabled"`
	Image        string   `yaml:"image"`
	Network      string   `yaml:"network"`
	MemoryLimit  string   `yaml:"memory_limit"`
	CPULimit     string   `yaml:"cpu_limit"`
	ReadOnlyRoot bool     `yaml:"read_only_root"`
	WorkdirMount string   `yaml:"workdir_mount"`
	ExtraArgs    []string `yaml:"extra_args"`
}

// EngineRuntimeConfig holds engine-level runtime settings.
type EngineRuntimeConfig struct {
	MaxLoops       int  `yaml:"max_loops"`
	MaxRetries     int  `yaml:"max_retries"`
	EnableReplan   bool `yaml:"enable_replan"`
	EnableParallel bool `yaml:"enable_parallel"`
}

// DefaultConfig returns a configuration with sensible defaults
func DefaultConfig() *Config {
	homeDir, _ := os.UserHomeDir()
	projectsRoot := filepath.Join(homeDir, "Projects")

	return &Config{
		ProjectsRoot: projectsRoot,
		LLM: LLMConfig{
			Provider:    "ollama",
			Model:       "qwen2.5:32b",
			BaseURL:     "http://localhost:11434",
			Temperature: 0.3,
			MaxTokens:   4096,
		},
		Safety: SafetyConfig{
			AllowPaths: []string{projectsRoot},
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
		Browser: BrowserConfig{
			Enabled:        false,
			Port:           8765,
			Token:          "",
			AutoScreenshot: true,
			BrowserDomains: []string{},
		},
		Isolation: IsolationConfig{
			Enabled: false,
			Docker: DockerConfig{
				Enabled:      false,
				Image:        "alpine:3.19",
				Network:      "none",
				MemoryLimit:  "512m",
				CPULimit:     "1.0",
				ReadOnlyRoot: true,
				WorkdirMount: projectsRoot,
			},
			MaxParallel:      3,
			RequireDockerFor: []string{"command"},
		},
		Engine: EngineRuntimeConfig{
			MaxLoops:       50,
			MaxRetries:     3,
			EnableReplan:   true,
			EnableParallel: true,
		},
	}
}

// LoadConfig loads configuration from a YAML file
func LoadConfig(path string) (*Config, error) {
	cfg := DefaultConfig()

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return cfg, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	cfg.ProjectsRoot = os.ExpandEnv(cfg.ProjectsRoot)
	cfg.SSH.KnownHostsFile = os.ExpandEnv(cfg.SSH.KnownHostsFile)
	cfg.SSH.DefaultKeyPath = os.ExpandEnv(cfg.SSH.DefaultKeyPath)
	cfg.Storage.Path = os.ExpandEnv(cfg.Storage.Path)
	cfg.Isolation.Docker.WorkdirMount = os.ExpandEnv(cfg.Isolation.Docker.WorkdirMount)

	if cfg.Isolation.Docker.WorkdirMount == "" {
		cfg.Isolation.Docker.WorkdirMount = cfg.ProjectsRoot
	}
	if cfg.Isolation.MaxParallel <= 0 {
		cfg.Isolation.MaxParallel = 3
	}

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
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

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
