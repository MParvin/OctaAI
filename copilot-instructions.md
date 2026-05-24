# GitHub Copilot Instructions for OctaAI Firefox Addon

## Project Overview

This Firefox addon serves as a **browser automation bridge** for the OctaAI autonomous agent, enabling it to interact with web pages, fill forms, click buttons, extract data, and perform complex web-based tasks that go beyond simple HTTP requests.

## What This Addon Does

### Core Capabilities

1. **Page Interaction Automation**
   - Click buttons, links, and elements by selector or text
   - Fill form fields (text inputs, dropdowns, checkboxes, radio buttons)
   - Submit forms and navigate between pages
   - Execute JavaScript on the current page

2. **Data Extraction**
   - Extract text content from specific elements
   - Scrape structured data (tables, lists, articles)
   - Take screenshots of full pages or specific elements
   - Extract page metadata (title, description, structured data)

3. **Authentication & Session Management**
   - Handle login forms automatically
   - Maintain session cookies between tasks
   - Support multi-step authentication flows
   - Store credentials securely (encrypted in addon storage)

4. **Real-Time Page Monitoring**
   - Wait for elements to appear before interacting
   - Monitor page changes and dynamic content loading
   - Handle single-page applications (SPA) with dynamic routing
   - Detect and handle popups, alerts, and modals

5. **WebSocket Communication**
   - Establishes WebSocket connection to OctaAI agent daemon
   - Receives commands from the AI agent in real-time
   - Sends back results, errors, and page state updates
   - Maintains persistent connection for multi-step workflows

### Architecture

```
Firefox Browser
     |
     +-- OctaAI Browser Addon
          |
          +-- Content Scripts (inject into web pages)
          +-- Background Script (WebSocket client)
          +-- Popup UI (manual control & status)
          |
          +--[WebSocket]-- OctaAI Agent Daemon (localhost:8765)
                              |
                              +-- Browser Tool
```

## Technical Implementation

### Addon Structure

```
octaai-firefox-addon/
├── manifest.json           # Firefox addon manifest (Manifest V3)
├── background.js           # WebSocket client, command dispatcher
├── content/
│   ├── content.js          # Injected into web pages
│   └── page-api.js         # DOM interaction utilities
├── popup/
│   ├── popup.html          # Extension popup UI
│   ├── popup.js            # UI logic, connection status
│   └── popup.css           # Styling
└── lib/
    ├── crypto.js           # Encryption for stored credentials
    └── dom-utils.js        # Helper functions for DOM manipulation
```

### Communication Protocol

**WebSocket Messages (JSON)**

```json
// Command from Agent -> Addon
{
  "id": "cmd_12345",
  "type": "navigate|click|fill|extract|execute|screenshot",
  "params": {
    "url": "https://example.com",
    "selector": "#element-id",
    "value": "input text",
    "script": "return document.title;"
  },
  "timeout": 30000
}

// Response from Addon -> Agent
{
  "id": "cmd_12345",
  "status": "success|error",
  "result": "extracted data or execution result",
  "error": "error message if failed",
  "page_state": {
    "url": "current URL",
    "title": "current page title",
    "ready": true
  }
}
```

### Supported Commands

1. **navigate** - Navigate to URL
2. **click** - Click element by selector
3. **fill** - Fill form field
4. **submit** - Submit form
5. **extract** - Extract element text/data
6. **screenshot** - Take page screenshot
7. **execute** - Execute JavaScript
8. **wait_for** - Wait for element to appear
9. **scroll** - Scroll to element or position
10. **get_cookies** - Get current cookies
11. **set_cookies** - Set cookies for domain
12. **get_page_source** - Get page HTML source

## Development Guidelines

### When Generating Code

1. **Use Manifest V3** - Firefox's latest addon API
2. **Error Handling** - Every command must have timeout and error handling
3. **Security First**:
   - Validate all incoming commands
   - Sanitize user inputs before DOM injection
   - Encrypt stored credentials
   - Require user approval for sensitive actions
4. **Async/Await** - All operations are asynchronous
5. **Cross-Origin** - Handle CORS and cross-domain restrictions
6. **Logging** - Comprehensive logging for debugging

### Key Dependencies

```json
{
  "permissions": [
    "activeTab",
    "tabs",
    "storage",
    "webNavigation",
    "webRequest",
    "cookies",
    "<all_urls>"
  ],
  "host_permissions": [
    "<all_urls>"
  ]
}
```

### Example Usage Scenarios

**Scenario 1: Login & Extract Data**
```javascript
// 1. Navigate to login page
await browser.navigate("https://example.com/login");

// 2. Fill credentials
await browser.fill("#username", "user@example.com");
await browser.fill("#password", "securepass");

// 3. Submit form
await browser.submit("#login-form");

// 4. Wait for dashboard
await browser.wait_for(".dashboard");

// 5. Extract data
const data = await browser.extract(".user-info");
```

**Scenario 2: Form Submission**
```javascript
// Fill multi-step form
await browser.fill("#name", "John Doe");
await browser.fill("#email", "john@example.com");
await browser.click("button[type='submit']");
await browser.wait_for(".success-message");
```

## Integration with OctaAI Agent

### Browser Tool Usage

The OctaAI agent will have a `BrowserTool` that:
- Maintains WebSocket connection to the Firefox addon
- Sends high-level commands to the addon
- Receives structured results
- Handles connection errors and retries

### Example Agent Goal

```bash
octa-agent goal "Go to https://news.ycombinator.com, extract top 10 posts, and save to posts.json"
```

**Agent Execution:**
1. Creates "Browse HackerNews" task
2. Uses BrowserTool to connect to Firefox addon
3. Sends navigate command
4. Sends extract command with CSS selectors
5. Receives JSON data
6. Uses FilesystemTool to save to file

## Security Considerations

1. **User Consent** - All browser automation requires user approval
2. **Credential Storage** - Passwords encrypted with user's master password
3. **Domain Whitelist** - Option to restrict addon to specific domains
4. **Rate Limiting** - Prevent abuse of automation
5. **Audit Log** - Log all commands executed for review

## Testing Strategy

1. **Unit Tests** - Test individual command handlers
2. **Integration Tests** - Test WebSocket communication
3. **E2E Tests** - Test full workflows (login, extract, navigate)
4. **Manual Testing** - Test on popular sites (GitHub, Twitter, etc.)

## Future Enhancements

- [ ] Chrome extension support (same codebase)
- [ ] Visual element selection tool
- [ ] Recording mode (record user actions -> generate script)
- [ ] AI-powered element detection (find button by purpose)
- [ ] Headless browser fallback (Puppeteer integration)
- [ ] Multi-tab support
- [ ] Proxy support for automation

## Code Style

- Use modern JavaScript (ES2020+)
- Prefer `async/await` over callbacks
- Use `const` by default, `let` when needed
- Comprehensive JSDoc comments
- Error messages should be user-friendly
