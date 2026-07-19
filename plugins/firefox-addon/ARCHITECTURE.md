# Architecture - OctaAI Firefox Extension

## System Overview

```
┌─────────────────────────────────────────────────────────────────┐
│ OctaAI Daemon                                                   │
│ (octa-agentd)                                                   │
│                                                                 │
│ ┌──────────────────────────────────────────────────────────┐   │
│ │ WebSocket Server (default: ws://localhost:8765/ws)         │   │
│ └────────────────────┬─────────────────────────────────────┘   │
└─────────────────────┼──────────────────────────────────────────┘
                      │
                      │ WebSocket Messages
                      │ (Commands & Responses)
                      │
                      ▼
┌─────────────────────────────────────────────────────────────────┐
│ Firefox Browser                                                 │
│                                                                 │
│ ┌────────────────────────────────────────────────────────────┐ │
│ │ Content Script (Per Tab)                                  │ │
│ │ • DOM Manipulation                                         │ │
│ │ • Click, Fill, Extract, etc.                             │ │
│ │ • Page Context Execution                                  │ │
│ └──────┬───────────────────────────────────────────┬────────┘ │
│        │ Message                                   │           │
│        │                                           │           │
│ ┌──────▼───────────────────────────────────────────▼────────┐ │
│ │ Background Script (Service Worker)                        │ │
│ │ • WebSocket Connection Management                         │ │
│ │ • Command Router                                          │ │
│ │ • Configuration Storage                                   │ │
│ │ • Activity Logging                                        │ │
│ │ • Tab Management                                          │ │
│ └────────────────────────────────────────────────────────────┘ │
│                                                                 │
│ ┌────────────────────────────────────────────────────────────┐ │
│ │ Popup UI                                                  │ │
│ │ • Connection Status                                        │ │
│ │ • Activity Log                                             │ │
│ │ • Quick Controls                                           │ │
│ └────────────────────────────────────────────────────────────┘ │
│                                                                 │
│ ┌────────────────────────────────────────────────────────────┐ │
│ │ Options Page                                              │ │
│ │ • Server URL Configuration                               │ │
│ │ • Token Management                                         │ │
│ │ • Timeout Settings                                         │ │
│ │ • Connection Testing                                       │ │
│ └────────────────────────────────────────────────────────────┘ │
│                                                                 │
│ ┌────────────────────────────────────────────────────────────┐ │
│ │ Browser Storage API                                       │ │
│ │ • Configuration (sync storage)                            │ │
│ │ • Activity Logs (sync storage)                            │ │
│ └────────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────┘
```

## Component Architecture

### 1. Background Service Worker

**File**: `src/background/background.js`

**Responsibilities**:
- Manage WebSocket connection lifecycle
- Route commands to appropriate content script
- Manage extension configuration
- Log activity and errors
- Handle tab events

**Key Objects**:
```javascript
wsManager: WebSocketManager    // WebSocket connection
storageManager: StorageManager // Configuration & logs
commandHandler: CommandHandler // Command routing
```

**Message Handlers**:
- `execute_command` - From content script
- `get_status` - Request connection status
- `connect` - Initiate connection
- `disconnect` - Close connection
- `update_config` - Update settings
- `get_activity_logs` - Retrieve logs
- `clear_activity_logs` - Clear log history

### 2. Content Script

**File**: `src/content/content.js`

**Responsibilities**:
- Execute all 12 command types on the DOM
- Handle page interactions (click, type, etc.)
- Extract page data
- Manage screenshots and page state
- Report results to background script

**Command Implementations**:
```javascript
cmdNavigate()        // Navigate to URL
cmdClick()          // Click element
cmdFill()           // Fill form field
cmdSubmit()         // Submit form
cmdExtract()        // Extract data
cmdScreenshot()     // Take screenshot
cmdExecute()        // Execute JS
cmdWaitFor()        // Wait for element
cmdScroll()         // Scroll page
cmdGetCookies()     // Get cookies
cmdSetCookies()     // Set cookies
cmdGetPageSource()  // Get HTML
```

**Lifecycle**:
- Injected into every tab automatically
- Persists across page navigation (in same tab)
- Can receive multiple commands concurrently
- Cleans up on tab close

### 3. Popup UI

**Files**: `src/popup/popup.html`, `src/popup/popup.js`

**Responsibilities**:
- Display connection status
- Show recent activity
- Provide quick controls
- Link to settings page

**Features**:
- Real-time status updates (every 2 seconds)
- Activity log with timestamps
- Connect/Disconnect buttons
- Settings access
- Clear logs function

### 4. Options Page

**Files**: `src/options/options.html`, `src/options/options.js`

**Responsibilities**:
- Allow user to configure settings
- Save configuration to storage
- Validate inputs
- Test connection

**Settings**:
```javascript
{
  serverUrl: string,      // WebSocket server URL
  token: string,          // Authentication token
  autoConnect: boolean,   // Auto-connect on startup
  commandTimeout: number, // Command timeout (ms)
  typeDelay: number       // Typing delay (ms)
}
```

### 5. Shared Utilities

#### WebSocketManager (`src/shared/websocket-manager.js`)

**Features**:
- Token-based authentication
- Automatic reconnection with exponential backoff
- Heartbeat/ping mechanism (30s interval)
- Command queuing with timeout handling
- Connection state tracking

**Key Methods**:
```javascript
connect()                // Connect to server
disconnect()             // Disconnect
sendCommand()            // Send command, wait for response
onMessage()              // Register message handler
onConnectionStateChange() // Register state change handler
getStatus()              // Get connection status
```

**Backoff Strategy**:
```
Initial delay: 3 seconds
Max delay: 30 seconds
Multiplier: 1.5x each retry
```

#### CommandHandler (`src/shared/command-handler.js`)

**Features**:
- Route commands to appropriate handlers
- Manage command lifecycle
- Error handling per command

#### StorageManager (`src/shared/storage-manager.js`)

**Features**:
- Load/save configuration
- Activity log management
- Automatic log truncation (50 entries max)

## Data Flow

### 1. Command Execution Flow

```
Daemon
  ↓ (sends command via WebSocket)
Background Script
  ├─ Parses command JSON
  ├─ Determines target tab
  ├─ Sends to Content Script
  │
Content Script
  ├─ Executes command
  ├─ Manipulates DOM
  ├─ Waits for result
  ├─ Gathers page state
  │
Background Script (receives response)
  ├─ Sends result back to Daemon via WebSocket
  └─ Logs activity
```

### 2. Configuration Flow

```
User → Options Page
  ↓ (saves settings)
Browser Storage (sync)
  ↓
Background Script
  ├─ Loads on startup
  ├─ Initializes managers
  ├─ Creates WebSocket with config
  │
Popup UI
  ├─ Polls for status
  ├─ Shows connection state
  └─ Updates activity logs
```

## Message Protocol

### Command Message (Daemon → Extension)

```typescript
interface Command {
  id: string;              // Unique command ID
  type: string;            // Command type (click, fill, etc.)
  targetTabId?: number;    // Specific tab ID
  params: {
    selector?: string;
    xpath?: string;
    value?: string;
    [key: string]: any;
  };
  timeout?: number;        // Max execution time
}
```

### Response Message (Extension → Daemon)

```typescript
interface Response {
  id: string;              // Matches command ID
  status: 'success' | 'error';
  result?: any;            // Command-specific result
  error?: string;          // Error message if failed
  tabId?: number;          // Tab where executed
  pageState?: {
    url: string;
    title: string;
    scrollY: number;
    scrollX: number;
  };
  timestamp: number;
}
```

## State Management

### Global State in Background

```javascript
// WebSocket connection state
wsManager.isConnected: boolean
wsManager.isConnecting: boolean
wsManager.pendingCommands: Map

// Configuration state
config: {
  serverUrl: string,
  token: string,
  autoConnect: boolean,
  commandTimeout: number,
  typeDelay: number
}

// Activity logs (in storage)
activityLogs: Array<{
  type: string,
  message: string,
  timestamp: number,
  error?: boolean
}>
```

### Per-Tab State in Content Script

```javascript
// Current executing command
currentCommand: Command | null

// Command queue
commandQueue: Array<Command>

// Page state
pageState: {
  url: string,
  title: string,
  scrollY: number,
  scrollX: number
}
```

## Error Handling

### Hierarchy

```
Command Execution Error
├─ Element not found
│  └─ Retry with different selector
├─ Timeout
│  └─ Return timeout error
├─ Navigation failed
│  └─ Return navigation error
└─ Invalid selector
   └─ Return selector error

Connection Error
├─ Network unreachable
│  └─ Auto-reconnect (exponential backoff)
├─ Authentication failed
│  └─ Log error, require new token
└─ Timeout
   └─ Retry with exponential backoff
```

### Error Propagation

```
Content Script (throws error)
  ↓
Background Script (catches)
  ├─ Logs to activity
  ├─ Sends error response to Daemon
  └─ Notifies Popup
```

## Extension Lifecycle

### Installation

```
1. User installs from Firefox Add-ons
2. manifest.json parsed
3. Background script loaded
4. initialize() called
5. Configuration loaded
6. WebSocket created
7. Auto-connect if enabled
```

### Runtime

```
User opens Firefox
  ↓
Background script starts
  ↓
Load configuration
  ↓
Create WebSocket connection
  ↓
Listen for messages
  ├─ Daemon commands → route to tab
  ├─ Popup queries → respond with status
  └─ Settings changes → reinitialize
```

### Shutdown

```
User closes Firefox
  ↓
Background script unload
  ↓
WebSocket closes
  ↓
Pending commands cancelled
  ↓
Activity logs saved
```

## Performance Characteristics

### Memory Usage

| Component | Memory |
|-----------|--------|
| Background script | ~5MB |
| Content script per tab | ~2MB |
| UI components | ~1MB |
| **Total** | **~8-10MB base + 2MB/tab** |

### Latency

| Operation | Latency |
|-----------|---------|
| DOM query | <5ms |
| Click | ~50ms |
| Fill | ~300ms (with typing animation) |
| Navigate | ~1000ms |
| Screenshot | ~500ms |
| Extract | <10ms |

### Throughput

| Metric | Value |
|--------|-------|
| Commands per second | 10-20 |
| Concurrent commands | 3-5 |
| Queue size | Unlimited |
| Message size limit | 256MB (WebSocket) |

## Security Model

### Threat Model

1. **Malicious Daemon**: Could send harmful commands
   - Mitigated by: Token authentication

2. **Page XSS**: Could intercept commands
   - Mitigated by: Content script isolation

3. **Man-in-the-middle**: Could intercept WebSocket
   - Mitigated by: Use WSS (WebSocket Secure) in production

4. **Privilege Escalation**: Content script limited
   - Mitigated by: Sandbox restrictions

### Trust Boundaries

```
Untrusted: Website JavaScript
  ↓ (isolated)
Content Script (Untrusted)
  ↓ (message passing)
Background Script (Semi-trusted)
  ↓ (authentication + URL validation)
Daemon (Trusted)
```

## Extensibility

### Adding New Commands

1. Implement in `content.js`:
```javascript
async function cmdNewCommand(params) {
  // Implementation
  return result;
}
```

2. Register in switch statement:
```javascript
case 'new_command':
  result = await cmdNewCommand(command.params);
  break;
```

3. Document in README.md

### Custom Event Handlers

```javascript
// In popup.js
wsManager.onMessage((message) => {
  if (message.type === 'custom_event') {
    // Handle custom event
  }
});
```

### Storage Extension

```javascript
// In storage-manager.js
async saveCustomData(key, data) {
  return new Promise((resolve) => {
    browser.storage.sync.set({ [key]: data }, resolve);
  });
}
```

---

**See also**:
- [DEVELOPMENT.md](DEVELOPMENT.md) - Development workflow
- [README.md](README.md) - User guide
- [TESTING.md](TESTING.md) - Testing strategy
