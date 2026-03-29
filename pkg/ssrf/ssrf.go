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
// and internal networks. Hostnames are resolved and the validated IP is
// used directly in the dial call to prevent DNS rebinding (TOCTOU between
// validation and connection). Redirects are disabled to match the webhook
// client convention and prevent redirect-based SSRF.
func NewSecureClient() *http.Client {
	return &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				host, port, err := net.SplitHostPort(addr)
				if err != nil {
					return nil, err //nolint:wrapcheck
				}

				ip := net.ParseIP(host)
				if ip == nil {
					// Use the context-aware resolver so the lookup respects
					// cancellation and deadlines from the caller.
					addrs, err := net.DefaultResolver.LookupIPAddr(ctx, host)
					if err != nil {
						return nil, fmt.Errorf("DNS resolution failed for host %s: %v", host, err)
					}
					if len(addrs) == 0 {
						return nil, fmt.Errorf("no IP addresses found for host: %s", host)
					}
					// Reject if ANY resolved IP is private/internal.
					// Select the first public IP for dialing so that DNS
					// round-robin cannot swap in a private address on retry.
					var selectedIP net.IP
					for _, addr := range addrs {
						if isPrivateOrInternal(addr.IP) {
							return nil, fmt.Errorf("%w", ErrPrivateIP)
						}
						if selectedIP == nil {
							selectedIP = addr.IP
						}
					}
					ip = selectedIP
				}
				if isPrivateOrInternal(ip) {
					return nil, fmt.Errorf("%w", ErrPrivateIP)
				}

				dialer := &net.Dialer{
					Timeout:   10 * time.Second,
					KeepAlive: 30 * time.Second,
				}
				// Dial using the validated IP to prevent DNS rebinding.
				// Without this, the dialer resolves the hostname again
				// independently, and the second resolution could return
				// a different (private) IP.
				// Note: creating a new dialer per call is wasteful but
				// ensures each connection has a fresh resolver state
				// (safe against cache poisoning). For high-throughput
				// webhook delivery, consider reusing a single dialer outside
				// this closure if performance becomes a bottleneck.
				return dialer.DialContext(ctx, network, net.JoinHostPort(ip.String(), port))
			},
			MaxIdleConns:          100,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		},
		// Refuse all HTTP redirects. For webhooks this prevents SSRF via a
		// redirect to an internal host. For LFS this is safe because the LFS
		// batch API returns explicit download/upload href values; the LFS client
		// calls those URLs directly and does not rely on server-issued redirects.
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

	// ip.IsPrivate() covers IPv6 ULA (fc00::/7) and IPv4 private ranges;
	// no separate fc00::/7 check is needed here.
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
// and all resolved IPs are public. The provided context is used for the DNS
// lookup; a 5-second sub-deadline is applied if the context has no deadline.
func ValidateURL(ctx context.Context, rawURL string) error {
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

	resolveCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	ips, err := net.DefaultResolver.LookupIPAddr(resolveCtx, hostname)
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

// ValidateHost resolves host and checks that none of the resolved IPs are
// private or internal. Use this for non-HTTP schemes (e.g. ssh://) where
// ValidateURL cannot be used. The provided context is used for the DNS lookup.
//
// A 5-second sub-deadline is applied for the DNS lookup regardless of any
// deadline already present on ctx. If ctx has a tighter deadline, that takes
// precedence. If ctx has a longer (or no) deadline, the 5-second guard is the
// effective timeout for the DNS resolution step.
func ValidateHost(ctx context.Context, host string) error {
	if host == "" {
		return fmt.Errorf("%w: missing hostname", ErrInvalidURL)
	}

	if isLocalhost(host) {
		return ErrPrivateIP
	}

	if ip := net.ParseIP(host); ip != nil {
		if isPrivateOrInternal(ip) {
			return ErrPrivateIP
		}
		return nil
	}

	resolveCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	ips, err := net.DefaultResolver.LookupIPAddr(resolveCtx, host)
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

// isLocalhost checks if the hostname is localhost or similar.
func isLocalhost(hostname string) bool {
	hostname = strings.ToLower(hostname)
	return hostname == "localhost" ||
		hostname == "localhost.localdomain" ||
		strings.HasSuffix(hostname, ".localhost")
}
