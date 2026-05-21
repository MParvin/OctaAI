package browser

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// Server manages WebSocket connections from browser addons
type Server struct {
	addr      string
	token     string
	clients   map[string]*Client
	clientsMu sync.RWMutex
	upgrader  websocket.Upgrader
	mux       *http.ServeMux
	server    *http.Server
}

// NewServer creates a new WebSocket server for browser connections
func NewServer(addr string, token string) *Server {
	s := &Server{
		addr:    addr,
		token:   token,
		clients: make(map[string]*Client),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				// Allow connections from browser extensions
				return true
			},
		},
	}

	s.mux = http.NewServeMux()
	s.mux.HandleFunc("/ws", s.handleWebSocket)
	s.mux.HandleFunc("/health", s.handleHealth)

	s.server = &http.Server{
		Addr:    addr,
		Handler: s.mux,
	}

	return s
}

// Start starts the WebSocket server
func (s *Server) Start() error {
	log.Printf("Starting browser WebSocket server on %s", s.addr)
	return s.server.ListenAndServe()
}

// Stop gracefully shuts down the server
func (s *Server) Stop(ctx context.Context) error {
	log.Println("Stopping browser WebSocket server...")

	// Close all client connections
	s.clientsMu.Lock()
	for _, client := range s.clients {
		client.Close()
	}
	s.clientsMu.Unlock()

	return s.server.Shutdown(ctx)
}

// handleWebSocket handles WebSocket connection requests
func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	// Check authentication token
	authToken := r.URL.Query().Get("token")
	if s.token != "" && authToken != s.token {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		log.Printf("Rejected connection: invalid token")
		return
	}

	// Upgrade connection to WebSocket
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Failed to upgrade connection: %v", err)
		return
	}

	// Create new client
	clientID := uuid.New().String()
	client := NewClient(clientID, conn)

	s.clientsMu.Lock()
	s.clients[clientID] = client
	s.clientsMu.Unlock()

	log.Printf("Browser connected: %s (total: %d)", clientID, len(s.clients))

	// Handle client connection
	go s.handleClient(client)
}

// handleClient manages a client connection
func (s *Server) handleClient(client *Client) {
	defer func() {
		client.Close()
		s.clientsMu.Lock()
		delete(s.clients, client.ID)
		s.clientsMu.Unlock()
		log.Printf("Browser disconnected: %s (remaining: %d)", client.ID, len(s.clients))
	}()

	// Start ping routine
	go client.pingRoutine()

	// Read messages from browser
	for {
		var response BrowserResponse
		err := client.conn.ReadJSON(&response)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error from %s: %v", client.ID, err)
			}
			break
		}

		// Route response to waiting command
		client.mu.RLock()
		respChan, exists := client.pendingCmds[response.ID]
		client.mu.RUnlock()

		if exists {
			select {
			case respChan <- &response:
			case <-time.After(5 * time.Second):
				log.Printf("Timeout sending response for command %s", response.ID)
			}
		} else {
			log.Printf("Received response for unknown command: %s", response.ID)
		}
	}
}

// handleHealth handles health check requests
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	s.clientsMu.RLock()
	clientCount := len(s.clients)
	s.clientsMu.RUnlock()

	status := map[string]interface{}{
		"status":             "ok",
		"connected_browsers": clientCount,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// SendCommand sends a command to a specific browser client
func (s *Server) SendCommand(clientID string, cmd *BrowserCommand, timeout time.Duration) (*BrowserResponse, error) {
	s.clientsMu.RLock()
	client, exists := s.clients[clientID]
	s.clientsMu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("browser client not found: %s", clientID)
	}

	return client.SendCommand(cmd, timeout)
}

// SendCommandToAny sends a command to any connected browser (first available)
func (s *Server) SendCommandToAny(cmd *BrowserCommand, timeout time.Duration) (*BrowserResponse, error) {
	s.clientsMu.RLock()
	defer s.clientsMu.RUnlock()

	if len(s.clients) == 0 {
		return nil, fmt.Errorf("no browser connected")
	}

	// Get first available client
	for _, client := range s.clients {
		return client.SendCommand(cmd, timeout)
	}

	return nil, fmt.Errorf("no available browser")
}

// GetConnectedBrowsers returns list of connected browser IDs
func (s *Server) GetConnectedBrowsers() []string {
	s.clientsMu.RLock()
	defer s.clientsMu.RUnlock()

	ids := make([]string, 0, len(s.clients))
	for id := range s.clients {
		ids = append(ids, id)
	}
	return ids
}

// HasConnectedBrowser returns true if at least one browser is connected
func (s *Server) HasConnectedBrowser() bool {
	s.clientsMu.RLock()
	defer s.clientsMu.RUnlock()
	return len(s.clients) > 0
}
