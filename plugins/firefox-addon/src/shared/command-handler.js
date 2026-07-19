/**
 * CommandHandler - Routes and executes commands from the daemon
 */
class CommandHandler {
  constructor() {
    this.handlers = new Map();
    this.registerDefaultHandlers();
  }

  /**
   * Register a command handler
   */
  register(commandType, handler) {
    this.handlers.set(commandType, handler);
  }

  /**
   * Execute a command
   */
  async execute(command) {
    const handler = this.handlers.get(command.type);
    if (!handler) {
      throw new Error(`Unknown command type: ${command.type}`);
    }

    try {
      const result = await handler(command.params);
      return {
        status: 'success',
        result,
        timestamp: Date.now()
      };
    } catch (error) {
      return {
        status: 'error',
        error: error.message,
        timestamp: Date.now()
      };
    }
  }

  /**
   * Register default command handlers
   */
  registerDefaultHandlers() {
    this.register('ping', () => ({ pong: true }));
    this.register('get_status', () => ({ status: 'ready' }));
  }
}

// Export for use in other scripts
if (typeof module !== 'undefined' && module.exports) {
  module.exports = CommandHandler;
}
