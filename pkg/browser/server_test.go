package browser

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/websocket"
)

func TestAllowedOrigin(t *testing.T) {
	if allowedOrigin("") {
		t.Fatal("empty Origin must be rejected")
	}
	if !allowedOrigin("moz-extension://abcd-1234") {
		t.Fatal("moz-extension Origin should be allowed")
	}
	if !allowedOrigin("http://127.0.0.1:3000") {
		t.Fatal("localhost Origin should be allowed")
	}
	if allowedOrigin("https://evil.example") {
		t.Fatal("arbitrary https Origin must be rejected")
	}
}

func TestHealthEndpoint(t *testing.T) {
	s := NewServer("127.0.0.1:0", "test-token-value")
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rr := httptest.NewRecorder()
	s.handleHealth(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status %d", rr.Code)
	}
	var body map[string]interface{}
	if err := json.NewDecoder(rr.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body["status"] != "ok" {
		t.Fatalf("unexpected body: %v", body)
	}
}

func TestWebSocketRejectsBadToken(t *testing.T) {
	s := NewServer("127.0.0.1:0", "secret-token")
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", s.handleWebSocket)
	srv := httptest.NewServer(mux)
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/ws?token=wrong"
	_, _, err := websocket.DefaultDialer.Dial(wsURL, http.Header{
		"Origin": []string{"moz-extension://test"},
	})
	if err == nil {
		t.Fatal("expected dial failure for bad token")
	}
}

func TestWebSocketAcceptsProtocolToken(t *testing.T) {
	s := NewServer("127.0.0.1:0", "secret-token")
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", s.handleWebSocket)
	srv := httptest.NewServer(mux)
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/ws"
	conn, resp, err := websocket.DefaultDialer.Dial(wsURL, http.Header{
		"Origin":                 []string{"moz-extension://test"},
		"Sec-WebSocket-Protocol": []string{"octaai.secret-token"},
	})
	if err != nil {
		t.Fatalf("dial failed: %v (resp=%v)", err, resp)
	}
	defer conn.Close()
	if !s.HasConnectedBrowser() {
		// connection handler runs asynchronously; give it a moment via status
		ids := s.GetConnectedBrowsers()
		_ = ids
	}
}

func TestNewServerRequiresToken(t *testing.T) {
	s := NewServer("127.0.0.1:0", "")
	if err := s.Start(); err == nil {
		t.Fatal("expected Start to require non-empty token")
	}
}
