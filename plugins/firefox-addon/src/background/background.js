/**
 * Background Script - Main service worker for the extension
 * Manages WebSocket connection and routes commands
 */

// Global WebSocket manager instance
let wsManager = null;
let storageManager = null;
let commandHandler = null;

/**
 * Initialize the background service
 */
async function initialize() {
  console.log('[Background] Initializing OctaAI addon');

  // Load shared classes (will be bundled by webpack)
  // These are injected at runtime by webpack

  // Initialize managers
  storageManager = new StorageManager();
  commandHandler = new CommandHandler();

  // Load configuration
  const config = await storageManager.loadConfig();
  console.log('[Background] Configuration loaded:', { ...config, token: '***' });

  // Initialize WebSocket manager
  wsManager = new WebSocketManager({
    url: config.serverUrl,
    token: config.token,
    autoReconnect: config.autoConnect,
    commandTimeout: config.commandTimeout
  });

  // Set up event handlers
  wsManager.onConnectionStateChange((state) => {
    console.log('[Background] Connection state changed:', state);
    notifyPopup({
      type: 'connection_state',
      data: state
    });
  });

  wsManager.onMessage((message) => {
    handleDaemonMessage(message);
  });

  // Auto-connect if enabled
  if (config.autoConnect) {
    connectToDaemon().catch(error => {
      console.error('[Background] Failed to auto-connect:', error);
    });
  }
}

/**
 * Connect to the OctaAI daemon
 */
async function connectToDaemon() {
  try {
    await wsManager.connect();
    console.log('[Background] Connected to daemon');
    await storageManager.addActivityLog({
      type: 'connection',
      message: 'Successfully connected to daemon'
    });
  } catch (error) {
    console.error('[Background] Connection failed:', error);
    await storageManager.addActivityLog({
      type: 'connection',
      message: `Connection failed: ${error.message}`,
      error: true
    });
    throw error;
  }
}

/**
 * Handle messages from the daemon
 */
async function handleDaemonMessage(message) {
  console.log('[Background] Received message from daemon:', message);

  if (message.type === 'command') {
    // Route command to appropriate tab
    handleCommandForTab(message);
  } else if (message.type === 'task_update') {
    // Broadcast task updates to popup
    notifyPopup({
      type: 'task_update',
      data: message
    });
  }

  // Log activity
  await storageManager.addActivityLog({
    type: 'message_received',
    commandType: message.type,
    commandId: message.id
  });
}

/**
 * Route a command to the appropriate content script
 */
async function handleCommandForTab(command) {
  const tabs = await browser.tabs.query({});
  const targetTab = tabs.find(tab => tab.id === command.targetTabId) || tabs[0];

  if (!targetTab) {
    console.error('[Background] No target tab found');
    // Send error response back to daemon
    await wsManager.sendCommand({
      type: 'command_result',
      commandId: command.id,
      status: 'error',
      error: 'No target tab found'
    });
    return;
  }

  try {
    // Send command to content script
    const result = await browser.tabs.sendMessage(targetTab.id, {
      type: 'execute_command',
      command
    });

    // Send result back to daemon
    await wsManager.sendCommand({
      type: 'command_result',
      commandId: command.id,
      status: 'success',
      result,
      tabId: targetTab.id
    });

    await storageManager.addActivityLog({
      type: 'command_executed',
      commandType: command.type,
      commandId: command.id,
      status: 'success'
    });
  } catch (error) {
    console.error('[Background] Failed to execute command:', error);

    // Send error response back to daemon
    await wsManager.sendCommand({
      type: 'command_result',
      commandId: command.id,
      status: 'error',
      error: error.message
    });

    await storageManager.addActivityLog({
      type: 'command_executed',
      commandType: command.type,
      commandId: command.id,
      status: 'error',
      error: error.message
    });
  }
}

/**
 * Notify the popup of updates
 */
function notifyPopup(message) {
  browser.runtime.sendMessage(message).catch(() => {
    // Popup not open, ignore
  });
}

/**
 * Handle messages from popup or content scripts
 */
browser.runtime.onMessage.addListener((message, sender, sendResponse) => {
  console.log('[Background] Received message:', message);

  if (message.type === 'get_status') {
    sendResponse({
      wsStatus: wsManager ? wsManager.getStatus() : null,
      timestamp: Date.now()
    });
  } else if (message.type === 'connect') {
    connectToDaemon()
      .then(() => {
        sendResponse({ success: true });
      })
      .catch(error => {
        sendResponse({ success: false, error: error.message });
      });
    return true; // Indicate we'll respond asynchronously
  } else if (message.type === 'disconnect') {
    if (wsManager) {
      wsManager.disconnect();
    }
    sendResponse({ success: true });
  } else if (message.type === 'update_config') {
    storageManager.saveConfig(message.config)
      .then(() => {
        // Reinitialize with new config
        initialize().then(() => {
          sendResponse({ success: true });
        });
      })
      .catch(error => {
        sendResponse({ success: false, error: error.message });
      });
    return true;
  } else if (message.type === 'get_activity_logs') {
    storageManager.getActivityLogs()
      .then(logs => {
        sendResponse({ logs });
      });
    return true;
  } else if (message.type === 'clear_activity_logs') {
    storageManager.clearActivityLogs()
      .then(() => {
        sendResponse({ success: true });
      });
  }
});

/**
 * Handle tab updates
 */
browser.tabs.onUpdated.addListener((tabId, changeInfo, tab) => {
  if (changeInfo.status === 'complete') {
    console.log('[Background] Tab loaded:', tabId, tab.url);

    // Notify popup if needed
    notifyPopup({
      type: 'tab_updated',
      tabId,
      tab
    });
  }
});

/**
 * Initialize on load
 */
initialize().catch(error => {
  console.error('[Background] Initialization failed:', error);
});
