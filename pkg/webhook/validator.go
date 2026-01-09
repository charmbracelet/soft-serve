package webhook

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
	"slices"
	"strings"
)

var (
	// ErrInvalidScheme is returned when the webhook URL scheme is not http or https.
	ErrInvalidScheme = errors.New("webhook URL must use http or https scheme")
	// ErrPrivateIP is returned when the webhook URL resolves to a private IP address.
	ErrPrivateIP = errors.New("webhook URL cannot resolve to private or internal IP addresses")
	// ErrInvalidURL is returned when the webhook URL is invalid.
	ErrInvalidURL = errors.New("invalid webhook URL")
)

// ValidateWebhookURL validates that a webhook URL is safe to use.
// It checks:
// - URL is properly formatted
// - Scheme is http or https
// - Hostname does not resolve to private/internal IP addresses
// - Hostname is not localhost or similar.
func ValidateWebhookURL(rawURL string) error {
	if rawURL == "" {
		return ErrInvalidURL
	}

	// Parse the URL
	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidURL, err)
	}

	// Check scheme
	if u.Scheme != "http" && u.Scheme != "https" {
		return ErrInvalidScheme
	}

	// Extract hostname (without port)
	hostname := u.Hostname()
	if hostname == "" {
		return fmt.Errorf("%w: missing hostname", ErrInvalidURL)
	}

	// Check for localhost variations
	if isLocalhost(hostname) {
		return ErrPrivateIP
	}

	// If it's an IP address, validate it directly
	if ip := net.ParseIP(hostname); ip != nil {
		if isPrivateOrInternalIP(ip) {
			return ErrPrivateIP
		}
		return nil
	}

	// Resolve hostname to IP addresses
	ips, err := net.DefaultResolver.LookupIPAddr(context.Background(), hostname)
	if err != nil {
		return fmt.Errorf("%w: cannot resolve hostname: %v", ErrInvalidURL, err)
	}

	// Check all resolved IPs
	if slices.ContainsFunc(ips, isPrivateOrInternalIPAddr) {
		return ErrPrivateIP
	}

	return nil
}

// isLocalhost checks if the hostname is localhost or similar.
func isLocalhost(hostname string) bool {
	hostname = strings.ToLower(hostname)
	return hostname == "localhost" ||
		hostname == "localhost.localdomain" ||
		strings.HasSuffix(hostname, ".localhost")
}

// isPrivateOrInternalIPAddr is a helper function that users net.IPAddr instead of net.IP.
func isPrivateOrInternalIPAddr(ipAddr net.IPAddr) bool {
	return isPrivateOrInternalIP(ipAddr.IP)
}

// isPrivateOrInternalIP checks if an IP address is private, internal, or reserved.
func isPrivateOrInternalIP(ip net.IP) bool {
	// Loopback addresses (127.0.0.0/8, ::1)
	if ip.IsLoopback() {
		return true
	}

	// Link-local addresses (169.254.0.0/16, fe80::/10)
	// This blocks AWS/GCP/Azure metadata services
	if ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return true
	}

	// Private addresses (10.0.0.0/8, 172.16.0.0/12, 192.168.0.0/16, fc00::/7)
	if ip.IsPrivate() {
		return true
	}

	// Unspecified addresses (0.0.0.0, ::)
	if ip.IsUnspecified() {
		return true
	}

	// Multicast addresses
	if ip.IsMulticast() {
		return true
	}

	// Additional checks for IPv4
	if ip4 := ip.To4(); ip4 != nil {
		// 0.0.0.0/8 (current network)
		if ip4[0] == 0 {
			return true
		}
		// 100.64.0.0/10 (Shared Address Space)
		if ip4[0] == 100 && ip4[1] >= 64 && ip4[1] <= 127 {
			return true
		}
		// 192.0.0.0/24 (IETF Protocol Assignments)
		if ip4[0] == 192 && ip4[1] == 0 && ip4[2] == 0 {
			return true
		}
		// 192.0.2.0/24 (TEST-NET-1)
		if ip4[0] == 192 && ip4[1] == 0 && ip4[2] == 2 {
			return true
		}
		// 198.18.0.0/15 (benchmarking)
		if ip4[0] == 198 && (ip4[1] == 18 || ip4[1] == 19) {
			return true
		}
		// 198.51.100.0/24 (TEST-NET-2)
		if ip4[0] == 198 && ip4[1] == 51 && ip4[2] == 100 {
			return true
		}
		// 203.0.113.0/24 (TEST-NET-3)
		if ip4[0] == 203 && ip4[1] == 0 && ip4[2] == 113 {
			return true
		}
		// 224.0.0.0/4 (Multicast - already handled by IsMulticast)
		// 240.0.0.0/4 (Reserved for future use)
		if ip4[0] >= 240 {
			return true
		}
		// 255.255.255.255/32 (Broadcast)
		if ip4[0] == 255 && ip4[1] == 255 && ip4[2] == 255 && ip4[3] == 255 {
			return true
		}
	}

	return false
}

// ValidateIPBeforeDial validates an IP address before establishing a connection.
// This is used to prevent DNS rebinding attacks.
func ValidateIPBeforeDial(ip net.IP) error {
	if isPrivateOrInternalIP(ip) {
		return ErrPrivateIP
	}
	return nil
}
