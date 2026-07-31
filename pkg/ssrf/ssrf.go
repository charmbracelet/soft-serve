package ssrf

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/netip"
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
				// Dial using the validated IP to prevent DNS rebinding.
				// Without this, the dialer resolves the hostname again
				// independently, and the second resolution could return
				// a different (private) IP.
				return dialer.DialContext(ctx, network, net.JoinHostPort(ip.String(), port))
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

// blockedPrefixes lists every network an outbound request must not reach.
//
// Prefixes are used rather than byte comparisons so the ranges stay readable
// and auditable against the RFCs that define them. This is also how the
// standard library models the same address space internally.
//
// The IPv6 transition ranges need explanation. Each embeds an arbitrary IPv4
// address inside an IPv6 one, so an address that looks public to the IPv6
// checks can name a loopback, RFC1918, or cloud metadata host once a relay
// decodes it. They are blocked in full rather than decoded and re-checked,
// because nothing here has a legitimate reason to reach a host through one of
// these encodings. Blocking the range removes the whole class of bypass
// instead of depending on getting each decoding exactly right.
var blockedPrefixes = []netip.Prefix{
	// IPv4.
	netip.MustParsePrefix("0.0.0.0/8"),       // "this network"
	netip.MustParsePrefix("10.0.0.0/8"),      // RFC1918 private
	netip.MustParsePrefix("100.64.0.0/10"),   // RFC6598 shared address space (CGNAT)
	netip.MustParsePrefix("127.0.0.0/8"),     // loopback
	netip.MustParsePrefix("169.254.0.0/16"),  // link-local, includes cloud metadata
	netip.MustParsePrefix("172.16.0.0/12"),   // RFC1918 private
	netip.MustParsePrefix("192.0.0.0/24"),    // IETF protocol assignments
	netip.MustParsePrefix("192.0.2.0/24"),    // TEST-NET-1
	netip.MustParsePrefix("192.88.99.0/24"),  // RFC7526 6to4 relay anycast
	netip.MustParsePrefix("192.168.0.0/16"),  // RFC1918 private
	netip.MustParsePrefix("198.18.0.0/15"),   // RFC2544 benchmarking
	netip.MustParsePrefix("198.51.100.0/24"), // TEST-NET-2
	netip.MustParsePrefix("203.0.113.0/24"),  // TEST-NET-3
	netip.MustParsePrefix("224.0.0.0/4"),     // multicast
	netip.MustParsePrefix("240.0.0.0/4"),     // reserved, includes broadcast

	// IPv6.
	netip.MustParsePrefix("::1/128"),        // loopback
	netip.MustParsePrefix("64:ff9b::/96"),   // RFC6052 NAT64 well-known prefix
	netip.MustParsePrefix("64:ff9b:1::/48"), // RFC8215 NAT64 local-use prefix
	netip.MustParsePrefix("100::/64"),       // RFC6666 discard-only
	netip.MustParsePrefix("2001::/32"),      // RFC4380 Teredo
	netip.MustParsePrefix("2001:db8::/32"),  // documentation
	netip.MustParsePrefix("2002::/16"),      // RFC3056 6to4
	netip.MustParsePrefix("fc00::/7"),       // unique local
	netip.MustParsePrefix("fe80::/10"),      // link-local
	netip.MustParsePrefix("ff00::/8"),       // multicast

	// IPv4-compatible IPv6 (::x.x.x.x), deprecated by RFC4291 but still
	// parsed. Unmap leaves this form alone, so the IPv4 prefixes above do not
	// apply to it. Covers the unspecified address too.
	netip.MustParsePrefix("::/96"),
}

// isPrivateOrInternal reports whether an IP address is private, internal, or
// otherwise unsafe to send an outbound request to.
func isPrivateOrInternal(ip net.IP) bool {
	addr, ok := netip.AddrFromSlice(ip)
	if !ok {
		// Not an address we can reason about, so refuse rather than allow.
		return true
	}

	// Treat an IPv4 address written in IPv6 form (::ffff:127.0.0.1) as the
	// IPv4 address it names, so the IPv4 prefixes apply to it.
	addr = addr.Unmap()

	return slices.ContainsFunc(blockedPrefixes, func(p netip.Prefix) bool {
		return p.Contains(addr)
	})
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
