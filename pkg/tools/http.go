package tools

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/mparvin/octaai/pkg/permission"
)

// HTTPTool provides HTTP operations with SSRF-safe dialing.
type HTTPTool struct {
	client *http.Client
}

// NewHTTPTool creates a new HTTP tool with a dialer that blocks private/link-local IPs.
func NewHTTPTool() *HTTPTool {
	dialer := &net.Dialer{Timeout: 10 * time.Second}
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			host, port, err := net.SplitHostPort(addr)
			if err != nil {
				return nil, err
			}
			ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
			if err != nil {
				return nil, err
			}
			var lastErr error
			for _, ip := range ips {
				if permission.IsBlockedIP(ip.IP) {
					lastErr = fmt.Errorf("blocked private or link-local address: %s", ip.IP)
					continue
				}
				conn, err := dialer.DialContext(ctx, network, net.JoinHostPort(ip.IP.String(), port))
				if err == nil {
					return conn, nil
				}
				lastErr = err
			}
			if lastErr == nil {
				lastErr = fmt.Errorf("no safe addresses for host %q", host)
			}
			return nil, lastErr
		},
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          10,
		IdleConnTimeout:       30 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	return &HTTPTool{
		client: &http.Client{
			Timeout:   30 * time.Second,
			Transport: transport,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= 5 {
					return fmt.Errorf("stopped after 5 redirects")
				}
				if err := validatePublicHTTPURL(req.URL.String()); err != nil {
					return err
				}
				return nil
			},
		},
	}
}

func validatePublicHTTPURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return fmt.Errorf("invalid URL: %q", raw)
	}
	scheme := strings.ToLower(u.Scheme)
	if scheme != "http" && scheme != "https" {
		return fmt.Errorf("unsupported URL scheme: %s", scheme)
	}
	host := strings.ToLower(u.Hostname())
	if host == "localhost" || strings.HasSuffix(host, ".localhost") {
		return fmt.Errorf("localhost URLs are not allowed")
	}
	if ip := net.ParseIP(host); ip != nil && permission.IsBlockedIP(ip) {
		return fmt.Errorf("private or link-local URLs are not allowed")
	}
	return nil
}

// Name implements Tool.Name
func (t *HTTPTool) Name() string {
	return "http"
}

// Schema implements Tool.Schema
func (t *HTTPTool) Schema() ToolSchema {
	return ToolSchema{
		Name:        "http",
		Description: "Make HTTP requests to web services",
		Parameters: map[string]ParamSchema{
			"method": {
				Type:        "string",
				Description: "HTTP method",
				Required:    true,
				Enum:        []string{"GET", "POST", "PUT", "DELETE"},
			},
			"url": {
				Type:        "string",
				Description: "Request URL",
				Required:    true,
			},
			"headers": {
				Type:        "object",
				Description: "HTTP headers as key-value pairs",
				Required:    false,
			},
			"body": {
				Type:        "string",
				Description: "Request body",
				Required:    false,
			},
		},
	}
}

// Execute implements Tool.Execute
func (t *HTTPTool) Execute(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
	method, ok := args["method"].(string)
	if !ok {
		return nil, fmt.Errorf("method is required")
	}

	rawURL, ok := args["url"].(string)
	if !ok {
		return nil, fmt.Errorf("url is required")
	}
	if err := validatePublicHTTPURL(rawURL); err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}

	var body io.Reader
	if bodyStr, ok := args["body"].(string); ok && bodyStr != "" {
		body = strings.NewReader(bodyStr)
	}

	req, err := http.NewRequestWithContext(ctx, method, rawURL, body)
	if err != nil {
		return &ToolResult{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	if headers, ok := args["headers"].(map[string]interface{}); ok {
		for key, value := range headers {
			if strValue, ok := value.(string); ok {
				req.Header.Set(key, strValue)
			}
		}
	}

	resp, err := t.client.Do(req)
	if err != nil {
		return &ToolResult{
			Success: false,
			Error:   err.Error(),
		}, nil
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return &ToolResult{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	success := resp.StatusCode >= 200 && resp.StatusCode < 300

	return &ToolResult{
		Success: success,
		Output:  string(respBody),
		Data: map[string]interface{}{
			"status_code": resp.StatusCode,
			"status":      resp.Status,
			"headers":     resp.Header,
		},
	}, nil
}
