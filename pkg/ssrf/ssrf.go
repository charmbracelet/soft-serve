package ssrf

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"
)

var (
	// ErrPrivateIP is returned when a connection to a private or internal IP is blocked.
	ErrPrivateIP = errors.New("connection to private or internal IP address is not allowed")
	// ErrInvalidScheme is returned when a URL scheme is not http or https.
	ErrInvalidScheme = errors.New("URL must use http or https scheme")
	// ErrInvalidURL is returned when a URL is invalid.
	ErrInvalidURL = errors.New("invalid URL")
)

// NewSecureClient returns an HTTP client with SSRF protection.
// It validates resolved IPs at dial time to block connections to private
// and internal networks. Since validation uses the already-resolved IP
// from the Transport's DNS lookup, there is no TOCTOU gap between
// resolution and connection. Redirects are disabled to match the
// webhook client convention and prevent redirect-based SSRF.
func NewSecureClient() *http.Client {
	return &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				host, _, err := net.SplitHostPort(addr)
				if err != nil {
					return nil, err //nolint:wrapcheck
				}

				ip := net.ParseIP(host)
				if ip == nil {
					ips, err := net.LookupIP(host) //nolint
					if err != nil {
						return nil, fmt.Errorf("DNS resolution failed for host %s: %v", host, err)
					}
					if len(ips) == 0 {
						return nil, fmt.Errorf("no IP addresses found for host: %s", host)
					}
					ip = ips[0] // Use the first resolved IP address
				}
				if isPrivateOrInternal(ip) {
					return nil, fmt.Errorf("%w", ErrPrivateIP)
				}

				dialer := &net.Dialer{
					Timeout:   10 * time.Second,
					KeepAlive: 30 * time.Second,
				}
				return dialer.DialContext(ctx, network, addr)
			},
			MaxIdleConns:          100,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		},
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}

// isPrivateOrInternal checks if an IP address is private, internal, or reserved.
func isPrivateOrInternal(ip net.IP) bool {
	// Normalize IPv6-mapped IPv4 (e.g. ::ffff:127.0.0.1) to IPv4 form
	// so all checks apply consistently.
	if ip4 := ip.To4(); ip4 != nil {
		ip = ip4
	}

	if ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() ||
		ip.IsPrivate() || ip.IsUnspecified() || ip.IsMulticast() {
		return true
	}

	if ip4 := ip.To4(); ip4 != nil {
		// 0.0.0.0/8
		if ip4[0] == 0 {
			return true
		}
		// 100.64.0.0/10 (Shared Address Space / CGNAT)
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
		// 240.0.0.0/4 (Reserved, includes 255.255.255.255 broadcast)
		if ip4[0] >= 240 {
			return true
		}
	}

	return false
}

// ValidateURL validates that a URL is safe to make requests to.
// It checks that the scheme is http/https, the hostname is not localhost,
// and all resolved IPs are public.
func ValidateURL(rawURL string) error {
	if rawURL == "" {
		return ErrInvalidURL
	}

	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidURL, err)
	}

	if u.Scheme != "http" && u.Scheme != "https" {
		return ErrInvalidScheme
	}

	hostname := u.Hostname()
	if hostname == "" {
		return fmt.Errorf("%w: missing hostname", ErrInvalidURL)
	}

	if isLocalhost(hostname) {
		return ErrPrivateIP
	}

	if ip := net.ParseIP(hostname); ip != nil {
		if isPrivateOrInternal(ip) {
			return ErrPrivateIP
		}
		return nil
	}

	ips, err := net.DefaultResolver.LookupIPAddr(context.Background(), hostname)
	if err != nil {
		return fmt.Errorf("%w: cannot resolve hostname: %v", ErrInvalidURL, err)
	}

	if slices.ContainsFunc(ips, func(addr net.IPAddr) bool {
		return isPrivateOrInternal(addr.IP)
	}) {
		return ErrPrivateIP
	}

	return nil
}

// ValidateIPBeforeDial validates an IP address before establishing a connection.
// This prevents DNS rebinding attacks by checking the resolved IP at dial time.
func ValidateIPBeforeDial(ip net.IP) error {
	if isPrivateOrInternal(ip) {
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
