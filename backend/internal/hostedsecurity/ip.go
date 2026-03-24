package hostedsecurity

import (
	"net"
	"net/http"
	"strings"
)

// ClientIP returns a best-effort client address from X-Forwarded-For (first hop) or RemoteAddr.
func ClientIP(r *http.Request) string {
	if r == nil {
		return ""
	}
	if xff := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); xff != "" {
		parts := strings.Split(xff, ",")
		if len(parts) > 0 {
			return strings.TrimSpace(parts[0])
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}
	return strings.TrimSpace(r.RemoteAddr)
}

// IPAllowed returns true if allowlist is empty (no restriction) or ip matches any CIDR or single IP.
func IPAllowed(ip string, cidrs []string) bool {
	if len(cidrs) == 0 {
		return true
	}
	parsed := net.ParseIP(strings.TrimSpace(ip))
	if parsed == nil {
		return false
	}
	for _, c := range cidrs {
		c = strings.TrimSpace(c)
		if c == "" {
			continue
		}
		if strings.Contains(c, "/") {
			_, network, err := net.ParseCIDR(c)
			if err != nil {
				continue
			}
			if network.Contains(parsed) {
				return true
			}
			continue
		}
		if host := net.ParseIP(c); host != nil && host.Equal(parsed) {
			return true
		}
	}
	return false
}
