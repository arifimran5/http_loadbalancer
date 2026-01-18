package balancer

import (
	"net"
	"net/http"
	"strings"
)

// trustedProxyCIDRs defines private/internal networks that are trusted proxies.
// Requests from these ranges will have their X-Forwarded-For headers parsed.
var trustedProxyCIDRs = []string{
	"127.0.0.0/8",    // Loopback
	"10.0.0.0/8",     // Private Class A
	"172.16.0.0/12",  // Private Class B
	"192.168.0.0/16", // Private Class C
	"::1/128",        // IPv6 loopback
	"fc00::/7",       // IPv6 private
}

var trustedNets []*net.IPNet

func init() {
	for _, cidr := range trustedProxyCIDRs {
		_, network, err := net.ParseCIDR(cidr)
		if err == nil {
			trustedNets = append(trustedNets, network)
		}
	}
}

// isTrustedProxy checks if the given IP is from a trusted proxy network.
func isTrustedProxy(ip net.IP) bool {
	for _, network := range trustedNets {
		if network.Contains(ip) {
			return true
		}
	}
	return false
}

// ExtractClientIP extracts the real client IP from the request.
// It safely parses X-Forwarded-For when the request comes from a trusted proxy.
func ExtractClientIP(req *http.Request) string {
	// Get the immediate connection IP
	remoteIP, _, err := net.SplitHostPort(req.RemoteAddr)
	if err != nil {
		// RemoteAddr might not have a port
		remoteIP = req.RemoteAddr
	}

	parsedRemoteIP := net.ParseIP(remoteIP)
	if parsedRemoteIP == nil {
		return remoteIP
	}

	// Only trust X-Forwarded-For if the immediate connection is from a trusted proxy
	if !isTrustedProxy(parsedRemoteIP) {
		return remoteIP
	}

	// Parse X-Forwarded-For header (format: "client, proxy1, proxy2")
	xff := req.Header.Get("X-Forwarded-For")
	if xff == "" {
		// Try standard Forwarded header
		forwarded := req.Header.Get("Forwarded")
		if forwarded != "" {
			return parseForwardedHeader(forwarded)
		}
		return remoteIP
	}

	// Get the leftmost (original client) IP that isn't a trusted proxy
	ips := strings.Split(xff, ",")
	for _, ip := range ips {
		ip = strings.TrimSpace(ip)
		parsedIP := net.ParseIP(ip)
		if parsedIP != nil && !isTrustedProxy(parsedIP) {
			return ip
		}
	}

	return remoteIP
}

// parseForwardedHeader extracts the client IP from the standard Forwarded header.
// Format: "for=192.0.2.60;proto=http;by=203.0.113.43"
func parseForwardedHeader(header string) string {
	parts := strings.Split(header, ";")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(strings.ToLower(part), "for=") {
			value := part[4:]
			// Remove quotes if present
			value = strings.Trim(value, `"`)
			// Handle IPv6 with brackets
			if strings.HasPrefix(value, "[") {
				if idx := strings.Index(value, "]"); idx != -1 {
					return value[1:idx]
				}
			}
			// Handle IPv4 or IPv6 without brackets (may have port)
			if host, _, err := net.SplitHostPort(value); err == nil {
				return host
			}
			return value
		}
	}
	return ""
}
