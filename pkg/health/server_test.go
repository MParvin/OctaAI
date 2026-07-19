package health

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthzAlwaysOK(t *testing.T) {
	s := New("127.0.0.1:0", nil)
	rr := httptest.NewRecorder()
	s.handleLive(rr, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("code %d", rr.Code)
	}
}

func TestReadyzReflectsReadyFlag(t *testing.T) {
	s := New("127.0.0.1:0", func() map[string]interface{} {
		return map[string]interface{}{"tools": 3}
	})
	rr := httptest.NewRecorder()
	s.handleReady(rr, httptest.NewRequest(http.MethodGet, "/readyz", nil))
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 before ready, got %d", rr.Code)
	}

	s.SetReady(true)
	rr = httptest.NewRecorder()
	s.handleReady(rr, httptest.NewRequest(http.MethodGet, "/readyz", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 when ready, got %d", rr.Code)
	}
	var body map[string]interface{}
	if err := json.NewDecoder(rr.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body["tools"].(float64) != 3 {
		t.Fatalf("unexpected body: %v", body)
	}
}
