package permission

import (
	"fmt"
	"net"
	"net/url"
	"strings"
)

func isBlockedIP(ip net.IP) bool {
	if ip.IsLoopback() || ip.IsUnspecified() || ip.IsMulticast() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsPrivate() {
		return true
	}
	if ip4 := ip.To4(); ip4 != nil {
		return ip4[0] == 169 && ip4[1] == 254
	}
	return false
}

// CheckURL validates an HTTP(S) URL against SSRF rules.
func (m *Manager) CheckURL(rawURL string) CheckResult {
	u, err := url.Parse(rawURL)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return CheckResult{
			Decision: DecisionDeny,
			Reason:   fmt.Sprintf("invalid URL: %q", rawURL),
		}
	}

	scheme := strings.ToLower(u.Scheme)
	if scheme != "http" && scheme != "https" {
		return CheckResult{
			Decision: DecisionDeny,
			Reason:   fmt.Sprintf("unsupported URL scheme: %s", scheme),
		}
	}

	host := strings.ToLower(u.Hostname())
	if host == "localhost" || strings.HasSuffix(host, ".localhost") {
		return CheckResult{
			Decision: DecisionDeny,
			Reason:   "localhost URLs are not allowed",
		}
	}

	if ip := net.ParseIP(host); ip != nil {
		if isBlockedIP(ip) {
			return CheckResult{Decision: DecisionDeny, Reason: "private or link-local URLs are not allowed"}
		}
	} else {
		ips, err := net.LookupIP(host)
		if err != nil {
			return CheckResult{Decision: DecisionDeny, Reason: fmt.Sprintf("failed to resolve host %q: %v", host, err)}
		}
		for _, resolved := range ips {
			if isBlockedIP(resolved) {
				return CheckResult{Decision: DecisionDeny, Reason: "private or link-local URLs are not allowed"}
			}
		}
	}
	if len(m.cfg.Safety.AllowHTTPHosts) > 0 {
		allowed := false
		for _, pattern := range m.cfg.Safety.AllowHTTPHosts {
			pattern = strings.ToLower(pattern)
			if host == pattern || strings.HasSuffix(host, "."+pattern) {
				allowed = true
				break
			}
		}
		if !allowed {
			return CheckResult{
				Decision: DecisionDeny,
				Reason:   fmt.Sprintf("host %q is not in allow_http_hosts", host),
			}
		}
	}

	return CheckResult{Decision: DecisionAllow}
}

// CheckBrowserDomain validates a browser navigation URL against browser_domains.
func (m *Manager) CheckBrowserDomain(rawURL string) CheckResult {
	if len(m.cfg.Browser.BrowserDomains) == 0 {
		return CheckResult{Decision: DecisionAllow}
	}

	u, err := url.Parse(rawURL)
	if err != nil || u.Host == "" {
		return CheckResult{
			Decision: DecisionDeny,
			Reason:   fmt.Sprintf("invalid browser URL: %q", rawURL),
		}
	}

	host := strings.ToLower(u.Hostname())
	for _, domain := range m.cfg.Browser.BrowserDomains {
		domain = strings.ToLower(domain)
		if host == domain || strings.HasSuffix(host, "."+domain) {
			return CheckResult{Decision: DecisionAllow}
		}
	}

	return CheckResult{
		Decision: DecisionDeny,
		Reason:   fmt.Sprintf("browser domain %q is not allowed", host),
	}
}
