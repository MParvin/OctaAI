package tools

import (
	"context"
	"fmt"
	"time"

	"github.com/mparvin/octaai/pkg/browser"
)

// BrowserTool provides browser automation capabilities
type BrowserTool struct {
	server *browser.Server
}

// NewBrowserTool creates a new browser automation tool
func NewBrowserTool(server *browser.Server) *BrowserTool {
	return &BrowserTool{
		server: server,
	}
}

// Name implements Tool.Name
func (t *BrowserTool) Name() string {
	return "browser"
}

// Schema implements Tool.Schema
func (t *BrowserTool) Schema() ToolSchema {
	return ToolSchema{
		Name:        "browser",
		Description: "Interact with web pages through Firefox browser addon. Requires browser addon to be connected.",
		Parameters: map[string]ParamSchema{
			"action": {
				Type:        "string",
				Description: "Action to perform",
				Required:    true,
				Enum: []string{
					"navigate", "click", "fill", "submit", "extract",
					"screenshot", "execute", "wait_for", "scroll",
					"get_cookies", "get_page_source",
				},
			},
			"url": {
				Type:        "string",
				Description: "URL to navigate to (for navigate action)",
				Required:    false,
			},
			"selector": {
				Type:        "string",
				Description: "CSS selector for element",
				Required:    false,
			},
			"value": {
				Type:        "string",
				Description: "Value to fill or text to match",
				Required:    false,
			},
			"script": {
				Type:        "string",
				Description: "JavaScript code to execute (for execute action)",
				Required:    false,
			},
			"timeout": {
				Type:        "number",
				Description: "Timeout in milliseconds (default: 30000)",
				Required:    false,
			},
			"output_path": {
				Type:        "string",
				Description: "Path to save screenshot (for screenshot action)",
				Required:    false,
			},
			"attribute": {
				Type:        "string",
				Description: "Attribute to extract from element (for extract action)",
				Required:    false,
			},
		},
	}
}

// Execute implements Tool.Execute
func (t *BrowserTool) Execute(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
	// Check if browser is connected
	if !t.server.HasConnectedBrowser() {
		return &ToolResult{
			Success: false,
			Error:   "No browser connected. Please install and connect the Firefox addon.",
		}, nil
	}

	action, ok := args["action"].(string)
	if !ok {
		return nil, fmt.Errorf("action is required")
	}

	// Build command
	cmd := &browser.BrowserCommand{
		Type:   action,
		Params: make(map[string]interface{}),
	}

	// Extract timeout
	timeout := 30 * time.Second
	if timeoutMs, ok := args["timeout"].(float64); ok {
		timeout = time.Duration(timeoutMs) * time.Millisecond
		cmd.Timeout = int(timeoutMs)
	} else {
		cmd.Timeout = 30000 // Default 30 seconds
	}

	// Build parameters based on action
	switch action {
	case browser.CommandNavigate:
		url, ok := args["url"].(string)
		if !ok {
			return nil, fmt.Errorf("url is required for navigate action")
		}
		cmd.Params["url"] = url

	case browser.CommandClick, browser.CommandFill, browser.CommandSubmit,
		browser.CommandExtract, browser.CommandWaitFor, browser.CommandScroll:
		if selector, ok := args["selector"].(string); ok {
			cmd.Params["selector"] = selector
		}
		if value, ok := args["value"].(string); ok {
			cmd.Params["value"] = value
		}
		if attribute, ok := args["attribute"].(string); ok {
			cmd.Params["attribute"] = attribute
		}

	case browser.CommandScreenshot:
		if selector, ok := args["selector"].(string); ok {
			cmd.Params["selector"] = selector
		}
		if outputPath, ok := args["output_path"].(string); ok {
			cmd.Params["output_path"] = outputPath
		}

	case browser.CommandExecute:
		script, ok := args["script"].(string)
		if !ok {
			return nil, fmt.Errorf("script is required for execute action")
		}
		cmd.Params["script"] = script

	case browser.CommandGetCookies:
		if domain, ok := args["domain"].(string); ok {
			cmd.Params["domain"] = domain
		}
	}

	// Send command to any available browser
	response, err := t.server.SendCommandToAny(cmd, timeout)
	if err != nil {
		return &ToolResult{
			Success: false,
			Error:   fmt.Sprintf("Browser command failed: %v", err),
		}, nil
	}

	// Process response
	if response.Status == browser.StatusError {
		return &ToolResult{
			Success: false,
			Error:   response.Error,
			Data:    response.PageState,
		}, nil
	}

	// Format result based on action
	output := ""
	if response.Result != nil {
		output = fmt.Sprintf("%v", response.Result)
	}

	return &ToolResult{
		Success: true,
		Output:  output,
		Data: map[string]interface{}{
			"result":     response.Result,
			"page_state": response.PageState,
		},
	}, nil
}
