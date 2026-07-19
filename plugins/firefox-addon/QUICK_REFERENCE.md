# Quick Reference - OctaAI Firefox Extension

## Setup (Copy & Paste)

```bash
cd /home/mparvin/Documents/MyGit/Projects/OctaAI/plugins/firefox-addon
npm install
npm run build
```

## Development (3 Terminals)

**Terminal 1**: Watch & rebuild
```bash
npm run dev
```

**Terminal 2**: Start daemon
```bash
cd ../../
./bin/octa-agentd --browser-port 8765
```

**Terminal 3**: Load in Firefox
```bash
npm start
```

## Common Commands

| Task | Command |
|------|---------|
| Build | `npm run build` |
| Watch | `npm run dev` |
| Lint | `npm run lint` |
| Test | `npm run test` (when implemented) |
| Start | `npm start` |
| Package | `npm run package` |
| Clean | `npm run clean` |

## Folder Structure

```
firefox-addon/
├── src/
│   ├── background/         # Service worker
│   ├── content/            # DOM manipulation
│   ├── popup/              # Status UI
│   ├── options/            # Settings
│   ├── shared/             # Utilities
│   └── icons/              # Images
├── dist/                   # Built extension
├── manifest.json           # Extension config
├── package.json            # Dependencies
├── webpack.config.js       # Build config
├── README.md               # User guide
├── DEVELOPMENT.md          # Dev guide
├── TESTING.md              # Test guide
└── ARCHITECTURE.md         # Technical docs
```

## 12 Commands Reference

### 1. Navigate
```json
{
  "type": "navigate",
  "params": {"url": "https://example.com"}
}
```

### 2. Click
```json
{
  "type": "click",
  "params": {"selector": ".button"}
}
```
Or with text: `"text": "Click me"`
Or with XPath: `"xpath": "//button[contains(text(), 'Click')]"`

### 3. Fill
```json
{
  "type": "fill",
  "params": {
    "selector": "input[name='email']",
    "value": "user@example.com",
    "typeDelay": 50
  }
}
```

### 4. Submit
```json
{
  "type": "submit",
  "params": {"selector": "form"}
}
```

### 5. Extract
```json
{
  "type": "extract",
  "params": {
    "selector": ".product-price",
    "multiple": true,
    "attribute": null
  }
}
```

### 6. Screenshot
```json
{
  "type": "screenshot",
  "params": {}
}
```

### 7. Execute
```json
{
  "type": "execute",
  "params": {"code": "return document.title;"}
}
```

### 8. Wait For
```json
{
  "type": "wait_for",
  "params": {
    "selector": ".loaded",
    "timeout": 10000
  }
}
```

### 9. Scroll
```json
{
  "type": "scroll",
  "params": {
    "selector": ".target",
    "block": "center"
  }
}
```

### 10. Get Cookies
```json
{
  "type": "get_cookies",
  "params": {}
}
```

### 11. Set Cookies
```json
{
  "type": "set_cookies",
  "params": {
    "cookies": [
      {"name": "session", "value": "abc123"}
    ]
  }
}
```

### 12. Get Page Source
```json
{
  "type": "get_page_source",
  "params": {}
}
```

## Testing Commands

```bash
# Test 1: Navigation
./bin/octa-agent goal "Navigate to https://example.com"

# Test 2: Screenshot
./bin/octa-agent goal "Take a screenshot"

# Test 3: Form filling
./bin/octa-agent goal "Fill email and click submit"

# Check status
./bin/octa-agent status

# View logs
./bin/octa-agent logs <goal-id>
```

## Settings Configuration

**Location**: Firefox → Extensions → OctaAI → Settings

**Parameters**:
- **Server URL**: `ws://localhost:8765/ws`
- **Token**: (optional, if daemon requires)
- **Auto-connect**: Yes/No
- **Command Timeout**: 30000ms
- **Typing Delay**: 50ms

## File Locations

| File | Purpose |
|------|---------|
| `src/background/background.js` | WebSocket connection |
| `src/content/content.js` | DOM commands (all 12) |
| `src/popup/popup.html` | Status UI |
| `src/options/options.html` | Settings page |
| `manifest.json` | Extension metadata |
| `package.json` | Build dependencies |

## Debugging Checklist

- [ ] Check browser console (F12)
- [ ] Look for `[Background]`, `[Content]`, `[Popup]` logs
- [ ] Open `about:debugging` to inspect extension
- [ ] Verify daemon is running: `ps aux | grep octa-agentd`
- [ ] Test WebSocket: Open options page → "Test Connection"
- [ ] Check Firefox permissions accepted all prompts
- [ ] Clear extension cache: `about:addons` → Settings

## Common Errors & Fixes

| Error | Solution |
|-------|----------|
| `WebSocket connection failed` | Check daemon URL in Settings |
| `Element not found` | Verify selector with Inspector (F12) |
| `Command timeout` | Increase timeout in Settings |
| `Memory high` | Clear activity logs in popup |
| `Extension won't load` | Check manifest.json syntax |
| `Permission denied` | Restart Firefox |

## Key Files to Edit

### Add New Command
**File**: `src/content/content.js`

```javascript
async function cmdMyCommand(params) {
  // Implementation
  return result;
}

// Add to switch in executeCommand():
case 'my_command':
  result = await cmdMyCommand(command.params);
  break;
```

### Change Default Settings
**File**: `src/options/options.js`

```javascript
const DEFAULT_CONFIG = {
  serverUrl: 'ws://localhost:8765/ws',
  token: '',
  autoConnect: true,
  commandTimeout: 30000,
  typeDelay: 50
};
```

### Add Storage
**File**: `src/shared/storage-manager.js`

```javascript
async saveMyData(data) {
  return new Promise((resolve) => {
    browser.storage.sync.set({ myData: data }, resolve);
  });
}
```

## Links

- **GitHub**: [octaai-firefox-addon](https://github.com/...)
- **Firefox Add-ons**: https://addons.mozilla.org/
- **WebExtensions API**: https://developer.mozilla.org/en-US/docs/Mozilla/Add-ons/WebExtensions
- **OctaAI Docs**: ../../docs/

## Performance Targets

| Metric | Target |
|--------|--------|
| Command latency | <100ms |
| Memory usage | <50MB |
| Success rate | >95% |
| Reconnection | <5 seconds |
| SPA support | Full |

## Phase Checklist

- [x] Phase 1: Project Setup
- [x] Phase 2: WebSocket Layer
- [x] Phase 3: Command Router
- [x] Phase 4: DOM Operations
- [x] Phase 5: User Interface
- [ ] Phase 6: Storage & Configuration
- [ ] Phase 7: Error Handling
- [ ] Phase 8: Testing
- [ ] Phase 9: Documentation
- [ ] Phase 10: Distribution

## Next Steps

1. ✅ Build extension: `npm run build`
2. ✅ Start daemon: `./bin/octa-agentd --browser-port 8765`
3. ✅ Load in Firefox: `npm start`
4. ⏳ Test all 12 commands
5. ⏳ Run test suite
6. ⏳ Submit to Firefox Add-ons

---

**Stuck?** Check [DEVELOPMENT.md](DEVELOPMENT.md) or [ARCHITECTURE.md](ARCHITECTURE.md)
