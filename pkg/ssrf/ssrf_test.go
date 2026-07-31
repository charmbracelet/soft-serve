package ssrf

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewSecureClientBlocksPrivateIPs(t *testing.T) {
	client := NewSecureClient()
	transport := client.Transport.(*http.Transport)

	tests := []struct {
		name    string
		addr    string
		wantErr bool
	}{
		{"block loopback", "127.0.0.1:80", true},
		{"block private 10.x", "10.0.0.1:80", true},
		{"block link-local", "169.254.169.254:80", true},
		{"block CGNAT", "100.64.0.1:80", true},
		{"allow public IP", "8.8.8.8:80", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
			defer cancel()

			conn, err := transport.DialContext(ctx, "tcp", tt.addr)
			if conn != nil {
				conn.Close()
			}

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error for %s, got none", tt.addr)
				}
			} else {
				if err != nil && errors.Is(err, ErrPrivateIP) {
					t.Errorf("should not block %s with SSRF error, got: %v", tt.addr, err)
				}
			}
		})
	}
}

func TestNewSecureClientBlocksPrivateHostnames(t *testing.T) {
	client := NewSecureClient()
	transport := client.Transport.(*http.Transport)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// "localhost" resolves to 127.0.0.1 (loopback) -- must be blocked.
	// This exercises the hostname resolution path in DialContext:
	// net.LookupIP("localhost") -> 127.0.0.1 -> isPrivateOrInternal -> blocked.
	conn, err := transport.DialContext(ctx, "tcp", "localhost:80")
	if conn != nil {
		conn.Close()
	}
	if !errors.Is(err, ErrPrivateIP) {
		t.Errorf("expected ErrPrivateIP for hostname resolving to loopback, got: %v", err)
	}
}

func TestNewSecureClientNilIPNotErrPrivateIP(t *testing.T) {
	client := NewSecureClient()
	transport := client.Transport.(*http.Transport)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	conn, err := transport.DialContext(ctx, "tcp", "not-an-ip:80")
	if conn != nil {
		conn.Close()
	}
	if err == nil {
		t.Fatal("expected error for non-IP address, got none")
	}
	if errors.Is(err, ErrPrivateIP) {
		t.Errorf("nil-IP path should not wrap ErrPrivateIP, got: %v", err)
	}
}

func TestNewSecureClientBlocksRedirects(t *testing.T) {
	redirectServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "http://8.8.8.8:8080/safe", http.StatusFound)
	}))
	defer redirectServer.Close()

	client := NewSecureClient()
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, redirectServer.URL, nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		// httptest uses 127.0.0.1, blocked by SSRF protection
		if !errors.Is(err, ErrPrivateIP) {
			t.Fatalf("Request failed with non-SSRF error: %v", err)
		}
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusFound {
		t.Errorf("Expected redirect response (302), got %d", resp.StatusCode)
	}
}

func TestIsPrivateOrInternal(t *testing.T) {
	tests := []struct {
		ip   string
		want bool
	}{
		// Public
		{"8.8.8.8", false},
		{"2001:4860:4860::8888", false},

		// Loopback
		{"127.0.0.1", true},
		{"::1", true},

		// Private ranges
		{"10.0.0.1", true},
		{"192.168.1.1", true},
		{"172.16.0.1", true},

		// Link-local (cloud metadata)
		{"169.254.169.254", true},

		// CGNAT boundaries
		{"100.64.0.1", true},
		{"100.127.255.255", true},

		// IPv6-mapped IPv4 (bypass vector the old webhook code missed)
		{"::ffff:127.0.0.1", true},
		{"::ffff:169.254.169.254", true},
		{"::ffff:8.8.8.8", false},

		// Reserved
		{"0.0.0.0", true},
		{"240.0.0.1", true},

		// IPv6 transition addresses. Each embeds an IPv4 address inside an
		// IPv6 one, so they look public to the IPv6 checks while naming an
		// internal host once a relay decodes them. The whole prefix is
		// blocked, so the embedded address does not matter: encodings of
		// public addresses are rejected too.
		{"2002:7f00:0001::", true},                        // 6to4 -> 127.0.0.1
		{"2002:a9fe:a9fe::", true},                        // 6to4 -> 169.254.169.254
		{"2002:0a00:0001::", true},                        // 6to4 -> 10.0.0.1
		{"2002:0808:0808::", true},                        // 6to4 -> 8.8.8.8, still blocked
		{"64:ff9b::7f00:1", true},                         // NAT64 well-known -> 127.0.0.1
		{"64:ff9b::a9fe:a9fe", true},                      // NAT64 well-known -> 169.254.169.254
		{"64:ff9b:1::7f00:1", true},                       // NAT64 local-use -> 127.0.0.1
		{"2001:0000:4136:e378:8000:63bf:8001:5601", true}, // Teredo
		{"::7f00:1", true},                                // IPv4-compatible -> 127.0.0.1
		{"192.88.99.1", true},                             // 6to4 relay anycast

		// Public IPv6 must stay reachable: the transition prefixes are narrow
		// and must not swallow ordinary global unicast.
		{"2606:4700:4700::1111", false},
		{"2a00:1450:4001::1", false},
		{"2400:cb00::1", false},
	}

	for _, tt := range tests {
		t.Run(tt.ip, func(t *testing.T) {
			ip := net.ParseIP(tt.ip)
			if ip == nil {
				t.Fatalf("failed to parse IP: %s", tt.ip)
			}
			if got := isPrivateOrInternal(ip); got != tt.want {
				t.Errorf("isPrivateOrInternal(%s) = %v, want %v", tt.ip, got, tt.want)
			}
		})
	}
}

// TestStdlibCoverageParity guards the switch from the standard library's
// Is*() helpers to an explicit prefix table. The helpers covered these ranges
// implicitly, so the table has to cover them too or the refactor silently
// narrows the guard.
func TestStdlibCoverageParity(t *testing.T) {
	for _, ip := range []string{
		// Loopback.
		"127.0.0.1", "127.255.255.254", "::1",
		// Private.
		"10.255.255.255", "172.31.255.255", "192.168.255.255", "fd00::1", "fdff::1",
		// Link-local unicast.
		"169.254.1.1", "fe80::1", "febf::1",
		// Multicast, including link-local and interface-local.
		"224.0.0.1", "239.255.255.255", "ff00::1", "ff01::1", "ff02::1", "ffff::1",
		// Unspecified.
		"0.0.0.0", "::",
	} {
		t.Run(ip, func(t *testing.T) {
			if !isPrivateOrInternal(net.ParseIP(ip)) {
				t.Errorf("isPrivateOrInternal(%s) = false, want true", ip)
			}
		})
	}
}

func TestValidateURL(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr bool
		errType error
	}{
		// Valid
		{"valid https", "https://1.1.1.1/webhook", false, nil},

		// Scheme validation
		{"ftp scheme", "ftp://example.com/webhook", true, ErrInvalidScheme},
		{"no scheme", "example.com/webhook", true, ErrInvalidScheme},

		// Localhost
		{"localhost", "http://localhost/webhook", true, ErrPrivateIP},
		{"subdomain.localhost", "http://test.localhost/webhook", true, ErrPrivateIP},

		// IP-based blocking (one per category -- range coverage is in TestIsPrivateOrInternal)
		{"loopback IP", "http://127.0.0.1/webhook", true, ErrPrivateIP},
		{"metadata IP", "http://169.254.169.254/latest/meta-data/", true, ErrPrivateIP},

		// Invalid URLs
		{"empty", "", true, ErrInvalidURL},
		{"missing hostname", "http:///webhook", true, ErrInvalidURL},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateURL(tt.url)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateURL(%q) error = %v, wantErr %v", tt.url, err, tt.wantErr)
				return
			}
			if tt.wantErr && tt.errType != nil {
				if !errors.Is(err, tt.errType) {
					t.Errorf("ValidateURL(%q) error = %v, want error type %v", tt.url, err, tt.errType)
				}
			}
		})
	}
}

func TestIsLocalhost(t *testing.T) {
	tests := []struct {
		hostname string
		want     bool
	}{
		{"localhost", true},
		{"LOCALHOST", true},
		{"test.localhost", true},
		{"example.com", false},
		{"localhost.com", false},
	}

	for _, tt := range tests {
		t.Run(tt.hostname, func(t *testing.T) {
			if got := isLocalhost(tt.hostname); got != tt.want {
				t.Errorf("isLocalhost(%s) = %v, want %v", tt.hostname, got, tt.want)
			}
		})
	}
}
