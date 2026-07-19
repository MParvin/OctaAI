/**
 * WebSocketManager - Handles connection to OctaAI daemon
 * Implements automatic reconnection, token authentication, and heartbeat
 */

/** Default daemon endpoint (path /ws, port 8765). */
const DEFAULT_WS_URL = 'ws://localhost:8765/ws';

/**
 * Normalize a server URL so it always targets the daemon /ws path.
 * Accepts bare host:port, trailing slash, or legacy root URLs.
 */
function normalizeWebSocketURL(url) {
  let u = (url || DEFAULT_WS_URL).trim();
  if (!u) {
    return DEFAULT_WS_URL;
  }
  // Strip legacy query token from stored URLs
  const q = u.indexOf('?');
  if (q >= 0) {
    u = u.slice(0, q);
  }
  u = u.replace(/\/+$/, '');
  if (!u.endsWith('/ws')) {
    u = `${u}/ws`;
  }
  return u;
}

class WebSocketManager {
  constructor(config = {}) {
    this.url = normalizeWebSocketURL(config.url || DEFAULT_WS_URL);
    this.token = config.token || '';
    this.autoReconnect = config.autoReconnect !== false;
    this.reconnectDelay = config.reconnectDelay || 3000;
    this.maxReconnectDelay = config.maxReconnectDelay || 30000;
    this.heartbeatInterval = config.heartbeatInterval || 30000;
    this.commandTimeout = config.commandTimeout || 30000;

    this.ws = null;
    this.isConnected = false;
    this.isConnecting = false;
    this.currentReconnectDelay = this.reconnectDelay;
    this.pendingCommands = new Map();
    this.messageHandlers = [];
    this.connectionStateHandlers = [];
    this.heartbeatTimer = null;
  }

  /**
   * Connect to the WebSocket server
   */
  connect() {
    if (this.isConnecting || this.isConnected) {
      return Promise.resolve();
    }

    this.isConnecting = true;
    return new Promise((resolve, reject) => {
      try {
        const wsUrl = normalizeWebSocketURL(this.url);
        // Prefer Sec-WebSocket-Protocol so the token is not in query logs/proxies.
        // Format matches pkg/browser/server.go: octaai.<token>
        const protocols = this.token ? [`octaai.${this.token}`] : undefined;
        this.ws = protocols ? new WebSocket(wsUrl, protocols) : new WebSocket(wsUrl);

        this.ws.onopen = () => {
          this.isConnected = true;
          this.isConnecting = false;
          this.currentReconnectDelay = this.reconnectDelay;
          this.startHeartbeat();
          this.notifyConnectionStateChange(true);
          resolve();
        };

        this.ws.onmessage = (event) => {
          try {
            const message = JSON.parse(event.data);
            this.handleMessage(message);
          } catch (e) {
            console.error('Failed to parse WebSocket message:', e);
          }
        };

        this.ws.onerror = (error) => {
          console.error('WebSocket error:', error);
          this.notifyConnectionStateChange(false, error.message);
          if (!this.isConnected && this.isConnecting) {
            reject(error);
          }
        };

        this.ws.onclose = () => {
          this.isConnected = false;
          this.isConnecting = false;
          this.stopHeartbeat();
          this.notifyConnectionStateChange(false);

          if (this.autoReconnect) {
            setTimeout(() => this.connect().catch(console.error), this.currentReconnectDelay);
            this.currentReconnectDelay = Math.min(
              this.currentReconnectDelay * 1.5,
              this.maxReconnectDelay
            );
          }
        };
      } catch (error) {
        this.isConnecting = false;
        reject(error);
      }
    });
  }

  /**
   * Send a command to the daemon
   */
  sendCommand(command) {
    if (!this.isConnected) {
      return Promise.reject(new Error('WebSocket not connected'));
    }

    return new Promise((resolve, reject) => {
      const id = this.generateId();
      const timeout = setTimeout(() => {
        this.pendingCommands.delete(id);
        reject(new Error(`Command timeout: ${command.type}`));
      }, this.commandTimeout);

      this.pendingCommands.set(id, {
        resolve,
        reject,
        timeout,
        command
      });

      try {
        this.ws.send(JSON.stringify({
          id,
          ...command
        }));
      } catch (error) {
        this.pendingCommands.delete(id);
        clearTimeout(timeout);
        reject(error);
      }
    });
  }

  /**
   * Handle incoming messages from the daemon
   */
  handleMessage(message) {
    if (message.id && this.pendingCommands.has(message.id)) {
      const pending = this.pendingCommands.get(message.id);
      this.pendingCommands.delete(message.id);
      clearTimeout(pending.timeout);

      if (message.status === 'success') {
        pending.resolve(message);
      } else {
        pending.reject(new Error(message.error || 'Command failed'));
      }
    } else {
      // Broadcast message to all registered handlers
      this.messageHandlers.forEach(handler => {
        try {
          handler(message);
        } catch (e) {
          console.error('Error in message handler:', e);
        }
      });
    }
  }

  /**
   * Register a handler for incoming messages
   */
  onMessage(handler) {
    this.messageHandlers.push(handler);
    return () => {
      const index = this.messageHandlers.indexOf(handler);
      if (index > -1) {
        this.messageHandlers.splice(index, 1);
      }
    };
  }

  /**
   * Register a handler for connection state changes
   */
  onConnectionStateChange(handler) {
    this.connectionStateHandlers.push(handler);
    return () => {
      const index = this.connectionStateHandlers.indexOf(handler);
      if (index > -1) {
        this.connectionStateHandlers.splice(index, 1);
      }
    };
  }

  /**
   * Notify all connection state handlers
   */
  notifyConnectionStateChange(isConnected, error = null) {
    this.connectionStateHandlers.forEach(handler => {
      try {
        handler({
          isConnected,
          error,
          timestamp: Date.now()
        });
      } catch (e) {
        console.error('Error in connection state handler:', e);
      }
    });
  }

  /**
   * Start heartbeat mechanism
   */
  startHeartbeat() {
    this.stopHeartbeat();
    this.heartbeatTimer = setInterval(() => {
      if (this.isConnected) {
        try {
          this.ws.send(JSON.stringify({ type: 'ping' }));
        } catch (e) {
          console.error('Failed to send heartbeat:', e);
        }
      }
    }, this.heartbeatInterval);
  }

  /**
   * Stop heartbeat mechanism
   */
  stopHeartbeat() {
    if (this.heartbeatTimer) {
      clearInterval(this.heartbeatTimer);
      this.heartbeatTimer = null;
    }
  }

  /**
   * Disconnect from the server
   */
  disconnect() {
    this.autoReconnect = false;
    this.stopHeartbeat();
    if (this.ws) {
      this.ws.close();
      this.ws = null;
    }
    this.isConnected = false;
    this.isConnecting = false;
  }

  /**
   * Generate a unique command ID
   */
  generateId() {
    return `${Date.now()}-${Math.random().toString(36).substr(2, 9)}`;
  }

  /**
   * Get connection status
   */
  getStatus() {
    return {
      isConnected: this.isConnected,
      isConnecting: this.isConnecting,
      pendingCommands: this.pendingCommands.size,
      url: this.url
    };
  }
}

// Export for use in other scripts
if (typeof module !== 'undefined' && module.exports) {
  module.exports = WebSocketManager;
}
