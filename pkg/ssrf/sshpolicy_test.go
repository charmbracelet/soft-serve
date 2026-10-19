package ssrf

import (
	"context"
	"errors"
	"net"
	"testing"
)

// mustAllowedHosts parses entries or fails the test.
func mustAllowedHosts(t *testing.T, entries ...string) AllowedHosts {
	t.Helper()
	a, err := ParseAllowedHosts(entries)
	if err != nil {
		t.Fatalf("ParseAllowedHosts(%q): %v", entries, err)
	}
	return a
}

// fakeResolver answers from a fixed table and reports anything else as not
// found, so CIDR matching can be tested without DNS.
func fakeResolver(t *testing.T, table map[string][]string) {
	t.Helper()
	orig := lookupIPAddr
	lookupIPAddr = func(_ context.Context, host string) ([]net.IPAddr, error) {
		ips, ok := table[host]
		if !ok {
			return nil, &net.DNSError{Err: "no such host", Name: host, IsNotFound: true}
		}
		var out []net.IPAddr
		for _, ip := range ips {
			out = append(out, net.IPAddr{IP: net.ParseIP(ip)})
		}
		return out, nil
	}
	t.Cleanup(func() { lookupIPAddr = orig })
}

func TestAllowedHostsMatch(t *testing.T) {
	fakeResolver(t, map[string][]string{
		"inside.example":  {"10.1.2.3"},
		"split.example":   {"10.1.2.3", "203.0.113.9"},
		"outside.example": {"203.0.113.9"},
		"v6.example":      {"fd00::1"},
	})

	tests := []struct {
		name    string
		allow   []string
		host    string
		port    string
		wantErr error
	}{
		// Hostnames match by name, case-insensitively, trailing dot ignored.
		{"hostname match", []string{"git.internal.example"}, "git.internal.example", "", nil},
		{"hostname case", []string{"Git.Internal.Example"}, "git.internal.example.", "", nil},
		{"hostname mismatch", []string{"git.internal.example"}, "evil.example", "", ErrSSHHostNotAllowed},
		{"hostname subdomain is not a match", []string{"internal.example"}, "git.internal.example", "", ErrSSHHostNotAllowed},

		// IP literals and CIDRs.
		{"ip literal entry", []string{"10.1.2.3"}, "10.1.2.3", "", nil},
		{"ip literal other", []string{"10.1.2.3"}, "10.1.2.4", "", ErrSSHHostNotAllowed},
		{"cidr match", []string{"10.0.0.0/8"}, "10.20.30.40", "", nil},
		{"cidr miss", []string{"10.0.0.0/8"}, "192.168.1.1", "", ErrSSHHostNotAllowed},
		{"cidr v6 match", []string{"fd00::/8"}, "fd12::1", "", nil},
		{"ipv4-mapped matches v4 cidr", []string{"10.0.0.0/8"}, "::ffff:10.1.1.1", "", nil},
		{"ip literal not covered by hostname entry", []string{"git.internal.example"}, "10.0.0.1", "", ErrSSHHostNotAllowed},

		// Hostnames not named by an entry must resolve into an allowed range,
		// with every address inside it.
		{"resolves inside cidr", []string{"10.0.0.0/8"}, "inside.example", "", nil},
		{"resolves inside v6 cidr", []string{"fd00::/8"}, "v6.example", "", nil},
		{"resolves outside cidr", []string{"10.0.0.0/8"}, "outside.example", "", ErrSSHHostNotAllowed},
		{"resolves partly outside cidr", []string{"10.0.0.0/8"}, "split.example", "", ErrSSHHostNotAllowed},
		{"unresolvable host fails closed", []string{"10.0.0.0/8"}, "does-not-exist.invalid", "", ErrSSHHostNotAllowed},
		{"name entry does not trigger lookup", []string{"outside.example"}, "outside.example", "", nil},

		// Ports. No port on the entry means any port; a port on the entry
		// must match, with a missing remote port meaning 22.
		{"entry without port, any port", []string{"gitlab.corp"}, "gitlab.corp", "2222", nil},
		{"entry port match", []string{"gitlab.corp:2222"}, "gitlab.corp", "2222", nil},
		{"entry port mismatch", []string{"gitlab.corp:2222"}, "gitlab.corp", "22", ErrSSHHostNotAllowed},
		{"entry port 22 matches default", []string{"gitlab.corp:22"}, "gitlab.corp", "", nil},
		{"entry port mismatch default", []string{"gitlab.corp:2222"}, "gitlab.corp", "", ErrSSHHostNotAllowed},
		{"cidr with port match", []string{"10.0.0.0/8:2222"}, "10.1.1.1", "2222", nil},
		{"cidr with port mismatch", []string{"10.0.0.0/8:2222"}, "10.1.1.1", "", ErrSSHHostNotAllowed},
		{"bracketed v6 cidr with port", []string{"[fd00::/8]:2222"}, "fd00::1", "2222", nil},
		{"bracketed v6 cidr with port mismatch", []string{"[fd00::/8]:2222"}, "fd00::1", "22", ErrSSHHostNotAllowed},

		{"empty allowlist", nil, "git.internal.example", "", ErrSSHHostNotAllowed},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := mustAllowedHosts(t, tt.allow...).Match(tt.host, tt.port)
			if tt.wantErr == nil && err != nil {
				t.Fatalf("Match(%q, %q) = %v, want nil", tt.host, tt.port, err)
			}
			if tt.wantErr != nil && !errors.Is(err, tt.wantErr) {
				t.Fatalf("Match(%q, %q) = %v, want %v", tt.host, tt.port, err, tt.wantErr)
			}
		})
	}
}

// TestAllowedHostsMatchResolved covers the real resolver, which the table
// above replaces so it runs offline.
func TestAllowedHostsMatchResolved(t *testing.T) {
	if testing.Short() {
		t.Skip("requires DNS")
	}

	// localhost resolves to loopback on every platform this runs on.
	if err := mustAllowedHosts(t, "127.0.0.0/8", "::1/128").Match("localhost", ""); err != nil {
		t.Errorf("localhost should resolve inside loopback: %v", err)
	}
	if err := mustAllowedHosts(t, "10.0.0.0/8").Match("localhost", ""); !errors.Is(err, ErrSSHHostNotAllowed) {
		t.Errorf("localhost resolved outside 10/8 but was allowed: %v", err)
	}
}

func TestParseAllowedHostsRejectsBadEntries(t *testing.T) {
	for _, entry := range []string{
		"",
		"   ",
		"10.0.0.0/33",
		"not a host",
		"host:notaport",
		"host:70000",
		":22",
		"0177.0.0.1",
		"-oProxyCommand=id",
	} {
		if _, err := ParseAllowedHosts([]string{entry}); err == nil {
			t.Errorf("ParseAllowedHosts(%q) accepted a bad entry", entry)
		}
	}
}

// TestValidateGitRemotePolicy covers the ssh policy as applied through
// ValidateGitRemote, for both URL and scp-style remotes.
func TestValidateGitRemotePolicy(t *testing.T) {
	fakeResolver(t, nil)

	listed := GitRemotePolicy{AllowedSSHHosts: mustAllowedHosts(t, "git.internal.example", "10.0.0.0/8", "gitlab.corp:2222")}
	// A non-empty allowlist binds admins too.
	listedAdmin := GitRemotePolicy{AllowedSSHHosts: listed.AllowedSSHHosts, AllowAnySSHHost: true}
	nobody := GitRemotePolicy{}
	admin := GitRemotePolicy{AllowAnySSHHost: true}

	tests := []struct {
		name    string
		remote  string
		policy  GitRemotePolicy
		wantErr error
	}{
		// Non-ssh transports are untouched by the ssh policy.
		{"https ignores policy", "https://1.1.1.1/x.git", nobody, nil},
		{"https still address checked", "https://10.0.0.1/x.git", listed, ErrPrivateIP},
		{"git ignores policy", "git://1.1.1.1/x.git", nobody, nil},

		// With an allowlist, the host must be on it, for everyone.
		{"ssh url allowed by name", "ssh://git@git.internal.example/x.git", listed, nil},
		{"ssh url allowed by cidr", "ssh://git@10.2.3.4/x.git", listed, nil},
		{"ssh url denied", "ssh://git@evil.example/x.git", listed, ErrSSHHostNotAllowed},
		{"ssh url denied for admin too", "ssh://git@evil.example/x.git", listedAdmin, ErrSSHHostNotAllowed},
		{"ssh url wrong port", "ssh://git@gitlab.corp/x.git", listed, ErrSSHHostNotAllowed},
		{"ssh url right port", "ssh://git@gitlab.corp:2222/x.git", listed, nil},
		{"ssh url ipv6 port", "ssh://git@[fd00::1]:2222/x.git", GitRemotePolicy{AllowedSSHHosts: mustAllowedHosts(t, "[fd00::/8]:2222")}, nil},
		{"git+ssh denied", "git+ssh://git@evil.example/x.git", listed, ErrSSHHostNotAllowed},
		{"ssh+git allowed", "ssh+git://git@git.internal.example/x.git", listed, nil},
		{"scp allowed", "git@git.internal.example:x.git", listed, nil},
		{"scp denied", "git@evil.example:x.git", listed, ErrSSHHostNotAllowed},
		{"scp bracketed port right", "[git@gitlab.corp:2222]:x.git", listed, nil},
		{"scp bracketed port wrong", "[git@gitlab.corp:22]:x.git", listed, ErrSSHHostNotAllowed},
		{"scp cidr", "git@10.9.9.9:x.git", listed, nil},
		{"scp ipv6 literal", "git@[fd00::1]:x.git", GitRemotePolicy{AllowedSSHHosts: mustAllowedHosts(t, "fd00::/8")}, nil},

		// With no allowlist, only a policy that allows any host passes.
		{"no allowlist, admin", "ssh://git@10.0.0.1/x.git", admin, nil},
		{"no allowlist, scp admin", "git@10.0.0.1:x.git", admin, nil},
		{"no allowlist, user", "ssh://git@10.0.0.1/x.git", nobody, ErrSSHRequiresAdmin},
		{"no allowlist, scp user", "git@10.0.0.1:x.git", nobody, ErrSSHRequiresAdmin},
		{"no allowlist, user bracketed", "[git@10.0.0.1:2222]:x.git", nobody, ErrSSHRequiresAdmin},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v, err := ValidateGitRemote(tt.remote, tt.policy)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("error = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if v.Transport == GitTransportSSH && len(v.Config) != 0 {
				t.Errorf("ssh remote carries git config: %v", v.Config)
			}
		})
	}
}

func TestSplitHostPort(t *testing.T) {
	tests := []struct {
		in, host, port string
	}{
		{"example.com", "example.com", ""},
		{"example.com:2222", "example.com", "2222"},
		{"10.0.0.0/8", "10.0.0.0/8", ""},
		{"10.0.0.0/8:2222", "10.0.0.0/8", "2222"},
		{"fd00::1", "fd00::1", ""},
		{"fd00::/8", "fd00::/8", ""},
		{"[fd00::1]", "fd00::1", ""},
		{"[fd00::1]:2222", "fd00::1", "2222"},
		{"[fd00::/8]:22", "fd00::/8", "22"},
		{"fe80::1%eth0", "fe80::1%eth0", ""},
	}
	for _, tt := range tests {
		host, port := splitHostPort(tt.in)
		if host != tt.host || port != tt.port {
			t.Errorf("splitHostPort(%q) = %q, %q; want %q, %q", tt.in, host, port, tt.host, tt.port)
		}
	}
}
