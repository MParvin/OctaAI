/**
 * Popup Script - UI for the extension popup
 */

let currentStatus = null;

/**
 * Initialize popup
 */
async function initPopup() {
  console.log('[Popup] Initializing');

  updateStatus();
  setupEventListeners();
  loadActivityLogs();

  // Update status every 2 seconds
  setInterval(updateStatus, 2000);
}

/**
 * Update connection status
 */
async function updateStatus() {
  try {
    const response = await browser.runtime.sendMessage({ type: 'get_status' });
    currentStatus = response.wsStatus;

    updateStatusUI(currentStatus);
  } catch (error) {
    console.error('[Popup] Failed to get status:', error);
  }
}

/**
 * Update status UI elements
 */
function updateStatusUI(status) {
  const statusDot = document.getElementById('statusDot');
  const statusText = document.getElementById('statusText');
  const connectBtn = document.getElementById('connectBtn');

  if (!status) {
    statusDot.className = 'status-dot disconnected';
    statusText.textContent = 'Disconnected';
    connectBtn.textContent = 'Connect';
    return;
  }

  if (status.isConnected) {
    statusDot.className = 'status-dot connected';
    statusText.textContent = `Connected (${status.pendingCommands} commands)`;
    connectBtn.textContent = 'Disconnect';
  } else if (status.isConnecting) {
    statusDot.className = 'status-dot connecting';
    statusText.textContent = 'Connecting...';
    connectBtn.textContent = 'Connecting...';
    connectBtn.disabled = true;
  } else {
    statusDot.className = 'status-dot disconnected';
    statusText.textContent = 'Disconnected';
    connectBtn.textContent = 'Connect';
    connectBtn.disabled = false;
  }
}

/**
 * Setup event listeners
 */
function setupEventListeners() {
  document.getElementById('connectBtn').addEventListener('click', toggleConnection);
  document.getElementById('settingsBtn').addEventListener('click', openSettings);
  document.getElementById('refreshBtn').addEventListener('click', updateStatus);
  document.getElementById('clearLogsBtn').addEventListener('click', clearActivityLogs);
  document.querySelector('.settings-link a').addEventListener('click', (e) => {
    e.preventDefault();
    openSettings();
  });
}

/**
 * Toggle connection
 */
async function toggleConnection() {
  const btn = document.getElementById('connectBtn');

  if (currentStatus?.isConnected) {
    // Disconnect
    btn.disabled = true;
    btn.textContent = 'Disconnecting...';
    try {
      await browser.runtime.sendMessage({ type: 'disconnect' });
      await updateStatus();
    } catch (error) {
      console.error('[Popup] Failed to disconnect:', error);
    }
    btn.disabled = false;
  } else {
    // Connect
    btn.disabled = true;
    btn.textContent = 'Connecting...';
    try {
      await browser.runtime.sendMessage({ type: 'connect' });
      await updateStatus();
    } catch (error) {
      console.error('[Popup] Failed to connect:', error);
      alert('Failed to connect: ' + error.message);
    }
    btn.disabled = false;
  }
}

/**
 * Open settings page
 */
function openSettings() {
  browser.runtime.openOptionsPage();
  window.close();
}

/**
 * Load and display activity logs
 */
async function loadActivityLogs() {
  try {
    const response = await browser.runtime.sendMessage({ type: 'get_activity_logs' });
    const logs = response.logs || [];

    const listContainer = document.getElementById('activityList');
    const logCount = document.getElementById('logCount');

    if (logs.length === 0) {
      listContainer.innerHTML = '<div class="empty-state">No activity yet</div>';
      logCount.textContent = '0';
      return;
    }

    logCount.textContent = logs.length;
    listContainer.innerHTML = logs
      .slice(0, 20)
      .map(log => createLogEntryHTML(log))
      .join('');
  } catch (error) {
    console.error('[Popup] Failed to load logs:', error);
  }
}

/**
 * Create HTML for a log entry
 */
function createLogEntryHTML(log) {
  const time = new Date(log.timestamp).toLocaleTimeString();
  const isError = log.error || log.status === 'error';
  const message = log.message || log.commandType || log.type;

  return `
    <div class="log-entry ${isError ? 'log-error' : ''}">
      <span class="log-type">${log.type}</span>
      <span class="log-message" title="${message}">${message}</span>
      <span class="log-time">${time}</span>
    </div>
  `;
}

/**
 * Clear activity logs
 */
async function clearActivityLogs() {
  if (!confirm('Clear all activity logs?')) {
    return;
  }

  try {
    await browser.runtime.sendMessage({ type: 'clear_activity_logs' });
    loadActivityLogs();
  } catch (error) {
    console.error('[Popup] Failed to clear logs:', error);
  }
}

// Initialize when popup opens
if (document.readyState === 'loading') {
  document.addEventListener('DOMContentLoaded', initPopup);
} else {
  initPopup();
}
