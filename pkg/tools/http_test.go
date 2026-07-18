package tools

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestValidatePublicHTTPURL(t *testing.T) {
	cases := []struct {
		url     string
		wantErr bool
	}{
		{"http://example.com/a", false},
		{"https://example.com", false},
		{"http://127.0.0.1/", true},
		{"http://localhost/admin", true},
		{"http://192.168.1.1/", true},
		{"http://10.0.0.5/x", true},
		{"file:///etc/passwd", true},
		{"ftp://example.com", true},
		{"not-a-url", true},
	}
	for _, tc := range cases {
		err := validatePublicHTTPURL(tc.url)
		if tc.wantErr && err == nil {
			t.Fatalf("%s: expected error", tc.url)
		}
		if !tc.wantErr && err != nil {
			t.Fatalf("%s: unexpected error: %v", tc.url, err)
		}
	}
}

func TestHTTPToolBlocksPrivateLiteralIP(t *testing.T) {
	tool := NewHTTPTool()
	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"method": "GET",
		"url":    "http://127.0.0.1:9/",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Success {
		t.Fatal("expected failure for loopback URL")
	}
}

func TestHTTPToolGETPublicServer(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ok"))
	}))
	defer srv.Close()

	// httptest binds to 127.0.0.1 — must be rejected by SSRF dialer.
	tool := NewHTTPTool()
	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"method": "GET",
		"url":    srv.URL,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Success {
		t.Fatal("httptest loopback URL must be blocked by SSRF controls")
	}
}
