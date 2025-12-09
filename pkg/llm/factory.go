package llm

import (
	"fmt"

	"github.com/mparvin/octaai/pkg/config"
)

// NewProvider creates a new LLM provider based on configuration
func NewProvider(cfg *config.LLMConfig) (Provider, error) {
	switch cfg.Provider {
	case "ollama":
		return NewOllamaProvider(cfg.BaseURL, cfg.Model), nil
	case "openai":
		if cfg.APIKey == "" {
			return nil, fmt.Errorf("OpenAI API key is required")
		}
		return NewOpenAIProvider(cfg.APIKey, cfg.Model), nil
	case "claude":
		if cfg.APIKey == "" {
			return nil, fmt.Errorf("Claude API key is required")
		}
		return NewClaudeProvider(cfg.APIKey, cfg.Model), nil
	default:
		return nil, fmt.Errorf("unsupported LLM provider: %s", cfg.Provider)
	}
}
