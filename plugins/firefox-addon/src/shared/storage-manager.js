/**
 * StorageManager - Handles configuration and activity logging
 */
class StorageManager {
  constructor() {
    this.storageKey = 'octaai-addon';
  }

  /**
   * Load configuration
   */
  async loadConfig() {
    return new Promise((resolve) => {
      browser.storage.sync.get('config', (result) => {
        resolve(result.config || this.getDefaultConfig());
      });
    });
  }

  /**
   * Save configuration
   */
  async saveConfig(config) {
    return new Promise((resolve) => {
      browser.storage.sync.set({ config }, resolve);
    });
  }

  /**
   * Get default configuration
   */
  getDefaultConfig() {
    return {
      serverUrl: 'ws://localhost:8765/ws',
      token: '',
      autoConnect: true,
      commandTimeout: 30000,
      maxActivityLogs: 50
    };
  }

  /**
   * Add activity log entry
   */
  async addActivityLog(entry) {
    return new Promise((resolve) => {
      browser.storage.sync.get('activityLogs', (result) => {
        let logs = result.activityLogs || [];
        logs = [
          {
            ...entry,
            timestamp: Date.now()
          },
          ...logs
        ].slice(0, 50);
        browser.storage.sync.set({ activityLogs: logs }, resolve);
      });
    });
  }

  /**
   * Get activity logs
   */
  async getActivityLogs() {
    return new Promise((resolve) => {
      browser.storage.sync.get('activityLogs', (result) => {
        resolve(result.activityLogs || []);
      });
    });
  }

  /**
   * Clear activity logs
   */
  async clearActivityLogs() {
    return new Promise((resolve) => {
      browser.storage.sync.remove('activityLogs', resolve);
    });
  }
}

// Export for use in other scripts
if (typeof module !== 'undefined' && module.exports) {
  module.exports = StorageManager;
}
