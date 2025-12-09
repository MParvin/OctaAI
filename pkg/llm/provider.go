package llm

import (
	"context"
)

// Provider defines the interface for LLM providers
type Provider interface {
	// Complete sends a prompt and returns the completion
	Complete(ctx context.Context, prompt string, opts *Options) (*Response, error)

	// Name returns the provider name
	Name() string

	// Model returns the model being used
	Model() string
}

// Options contains parameters for LLM completion
type Options struct {
	Temperature  float64
	MaxTokens    int
	SystemPrompt string
	Stop         []string
}

// Response represents the LLM response
type Response struct {
	Content      string
	FinishReason string
	Usage        *Usage
}

// Usage contains token usage information
type Usage struct {
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
}

// Message represents a chat message
type Message struct {
	Role    string // "system", "user", "assistant"
	Content string
}
