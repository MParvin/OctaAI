package tools

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// HTTPTool provides HTTP operations
type HTTPTool struct {
	client *http.Client
}

// NewHTTPTool creates a new HTTP tool
func NewHTTPTool() *HTTPTool {
	return &HTTPTool{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Name implements Tool.Name
func (t *HTTPTool) Name() string {
	return "http"
}

// Schema implements Tool.Schema
func (t *HTTPTool) Schema() ToolSchema {
	return ToolSchema{
		Name:        "http",
		Description: "Make HTTP requests to web services",
		Parameters: map[string]ParamSchema{
			"method": {
				Type:        "string",
				Description: "HTTP method",
				Required:    true,
				Enum:        []string{"GET", "POST", "PUT", "DELETE"},
			},
			"url": {
				Type:        "string",
				Description: "Request URL",
				Required:    true,
			},
			"headers": {
				Type:        "object",
				Description: "HTTP headers as key-value pairs",
				Required:    false,
			},
			"body": {
				Type:        "string",
				Description: "Request body",
				Required:    false,
			},
		},
	}
}

// Execute implements Tool.Execute
func (t *HTTPTool) Execute(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
	method, ok := args["method"].(string)
	if !ok {
		return nil, fmt.Errorf("method is required")
	}

	url, ok := args["url"].(string)
	if !ok {
		return nil, fmt.Errorf("url is required")
	}

	var body io.Reader
	if bodyStr, ok := args["body"].(string); ok && bodyStr != "" {
		body = strings.NewReader(bodyStr)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return &ToolResult{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	// Add headers
	if headers, ok := args["headers"].(map[string]interface{}); ok {
		for key, value := range headers {
			if strValue, ok := value.(string); ok {
				req.Header.Set(key, strValue)
			}
		}
	}

	resp, err := t.client.Do(req)
	if err != nil {
		return &ToolResult{
			Success: false,
			Error:   err.Error(),
		}, nil
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return &ToolResult{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	success := resp.StatusCode >= 200 && resp.StatusCode < 300

	return &ToolResult{
		Success: success,
		Output:  string(respBody),
		Data: map[string]interface{}{
			"status_code": resp.StatusCode,
			"status":      resp.Status,
			"headers":     resp.Header,
		},
	}, nil
}
