package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// OllamaProvider implements Provider for Ollama
type OllamaProvider struct {
	baseURL string
	model   string
	client  *http.Client
}

// NewOllamaProvider creates a new Ollama provider
func NewOllamaProvider(baseURL, model string) *OllamaProvider {
	return &OllamaProvider{
		baseURL: baseURL,
		model:   model,
		client:  &http.Client{},
	}
}

// ollamaRequest represents an Ollama API request
type ollamaRequest struct {
	Model   string                 `json:"model"`
	Prompt  string                 `json:"prompt"`
	Stream  bool                   `json:"stream"`
	Options map[string]interface{} `json:"options,omitempty"`
}

// ollamaResponse represents an Ollama API response
type ollamaResponse struct {
	Model    string `json:"model"`
	Response string `json:"response"`
	Done     bool   `json:"done"`
	Context  []int  `json:"context,omitempty"`
}

// Complete implements Provider.Complete
func (p *OllamaProvider) Complete(ctx context.Context, prompt string, opts *Options) (*Response, error) {
	if opts == nil {
		opts = &Options{
			Temperature: 0.3,
			MaxTokens:   4096,
		}
	}

	// Build full prompt with system message if provided
	fullPrompt := prompt
	if opts.SystemPrompt != "" {
		fullPrompt = opts.SystemPrompt + "\n\n" + prompt
	}

	// Prepare request
	reqBody := ollamaRequest{
		Model:  p.model,
		Prompt: fullPrompt,
		Stream: false,
		Options: map[string]interface{}{
			"temperature": opts.Temperature,
			"num_predict": opts.MaxTokens,
		},
	}

	if len(opts.Stop) > 0 {
		reqBody.Options["stop"] = opts.Stop
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	url := fmt.Sprintf("%s/api/generate", p.baseURL)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// Send request
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ollama API error (status %d): %s", resp.StatusCode, string(body))
	}

	// Parse response
	var ollamaResp ollamaResponse
	if err := json.NewDecoder(resp.Body).Decode(&ollamaResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &Response{
		Content:      ollamaResp.Response,
		FinishReason: "complete",
	}, nil
}

// Name implements Provider.Name
func (p *OllamaProvider) Name() string {
	return "ollama"
}

// Model implements Provider.Model
func (p *OllamaProvider) Model() string {
	return p.model
}
