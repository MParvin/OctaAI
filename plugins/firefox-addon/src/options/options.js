/**
 * Options Script - Handle settings page
 */

// Default configuration values
const DEFAULT_CONFIG = {
  serverUrl: 'ws://localhost:8765/ws',
  token: '',
  autoConnect: true,
  commandTimeout: 30000,
  typeDelay: 50
};

/**
 * Initialize options page
 */
async function initOptions() {
  console.log('[Options] Initializing');

  const config = await loadConfig();
  populateForm(config);

  // Setup event listeners
  document.getElementById('settingsForm').addEventListener('submit', saveSettings);
  document.getElementById('resetBtn').addEventListener('click', resetToDefaults);
  document.getElementById('testConnection').addEventListener('click', testConnection);
}

/**
 * Load configuration from storage
 */
async function loadConfig() {
  return new Promise((resolve) => {
    browser.storage.sync.get('config', (result) => {
      resolve(result.config || DEFAULT_CONFIG);
    });
  });
}

/**
 * Save configuration to storage
 */
async function saveConfig(config) {
  return new Promise((resolve) => {
    browser.storage.sync.set({ config }, resolve);
  });
}

/**
 * Populate form with configuration values
 */
function populateForm(config) {
  document.getElementById('serverUrl').value = config.serverUrl || DEFAULT_CONFIG.serverUrl;
  document.getElementById('token').value = config.token || '';
  document.getElementById('autoConnect').checked = config.autoConnect !== false;
  document.getElementById('commandTimeout').value = config.commandTimeout || DEFAULT_CONFIG.commandTimeout;
  document.getElementById('typeDelay').value = config.typeDelay || DEFAULT_CONFIG.typeDelay;
}

/**
 * Get form values
 */
function getFormValues() {
  return {
    serverUrl: document.getElementById('serverUrl').value.trim(),
    token: document.getElementById('token').value.trim(),
    autoConnect: document.getElementById('autoConnect').checked,
    commandTimeout: parseInt(document.getElementById('commandTimeout').value, 10) || DEFAULT_CONFIG.commandTimeout,
    typeDelay: parseInt(document.getElementById('typeDelay').value, 10) || DEFAULT_CONFIG.typeDelay
  };
}

/**
 * Save settings
 */
async function saveSettings(e) {
  e.preventDefault();

  const config = getFormValues();

  // Validate URL
  try {
    new URL(config.serverUrl.replace('ws://', 'http://').replace('wss://', 'https://'));
  } catch (error) {
    showStatus('Invalid server URL', 'error');
    return;
  }

  try {
    await saveConfig(config);
    showStatus('Settings saved successfully!', 'success');

    // Notify background script to update configuration
    browser.runtime.sendMessage({
      type: 'update_config',
      config
    }).catch(() => {
      // Background script might not be ready
    });
  } catch (error) {
    console.error('[Options] Failed to save settings:', error);
    showStatus('Failed to save settings: ' + error.message, 'error');
  }
}

/**
 * Reset to defaults
 */
async function resetToDefaults() {
  if (!confirm('Reset all settings to default values?')) {
    return;
  }

  populateForm(DEFAULT_CONFIG);
  await saveConfig(DEFAULT_CONFIG);
  showStatus('Settings reset to defaults', 'success');
}

/**
 * Test connection to daemon
 */
async function testConnection() {
  const config = getFormValues();
  const button = document.getElementById('testConnection');
  const resultDiv = document.getElementById('testResult');

  button.disabled = true;
  button.textContent = 'Testing...';
  resultDiv.style.display = 'none';

  try {
    // Create a test WebSocket connection
    const wsUrl = config.token ? `${config.serverUrl}?token=${config.token}` : config.serverUrl;

    const testPromise = new Promise((resolve, reject) => {
      const timeout = setTimeout(() => {
        reject(new Error('Connection timeout (10s)'));
      }, 10000);

      try {
        const ws = new WebSocket(wsUrl);

        ws.onopen = () => {
          clearTimeout(timeout);
          ws.close();
          resolve('Connection successful!');
        };

        ws.onerror = (error) => {
          clearTimeout(timeout);
          reject(new Error('Connection failed: ' + (error.message || 'Unknown error')));
        };

        ws.onclose = () => {
          clearTimeout(timeout);
          if (ws.readyState === WebSocket.CLOSED) {
            resolve('Connection closed by server');
          }
        };
      } catch (error) {
        clearTimeout(timeout);
        reject(error);
      }
    });

    await testPromise;
    resultDiv.style.color = '#2e7d32';
    resultDiv.innerHTML = '✓ Successfully connected to daemon';
    resultDiv.style.display = 'block';
  } catch (error) {
    console.error('[Options] Connection test failed:', error);
    resultDiv.style.color = '#c62828';
    resultDiv.innerHTML = '✗ ' + error.message;
    resultDiv.style.display = 'block';
  } finally {
    button.disabled = false;
    button.textContent = 'Test Connection';
  }
}

/**
 * Show status message
 */
function showStatus(message, type = 'success') {
  const statusDiv = document.getElementById('statusMessage');
  statusDiv.textContent = message;
  statusDiv.className = `status-message show ${type}`;

  setTimeout(() => {
    statusDiv.classList.remove('show');
  }, 4000);
}

// Initialize when page loads
if (document.readyState === 'loading') {
  document.addEventListener('DOMContentLoaded', initOptions);
} else {
  initOptions();
}
