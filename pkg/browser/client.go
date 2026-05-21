package browser

import (
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// Client represents a connected browser instance
type Client struct {
	ID            string
	conn          *websocket.Conn
	connectedAt   time.Time
	lastHeartbeat time.Time
	pendingCmds   map[string]chan *BrowserResponse
	mu            sync.RWMutex
	closed        bool
}

// NewClient creates a new browser client
func NewClient(id string, conn *websocket.Conn) *Client {
	now := time.Now()
	return &Client{
		ID:            id,
		conn:          conn,
		connectedAt:   now,
		lastHeartbeat: now,
		pendingCmds:   make(map[string]chan *BrowserResponse),
	}
}

// SendCommand sends a command to the browser and waits for response
func (c *Client) SendCommand(cmd *BrowserCommand, timeout time.Duration) (*BrowserResponse, error) {
	if c.closed {
		return nil, fmt.Errorf("client connection closed")
	}

	// Generate command ID if not set
	if cmd.ID == "" {
		cmd.ID = uuid.New().String()
	}

	// Create response channel
	respChan := make(chan *BrowserResponse, 1)
	c.mu.Lock()
	c.pendingCmds[cmd.ID] = respChan
	c.mu.Unlock()

	// Cleanup
	defer func() {
		c.mu.Lock()
		delete(c.pendingCmds, cmd.ID)
		c.mu.Unlock()
		close(respChan)
	}()

	// Send command
	err := c.conn.WriteJSON(cmd)
	if err != nil {
		return nil, fmt.Errorf("failed to send command: %w", err)
	}

	// Wait for response with timeout
	select {
	case response := <-respChan:
		return response, nil
	case <-time.After(timeout):
		return nil, fmt.Errorf("command timeout after %v", timeout)
	}
}

// Close closes the client connection
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return nil
	}

	c.closed = true

	// Close all pending command channels
	for _, ch := range c.pendingCmds {
		close(ch)
	}
	c.pendingCmds = make(map[string]chan *BrowserResponse)

	return c.conn.Close()
}

// pingRoutine sends periodic ping messages to keep connection alive
func (c *Client) pingRoutine() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		if c.closed {
			return
		}

		c.mu.Lock()
		err := c.conn.WriteMessage(websocket.PingMessage, nil)
		c.mu.Unlock()

		if err != nil {
			return
		}

		c.lastHeartbeat = time.Now()
	}
}

// IsAlive checks if the client is still connected
func (c *Client) IsAlive() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return !c.closed && time.Since(c.lastHeartbeat) < 2*time.Minute
}

// GetStats returns client statistics
func (c *Client) GetStats() map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return map[string]interface{}{
		"id":               c.ID,
		"connected_at":     c.connectedAt,
		"last_heartbeat":   c.lastHeartbeat,
		"pending_commands": len(c.pendingCmds),
		"alive":            !c.closed && time.Since(c.lastHeartbeat) < 2*time.Minute,
	}
}
