package health

import (
	"context"
	"encoding/json"
	"net/http"
	"sync/atomic"
	"time"
)

// StatusFunc reports extra readiness fields (storage, browsers, etc.).
type StatusFunc func() map[string]interface{}

// Server exposes /healthz and /readyz for process supervision.
type Server struct {
	mux        *http.ServeMux
	ready      atomic.Bool
	statusFn   StatusFunc
	httpServer *http.Server
}

// New creates a health server. Call SetReady(true) after startup completes.
func New(addr string, statusFn StatusFunc) *Server {
	s := &Server{
		mux:      http.NewServeMux(),
		statusFn: statusFn,
	}
	s.mux.HandleFunc("/healthz", s.handleLive)
	s.mux.HandleFunc("/readyz", s.handleReady)
	s.mux.HandleFunc("/health", s.handleLive)
	s.httpServer = &http.Server{
		Addr:              addr,
		Handler:           s.mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	return s
}

// SetReady marks the daemon as ready or not.
func (s *Server) SetReady(ready bool) {
	s.ready.Store(ready)
}

// Start listens until the server is closed.
func (s *Server) Start() error {
	return s.httpServer.ListenAndServe()
}

// Shutdown stops the health server.
func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}

func (s *Server) handleLive(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]interface{}{"status": "ok"})
}

func (s *Server) handleReady(w http.ResponseWriter, r *http.Request) {
	payload := map[string]interface{}{"status": "ready", "ready": s.ready.Load()}
	if s.statusFn != nil {
		for k, v := range s.statusFn() {
			payload[k] = v
		}
	}
	code := http.StatusOK
	if !s.ready.Load() {
		code = http.StatusServiceUnavailable
		payload["status"] = "not_ready"
	}
	writeJSON(w, code, payload)
}

func writeJSON(w http.ResponseWriter, code int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}
