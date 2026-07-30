package ssrf

import (
	"errors"
	"net/netip"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"testing"
)

func TestValidateGitRemote(t *testing.T) {
	tests := []struct {
		name      string
		remote    string
		transport GitRemoteTransport
		wantErr   error
	}{
		// Allowed network remotes. IP literals only, so the suite does not
		// depend on DNS. Hostname resolution is covered separately.
		{"public http", "http://1.1.1.1/x.git", GitTransportHTTP, nil},
		{"public https", "https://1.1.1.1/x.git", GitTransportHTTP, nil},
		{"public https with port", "https://1.1.1.1:8443/x.git", GitTransportHTTP, nil},
		{"public ipv6", "https://[2606:4700::1111]/x.git", GitTransportHTTP, nil},

		// SSH is reachability-governed by the operator's client key.
		{"ssh url", "ssh://git@10.0.0.1/x.git", GitTransportSSH, nil},
		{"bare ssh", "git@github.com:charmbracelet/soft-serve.git", GitTransportSSH, nil},
		{"git+ssh", "git+ssh://git@example.com/x.git", GitTransportSSH, nil},
		{"ssh+git", "ssh+git://git@example.com/x.git", GitTransportSSH, nil},

		// Loopback and private ranges over http.
		{"loopback", "http://127.0.0.1/x.git", 0, ErrPrivateIP},
		{"localhost", "http://localhost:8080/x.git", 0, ErrPrivateIP},
		{"private 10.x", "http://10.0.0.1/x.git", 0, ErrPrivateIP},
		{"private 192.168.x", "http://192.168.1.1/x.git", 0, ErrPrivateIP},
		{"cloud metadata", "http://169.254.169.254/latest/meta-data/", 0, ErrPrivateIP},
		{"ipv6 loopback", "http://[::1]/x.git", 0, ErrPrivateIP},
		{"ipv4-mapped loopback", "http://[::ffff:127.0.0.1]/x.git", 0, ErrPrivateIP},

		// git:// is a raw TCP connect and must be validated the same way.
		// Regression: an earlier fix checked http(s) on import but let
		// git:// through on mirror sync.
		{"git scheme private", "git://10.0.0.1/x.git", 0, ErrPrivateIP},
		{"git scheme metadata", "git://169.254.169.254/x.git", 0, ErrPrivateIP},
		{"git scheme public", "git://1.1.1.1/x.git", GitTransportGit, nil},

		// Non-canonical IPv4 literals parse differently in Go and libcurl.
		// Regression: net.ParseIP reads 0177.0.0.1 as public 177.0.0.1 while
		// libcurl reads it as loopback.
		{"octal loopback", "http://0177.0.0.1/x.git", 0, ErrAmbiguousHost},
		{"hex loopback", "http://0x7f.1/x.git", 0, ErrAmbiguousHost},
		{"short-form loopback", "http://127.1/x.git", 0, ErrAmbiguousHost},
		{"decimal loopback", "http://2130706433/x.git", 0, ErrAmbiguousHost},
		{"octal private", "http://010.0.0.1/x.git", 0, ErrAmbiguousHost},

		// Non-network schemes are not remotes at all.
		{"file scheme", "file:///etc/passwd", 0, ErrUnsupportedRemoteScheme},
		{"ext scheme", "ext::sh -c whoami", 0, ErrUnsupportedRemoteScheme},
		{"local path", "/data/repos/secret.git", 0, ErrUnsupportedRemoteScheme},

		{"empty", "", 0, ErrInvalidURL},
		{"no host", "https:///x.git", 0, ErrInvalidURL},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ValidateGitRemote(tt.remote)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("ValidateGitRemote(%q) error = %v, want %v", tt.remote, err, tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Fatalf("ValidateGitRemote(%q) unexpected error: %v", tt.remote, err)
			}
			if got.Transport != tt.transport {
				t.Errorf("ValidateGitRemote(%q) transport = %v, want %v", tt.remote, got.Transport, tt.transport)
			}
		})
	}
}

// TestValidateGitRemoteResolvesHostnames covers the DNS path, which the main
// table deliberately avoids so it can run offline.
func TestValidateGitRemoteResolvesHostnames(t *testing.T) {
	if testing.Short() {
		t.Skip("requires DNS")
	}

	v, err := ValidateGitRemote("https://github.com/charmbracelet/soft-serve.git")
	if err != nil {
		t.Fatalf("public hostname rejected: %v", err)
	}
	if v.Transport != GitTransportHTTP {
		t.Errorf("transport = %v, want %v", v.Transport, GitTransportHTTP)
	}
	// A resolved hostname must be pinned, or git re-resolves it.
	if len(v.Config) == 0 {
		t.Error("resolved hostname was not pinned")
	}
}

// TestGitEnvDisablesRedirects covers the bypass where a validated public URL
// 302s to an internal address. git follows the first redirect by default, so
// validation is meaningless without this.
func TestGitEnvDisablesRedirects(t *testing.T) {
	for _, name := range []string{"no remotes", "one remote"} {
		t.Run(name, func(t *testing.T) {
			var env []string
			if name == "no remotes" {
				env = GitEnv()
			} else {
				env = GitEnv(ValidatedGitRemote{Transport: GitTransportHTTP})
			}
			if !hasGitConfig(env, "http.followRedirects", "false") {
				t.Errorf("GitEnv did not disable redirects: %v", env)
			}
		})
	}
}

// TestCurlResolvePin covers DNS rebinding: git re-resolves the hostname
// itself, so the validated address has to be pinned for the check to survive
// into the subprocess.
func TestCurlResolvePin(t *testing.T) {
	tests := []struct {
		name string
		url  string
		addr string
		want string
	}{
		{"https default port", "https://example.com/x.git", "1.1.1.1", "example.com:443:1.1.1.1"},
		{"http default port", "http://example.com/x.git", "1.1.1.1", "example.com:80:1.1.1.1"},
		{"explicit port", "https://example.com:8443/x.git", "1.1.1.1", "example.com:8443:1.1.1.1"},
		{"ipv6 address", "https://example.com/x.git", "2606:4700::1111", "example.com:443:2606:4700::1111"},
		// An IP literal is pinned by construction; nothing to re-resolve.
		{"ip literal needs no pin", "https://1.1.1.1/x.git", "1.1.1.1", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u, err := url.Parse(tt.url)
			if err != nil {
				t.Fatalf("bad test url: %v", err)
			}

			got := curlResolvePin(u, netip.MustParseAddr(tt.addr))

			if tt.want == "" {
				if len(got) != 0 {
					t.Fatalf("curlResolvePin(%q) = %v, want none", tt.url, got)
				}
				return
			}

			if len(got) != 1 {
				t.Fatalf("curlResolvePin(%q) = %v, want 1 entry", tt.url, got)
			}
			if got[0].Key != "http.curloptResolve" {
				t.Errorf("key = %q, want http.curloptResolve", got[0].Key)
			}
			if got[0].Value != tt.want {
				t.Errorf("value = %q, want %q", got[0].Value, tt.want)
			}
		})
	}
}

// TestGitEnvCountMatchesEntries guards the encoding itself. git reads exactly
// GIT_CONFIG_COUNT entries, so a miscount silently drops the security settings
// rather than failing loudly.
func TestGitEnvCountMatchesEntries(t *testing.T) {
	pinned := ValidatedGitRemote{
		Transport: GitTransportHTTP,
		Config: []GitConfigEntry{{
			Key:   "http.curloptResolve",
			Value: "example.com:443:1.1.1.1",
		}},
	}

	for _, tc := range []struct {
		name    string
		remotes []ValidatedGitRemote
	}{
		{"none", nil},
		{"unpinned", []ValidatedGitRemote{{Transport: GitTransportSSH}}},
		{"one pinned", []ValidatedGitRemote{pinned}},
		{"multiple pinned", []ValidatedGitRemote{pinned, pinned}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			env := GitEnv(tc.remotes...)

			var count string
			keys := 0
			for _, e := range env {
				switch {
				case strings.HasPrefix(e, "GIT_CONFIG_COUNT="):
					count = strings.TrimPrefix(e, "GIT_CONFIG_COUNT=")
				case strings.HasPrefix(e, "GIT_CONFIG_KEY_"):
					keys++
				}
			}

			if want := strconv.Itoa(keys); count != want {
				t.Errorf("GIT_CONFIG_COUNT = %q, want %q (env: %v)", count, want, env)
			}
			// Every key must have a matching value at the same index.
			for i := range keys {
				if !slices.ContainsFunc(env, func(e string) bool {
					return strings.HasPrefix(e, "GIT_CONFIG_VALUE_"+strconv.Itoa(i)+"=")
				}) {
					t.Errorf("GIT_CONFIG_KEY_%d has no matching value: %v", i, env)
				}
			}
		})
	}
}

func TestIsDNSName(t *testing.T) {
	tests := []struct {
		host string
		want bool
	}{
		{"example.com", true},
		{"sub.example.com", true},
		{"example.com.", true},
		{"my-host", true},

		// All-numeric final label means an IP was intended. netip already
		// rejected these as non-canonical, so they must not be resolved.
		{"0177.0.0.1", false},
		{"127.1", false},
		{"2130706433", false},
		{"0x7f.1", false},

		{"", false},
		{"-leading.com", false},
		{"trailing-.com", false},
		{"has space.com", false},
		{"double..dot", false},
	}

	for _, tt := range tests {
		t.Run(tt.host, func(t *testing.T) {
			if got := isDNSName(tt.host); got != tt.want {
				t.Errorf("isDNSName(%q) = %v, want %v", tt.host, got, tt.want)
			}
		})
	}
}

func hasGitConfig(env []string, key, value string) bool {
	for _, e := range env {
		name, v, ok := strings.Cut(e, "=")
		if !ok || !strings.HasPrefix(name, "GIT_CONFIG_KEY_") || v != key {
			continue
		}
		idx := strings.TrimPrefix(name, "GIT_CONFIG_KEY_")
		if slices.Contains(env, "GIT_CONFIG_VALUE_"+idx+"="+value) {
			return true
		}
	}
	return false
}
