# TODO: Firefox Addon Integration for OctaAI Agent

## Overview

Integrate Firefox addon support into OctaAI agent to enable browser automation and web interaction capabilities.

## Phase 1: WebSocket Server for Addon Connection

### 1.1 Create WebSocket Server Package ✅

- [x] Create `pkg/browser/server.go`
  - [x] WebSocket server that listens on `ws://localhost:8765`
  - [x] Handle addon connection handshake
  - [x] Support multiple concurrent connections (multi-browser support)
  - [x] Implement ping/pong for connection health check
  - [x] Connection authentication with token

- [x] Create `pkg/browser/client.go`
  - [x] Track connected browsers
  - [x] Send commands to specific browser instance
  - [x] Receive responses and match to command IDs
  - [x] Handle connection errors and reconnection

- [x] Create `pkg/browser/types.go`
  ```go
  type BrowserCommand struct {
      ID      string                 `json:"id"`
      Type    string                 `json:"type"` // navigate, click, fill, etc.
      Params  map[string]interface{} `json:"params"`
      Timeout int                    `json:"timeout"` // milliseconds
  }

  type BrowserResponse struct {
      ID        string                 `json:"id"`
      Status    string                 `json:"status"` // success, error
      Result    interface{}            `json:"result"`
      Error     string                 `json:"error,omitempty"`
      PageState map[string]interface{} `json:"page_state"`
  }
  ```

### 1.2 Update Agent Daemon ✅

- [x] Modify `cmd/octa-agentd/main.go`
  - [x] Start WebSocket server on daemon startup
  - [ ] Add flag `--browser-port` (default: 8765) [TODO: Add CLI flag support]
  - [x] Log when browser connects/disconnects
  - [x] Graceful shutdown of WebSocket connections

- [x] Add configuration in `pkg/config/config.go`
  ```go
  type BrowserConfig struct {
      Enabled        bool     `yaml:"enabled"`
      Port           int      `yaml:"port"`
      Token          string   `yaml:"token"`
      AutoScreenshot bool     `yaml:"auto_screenshot"`
      BrowserDomains []string `yaml:"browser_domains"`
  }
  ```

## Phase 2: Browser Tool Im ✅

- [x] Create `pkg/tools/browser.go`
  ```go
  type BrowserTool struct {
      server *browser.Server
  }
  ```

- [x] Implement tool methods (via Execute interface):
  - [x] `Navigate(url string) error`
  - [x] `Click(selector string) error`
  - [x] `Fill(selector string, value string) error`
  - [x] `Submit(selector string) error`
  - [x] `Extract(selector string) (string, error)`
  - [x] `Screenshot(selector string) ([]byte, error)`
  - [x] `ExecuteScript(script string) (interface{}, error)`
  - [x] `WaitForElement(selector string, timeout int) error`
  - [x] `GetCookies(domain string) ([]Cookie, error)`

- [x] Register browser tool in daemon startup

- [ ] Register browser tool in `pkg/tools/registry.go`

### 2.2 Tool Schema Definition ✅

- [x] Update tool schema to include browser actions:
  ```json
  {
    "name": "browser",
    "description": "Interact with web pages through Firefox browser",
    "actions": [
      {
        "name": "navigate",
        "description": "Navigate to a URL",
        "parameters": {
          "url": "string (required) - The URL to visit"
        }
      },
      {
        "name": "click",
        "description": "Click an element on the page",
        "parameters": {
          "selector": "string (required) - CSS selector",
          "text": "string (optional) - Click element with this text"
        }
      },
      {
        "name": "fill",
        "description": "Fill a form field",
        "parameters": {
          "selector": "string (required) - CSS selector",
          "value": "string (required) - Value to fill"
        }
      },
      {
        "name": "extract",
        "description": "Extract text or data from page",
        "parameters": {
          "selector": "string (required) - CSS selector",
          "attribute": "string (optional) - Extract attribute instead of text"
        }
      },
      {
        "name": "screenshot",
        "description": "Take screenshot of page or element",
        "parameters": {
          "selector": "string (optional) - CSS selector for specific element",
          "output_path": "string (required) - Where to save screenshot"
        }
      }
    ]
  }
  ```

## Phase 3: Firefox Addon Development

### 3.1 Create Addon Project

- [ ] Create new directory `octaai-firefox-addon/` (sibling to OctaAI)
- [ ] Initialize manifest.json (Manifest V3)
- [ ] Set up development environment
  - [ ] webpack/rollup for bundling
  - [ ] ESLint configuration
  - [ ] Testing framework (Jest)

### 3.2 Background Script (WebSocket Client)

- [ ] Create `background.js`
  - [ ] Connect to `ws://localhost:8765` on startup
  - [ ] Authenticate with token from storage
  - [ ] Handle incoming commands
  - [ ] Route commands to content scripts
  - [ ] Send responses back to agent
  - [ ] Reconnection logic with exponential backoff

### 3.3 Content Scripts (Page Interaction)

- [ ] Create `content/content.js`
  - [ ] Listen for commands from background script
  - [ ] Implement DOM interaction functions
  - [ ] Handle async operations (wait for elements)
  - [ ] Error handling and timeouts
  - [ ] Send results back to background script

- [ ] Create `content/page-api.js`
  - [ ] Helper functions for common operations
  - [ ] Element finding strategies (CSS, XPath, text)
  - [ ] Form filling utilities
  - [ ] Data extraction utilities

### 3.4 Popup UI

- [ ] Create `popup/popup.html`
  - [ ] Connection status indicator
  - [ ] List of recent commands executed
  - [ ] Manual control buttons (connect/disconnect)
  - [ ] Settings page link

- [ ] Create `popup/popup.js`
  - [ ] Display connection status
  - [ ] Show activity log
  - [ ] Handle user actions

### 3.5 Options/Settings Page

- [ ] Create `options/options.html`
  - [ ] Configure agent daemon address/port
  - [ ] Set authentication token
  - [ ] Domain whitelist
  - [ ] Enable/disable auto-connect

## Phase 4: Integration & Testing

### 4.1 End-to-End Testing

- [ ] Test basic navigation
  ```bash
  octa-agent goal "Navigate to https://example.com and extract the main heading"
  ```

- [ ] Test form filling
  ```bash
  octa-agent goal "Go to https://httpbin.org/forms/post, fill out the form with test data, and submit it"
  ```

- [ ] Test multi-step workflow
  ```bash
  octa-agent goal "Go to GitHub trending page, extract top 5 repositories, and save to repos.json"
  ```

- [ ] Test error handling
  - Element not found
  - Page timeout
  - Connection lost during command

### 4.2 Security Testing

- [ ] Test authentication token validation
- [ ] Test command injection prevention
- [ ] Test XSS protection in content scripts
- [ ] Test CORS handling
- [ ] Test rate limiting

### 4.3 Performance Testing

- [ ] Measure command latency (agent -> addon -> response)
- [ ] Test with multiple browser instances
- [ ] Test long-running sessions (memory leaks)
- [ ] Test concurrent commands

## Phase 5: Documentation

### 5.1 User Documentation

- [ ] Update `README.md` with browser automation features
- [ ] Create `docs/BROWSER_AUTOMATION.md`
  - [ ] Installation guide for Firefox addon
  - [ ] Configuration guide
  - [ ] Example use cases
  - [ ] Troubleshooting

- [ ] Create `examples/browser/` directory
  - [ ] `web_scraping.md`
  - [ ] `form_automation.md`
  - [ ] `login_automation.md`
  - [ ] `data_extraction.md`

### 5.2 Developer Documentation

- [ ] Document WebSocket protocol
- [ ] Document addon architecture
- [ ] Create contribution guide for addon
- [ ] Add browser tool to API documentation

## Phase 6: Advanced Features (Future)

- [ ] Chrome extension support (cross-browser)
- [ ] Headless browser fallback (Puppeteer/Playwright)
- [ ] Visual element recorder (record user actions)
- [ ] AI-powered element detection
- [ ] Multi-tab support
- [ ] Browser profile management
- [ ] Proxy configuration
- [ ] Cookie/session management UI

## Dependencies to Add

```go
// go.mod
require (
    github.com/gorilla/websocket v1.5.1  // WebSocket server
    github.com/google/uuid v1.5.0        // Command ID generation
)
```

## Config File Updates

```yaml
# ~/.config/octaai/config.yaml
projects_root: "/home/user/Projects"

llm:
  provider: "ollama"
  model: "qwen2.5:7b"
  base_url: "http://localhost:11434"

browser:
  enabled: true
  port: 8765
  token: "your-secret-token-here"  # Generate with: uuidgen
  auto_screenshot: true  # Screenshot on error for debugging

safety:
  allow_paths:
    - "/home/user/Projects"
  deny_commands:
    - "rm -rf /"
  browser_domains:  # Optional whitelist
    - "github.com"
    - "*.google.com"
```

## Priority Order

1. **High Priority**: Phase 1 & 2 (WebSocket + Browser Tool)
2. **High Priority**: Phase 3.1-3.3 (Basic addon functionality)
3. **Medium Priority**: Phase 3.4-3.5 (UI/Settings)
4. **Medium Priority**: Phase 4 (Testing)
5. **Low Priority**: Phase 5 (Documentation)
6. **Low Priority**: Phase 6 (Advanced features)

## Estimated Timeline

- Phase 1-2: 2-3 days
- Phase 3: 3-4 days
- Phase 4: 2-3 days
- Phase 5: 1-2 days
- **Total**: ~10-14 days for MVP

## Success Criteria

✅ Agent can navigate to URLs via browser
✅ Agent can extract data from web pages
✅ Agent can fill and submit forms
✅ Agent can handle login flows
✅ Addon connects automatically to daemon
✅ Error handling works correctly
✅ Basic UI shows connection status
✅ Documentation is complete