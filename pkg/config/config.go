package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

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
	Features     FeatureFlags        `yaml:"features"`
}

// FeatureFlags holds experimental v2 toggles.
// Most flags are parsed for forward compatibility but are not yet wired into
// the production engine (pkg/engine). Leaving them true has no runtime effect
// until the corresponding Phase 7 integration lands. See IMPLEMENTATION_PLAN.md.
type FeatureFlags struct {
	// UseHTNPlanner — UNIMPLEMENTED in engine (packages exist under pkg/planner).
	UseHTNPlanner bool `yaml:"use_htn_planner"`

	// UseDAGExecutor — UNIMPLEMENTED (pkg/workflow helpers are test-only today).
	UseDAGExecutor bool `yaml:"use_dag_executor"`

	// EnableAG2 — UNIMPLEMENTED (no AutoGen integration).
	EnableAG2 bool `yaml:"enable_ag2"`

	// UseVectorMemory — UNIMPLEMENTED (memory uses TF-IDF in pkg/memory/semantic.go).
	UseVectorMemory bool `yaml:"use_vector_memory"`

	// UseCapabilities — registry code exists (pkg/capability) but engine ignores this flag.
	UseCapabilities bool `yaml:"use_capabilities"`

	// EnableMCP — UNIMPLEMENTED.
	EnableMCP bool `yaml:"enable_mcp"`

	// EnableAdaptiveReplan — UNIMPLEMENTED (v1 replan uses engine.enable_replan).
	EnableAdaptiveReplan bool `yaml:"enable_adaptive_replan"`

	// EnableReflection — UNIMPLEMENTED.
	EnableReflection bool `yaml:"enable_reflection"`
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
	AllowHTTPHosts         []string `yaml:"allow_http_hosts"`
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
		Features: FeatureFlags{
			// All experimental; defaults off until engine wiring exists.
			UseHTNPlanner:        false,
			UseDAGExecutor:       false,
			EnableAG2:            false,
			UseVectorMemory:      false,
			UseCapabilities:      false,
			EnableMCP:            false,
			EnableAdaptiveReplan: false,
			EnableReflection:     false,
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

	cfg.ProjectsRoot = ExpandPath(cfg.ProjectsRoot)
	for i, p := range cfg.Safety.AllowPaths {
		cfg.Safety.AllowPaths[i] = ExpandPath(p)
	}
	cfg.SSH.KnownHostsFile = ExpandPath(cfg.SSH.KnownHostsFile)
	cfg.SSH.DefaultKeyPath = ExpandPath(cfg.SSH.DefaultKeyPath)
	cfg.Storage.Path = ExpandPath(cfg.Storage.Path)
	cfg.Isolation.Docker.WorkdirMount = ExpandPath(cfg.Isolation.Docker.WorkdirMount)

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

	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// ExpandPath expands environment variables and a leading ~ using the HOME variable.
func ExpandPath(path string) string {
	path = os.ExpandEnv(path)
	if path == "" {
		return path
	}

	home := os.Getenv("HOME")
	if home == "" {
		home, _ = os.UserHomeDir()
	}
	if home == "" {
		return path
	}

	if path == "~" {
		return home
	}
	if strings.HasPrefix(path, "~/") {
		return filepath.Join(home, path[2:])
	}
	return path
}

// ResolveProjectPath expands ~ and env vars, then resolves relative paths against projects_root.
func ResolveProjectPath(cfg *Config, path string) string {
	path = ExpandPath(path)
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(cfg.ProjectsRoot, path)
}

// ConfigPath returns the default configuration file path
func ConfigPath() string {
	homeDir, _ := os.UserHomeDir()
	return filepath.Join(homeDir, ".config", "octaai", "config.yaml")
}
