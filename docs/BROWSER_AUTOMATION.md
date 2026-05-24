# Browser Automation with OctaAI

OctaAI can control a live Firefox browser through a companion extension. This lets the agent navigate websites, fill forms, extract data, and automate multi-step web workflows — all driven by natural-language goals.

## Architecture

```
octa-agentd  ──WebSocket──►  Firefox Extension  ──DOM API──►  Web Page
     │                         background.js        content.js
     │
     └── pkg/tools/browser.go   (browser tool — calls SendCommandToAny)
     └── pkg/browser/server.go  (WebSocket server on ws://localhost:8765)
```

The daemon exposes a WebSocket server. The Firefox extension connects to it on startup, authenticates with a shared token, and relays commands from the agent to the active browser tab.

---

## Installation

### 1. Build and start the daemon

```bash
make build
./bin/octa-agentd --browser-port 8765
```

### 2. Install the Firefox extension

1. Open Firefox and navigate to `about:debugging`.
2. Click **This Firefox** → **Load Temporary Add-on**.
3. Select `octaai-firefox-addon/dist/manifest.json`.

To build the extension first:

```bash
cd ../octaai-firefox-addon
npm install
npm run build        # production build → dist/
```

### 3. Configure a shared token

Generate a random token:

```bash
uuidgen
# e.g. a1b2c3d4-e5f6-7890-abcd-ef1234567890
```

Add it to `config.yaml`:

```yaml
browser:
  enabled: true
  port: 8765
  token: "a1b2c3d4-e5f6-7890-abcd-ef1234567890"
  auto_screenshot: true
```

Open the extension settings in Firefox (**⚙ Settings** in the popup) and paste the same token.

### 4. Verify the connection

The extension popup should show a teal **Connected** indicator within a few seconds of the daemon starting. The daemon logs `browser client connected` to stdout.

---

## Configuration Reference

All browser settings live under the `browser:` key in `config.yaml`:

| Key              | Type     | Default | Description                                         |
|------------------|----------|---------|-----------------------------------------------------|
| `enabled`        | bool     | `false` | Enable the WebSocket server                         |
| `port`           | int      | `8765`  | Port the daemon listens on                          |
| `token`          | string   | `""`    | Shared secret for extension authentication          |
| `auto_screenshot`| bool     | `false` | Capture a screenshot automatically on tool error    |
| `browser_domains`| []string | `[]`    | Optional domain whitelist (empty = allow all)       |

The `--browser-port` flag on `octa-agentd` overrides `browser.port`.

---

## Supported Browser Actions

The `browser` tool accepts an `action` parameter and action-specific params:

| Action           | Required params         | Optional params                      |
|------------------|-------------------------|--------------------------------------|
| `navigate`       | `url`                   | —                                    |
| `click`          | `selector` or `text`    | `xpath`                              |
| `fill`           | (`selector`\|`text`), `value` | `xpath`                       |
| `submit`         | `selector` or `text`    | `xpath`                              |
| `extract`        | `selector` or `text`    | `xpath`, `attribute`, `multiple`     |
| `screenshot`     | `output_path`           | `selector`                           |
| `execute`        | `script`                | —                                    |
| `wait_for`       | `selector` or `text`    | `xpath`, `timeout`, `interval`       |
| `scroll`         | —                       | `selector`, `x`, `y`                 |
| `get_page_source`| —                       | —                                    |
| `get_cookies`    | —                       | `domain`                             |

### Element selectors

Every action that targets a DOM element accepts one of three locator strategies (evaluated in order):

- **`xpath`** — XPath expression, e.g. `//button[text()='Submit']`
- **`selector`** — CSS selector, e.g. `#login-form input[name="email"]`
- **`text`** — case-insensitive substring of visible text, e.g. `"Sign in"`

---

## Example Goals

```bash
# Extract the main heading from a page
./bin/octa-agent goal "Navigate to https://example.com and extract the h1 heading text"

# Fill and submit a form
./bin/octa-agent goal "Go to https://httpbin.org/forms/post, fill the custname field with 'Alice', fill the custemail field with 'alice@example.com', and submit the form"

# Multi-step data extraction
./bin/octa-agent goal "Go to https://github.com/trending, extract the names of the top 5 repositories, and save them to /tmp/trending.txt"

# Login flow
./bin/octa-agent goal "Navigate to https://example.com/login, fill in username 'admin' and password stored in env var APP_PASSWORD, then click the Login button"
```

---

## Security

### Authentication

The extension sends its token as a query-parameter when connecting:
```
ws://localhost:8765?token=<token>
```
The daemon rejects connections with a missing or incorrect token immediately.

### Token storage

- **Daemon side**: stored in `config.yaml` (should be `chmod 600`).
- **Extension side**: stored in Firefox's `storage.sync` (encrypted by the browser profile).

Never commit `config.yaml` to source control. The repository's `.gitignore` already excludes it.

### Domain whitelist

Set `browser_domains` in `config.yaml` to restrict which sites the agent is permitted to interact with:

```yaml
browser:
  browser_domains:
    - "github.com"
    - "*.google.com"
```

### Command injection

The `execute` action runs arbitrary JavaScript inside the active tab. Restrict access to the daemon socket to trusted local processes only. Do not expose port 8765 on a network interface.

### Content Security Policy

The extension injects `content.js` only on explicit request from the background script (`scripting.executeScript`), not automatically on every page load, minimising the attack surface.

---

## Troubleshooting

| Symptom | Likely cause | Fix |
|---|---|---|
| Popup shows "Disconnected" | Daemon not running or wrong port | Start `octa-agentd`, check port in both config and extension settings |
| "Authentication failed" in popup log | Token mismatch | Copy the exact token from `config.yaml` into extension settings |
| `browser tool: no browser connected` | Extension not installed / page not loaded | Load the extension in `about:debugging` |
| Command times out | Page load takes longer than `timeout` | Increase `timeout` param in the goal description |
| Screenshots empty | `auto_screenshot` requires a connected browser | Ensure the extension is connected before running goals |

---

## Development

To rebuild the extension after making changes to addon source:

```bash
cd ../octaai-firefox-addon
npm run build:dev      # development build with source maps
npm run watch          # incremental rebuild on save
npm test               # run Jest unit tests
```

Reload the extension in Firefox after each build: `about:debugging` → **Reload**.
