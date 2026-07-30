package ssrf

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/netip"
	"net/url"
	"strconv"
	"strings"
)

var (
	// ErrUnsupportedRemoteScheme is returned when a git remote uses a scheme
	// that is neither a supported network transport nor safe to hand to the
	// git subprocess (e.g. file://, ext::, or a bare local path).
	ErrUnsupportedRemoteScheme = errors.New("remote scheme is not allowed")
	// ErrAmbiguousHost is returned when a remote host is neither a canonical
	// IP literal nor a valid DNS name. Non-canonical IPv4 literals such as
	// "0177.0.0.1" land here: Go reads them as one address and libcurl reads
	// them as another, so they can never be validated safely.
	ErrAmbiguousHost = errors.New("remote host is not a canonical IP or DNS name")
)

// GitRemoteTransport describes how a validated remote will be reached.
type GitRemoteTransport int

const (
	// GitTransportHTTP is an http/https remote. These are validated and
	// pinned to the resolved IP.
	GitTransportHTTP GitRemoteTransport = iota
	// GitTransportGit is a git:// remote. These are validated but cannot be
	// pinned, since the git protocol dials directly rather than via libcurl.
	GitTransportGit
	// GitTransportSSH is an ssh remote. Reachability is governed by the
	// operator's SSH client key, so these are not IP-validated.
	GitTransportSSH
)

// GitConfigEntry is a single git configuration key/value pair.
type GitConfigEntry struct {
	Key   string
	Value string
}

// ValidatedGitRemote is the result of validating a git remote URL.
type ValidatedGitRemote struct {
	// Transport is the transport the remote will use.
	Transport GitRemoteTransport
	// Config holds per-remote git configuration required for the validation
	// to hold, such as pinning a hostname to the address that was validated.
	// Render it with GitEnv, which also applies the settings that are needed
	// regardless of remote.
	Config []GitConfigEntry
}

// GitEnv renders the environment that must be passed to any git subprocess
// touching the given validated remotes.
//
// Validation alone is not sufficient for a subprocess: git resolves hostnames
// again itself, and follows the first HTTP redirect by default. Both walk
// straight through a validate-then-exec check, so this environment disables
// redirects and pins each validated address. Callers MUST apply it to every
// git command that touches the remote, or the validation means nothing.
func GitEnv(remotes ...ValidatedGitRemote) []string {
	// git follows the first HTTP redirect by default, which lets a public URL
	// hand off to an internal one after validation has already passed. This
	// costs imports of moved repositories, which is the right trade for a
	// user-supplied remote.
	entries := []GitConfigEntry{{Key: "http.followRedirects", Value: "false"}}
	for _, r := range remotes {
		entries = append(entries, r.Config...)
	}

	env := []string{fmt.Sprintf("GIT_CONFIG_COUNT=%d", len(entries))}
	for i, e := range entries {
		env = append(env,
			fmt.Sprintf("GIT_CONFIG_KEY_%d=%s", i, e.Key),
			fmt.Sprintf("GIT_CONFIG_VALUE_%d=%s", i, e.Value),
		)
	}
	return env
}

// ValidateGitRemote validates a git remote URL against private, internal, and
// loopback address ranges.
//
// The result must be passed to GitEnv, and that environment applied to every
// git subprocess touching the remote. Validation on its own does not survive
// contact with git, which re-resolves hostnames and follows redirects.
//
// SSH remotes are allowed without IP validation: they authenticate with the
// operator's own client key rather than anything an importing user controls.
func ValidateGitRemote(remote string) (ValidatedGitRemote, error) {
	if remote == "" {
		return ValidatedGitRemote{}, ErrInvalidURL
	}

	u, err := url.Parse(remote)
	if err != nil {
		// Bare SSH syntax (git@host:path) is not a parseable URL. Anything
		// else that fails to parse is rejected rather than passed through.
		if isBareSSHRemote(remote) {
			return ValidatedGitRemote{Transport: GitTransportSSH}, nil
		}
		return ValidatedGitRemote{}, fmt.Errorf("%w: %v", ErrInvalidURL, err)
	}

	switch u.Scheme {
	case "ssh", "git+ssh", "ssh+git":
		return ValidatedGitRemote{Transport: GitTransportSSH}, nil

	case "git":
		// git:// is a plain TCP connect to an arbitrary host:port, so it is
		// as much an SSRF primitive as http and gets the same validation.
		// It cannot be pinned, since libcurl is not involved.
		if _, err := validateRemoteHost(u.Hostname()); err != nil {
			return ValidatedGitRemote{}, err
		}
		return ValidatedGitRemote{Transport: GitTransportGit}, nil

	case "http", "https":
		addr, err := validateRemoteHost(u.Hostname())
		if err != nil {
			return ValidatedGitRemote{}, err
		}
		// Pin the validated address so git's own resolution cannot return a
		// different (private) one than the address we just checked.
		return ValidatedGitRemote{
			Transport: GitTransportHTTP,
			Config:    curlResolvePin(u, addr),
		}, nil

	default:
		// file://, ext::, and bare local paths land here. None are network
		// remotes, and ext:: is arbitrary command execution.
		return ValidatedGitRemote{}, fmt.Errorf("%w: %q", ErrUnsupportedRemoteScheme, u.Scheme)
	}
}

// validateRemoteHost checks a remote host against private and internal ranges,
// returning the validated address so callers can pin it.
func validateRemoteHost(host string) (netip.Addr, error) {
	if host == "" {
		return netip.Addr{}, fmt.Errorf("%w: missing hostname", ErrInvalidURL)
	}

	if isLocalhost(host) {
		return netip.Addr{}, ErrPrivateIP
	}

	// netip.ParseAddr accepts only canonical forms, so non-canonical IPv4
	// literals (octal, hex, short-form) fall through to the DNS name check
	// below and are rejected there rather than being resolved.
	if addr, err := netip.ParseAddr(host); err == nil {
		if isPrivateOrInternal(net.IP(addr.AsSlice())) {
			return netip.Addr{}, ErrPrivateIP
		}
		return addr, nil
	}

	if !isDNSName(host) {
		return netip.Addr{}, fmt.Errorf("%w: %q", ErrAmbiguousHost, host)
	}

	ips, err := net.DefaultResolver.LookupIPAddr(context.Background(), host)
	if err != nil {
		return netip.Addr{}, fmt.Errorf("%w: cannot resolve hostname: %v", ErrInvalidURL, err)
	}
	if len(ips) == 0 {
		return netip.Addr{}, fmt.Errorf("%w: no addresses for hostname", ErrInvalidURL)
	}

	// Every resolved address must be public. A hostname answering with both
	// a public and a private address is a rebinding attempt, not a remote.
	for _, ip := range ips {
		if isPrivateOrInternal(ip.IP) {
			return netip.Addr{}, ErrPrivateIP
		}
	}

	addr, ok := netip.AddrFromSlice(ips[0].IP)
	if !ok {
		return netip.Addr{}, fmt.Errorf("%w: unusable address for hostname", ErrInvalidURL)
	}
	return addr.Unmap(), nil
}

// isDNSName reports whether a host is a valid DNS name. The final label must
// not be all-numeric (RFC 1123 section 2.1), which is what separates a real
// hostname from an IPv4 literal that netip rejected as non-canonical.
func isDNSName(host string) bool {
	host = strings.TrimSuffix(host, ".")
	if host == "" || len(host) > 253 {
		return false
	}

	labels := strings.Split(host, ".")
	for _, label := range labels {
		if label == "" || len(label) > 63 {
			return false
		}
		if label[0] == '-' || label[len(label)-1] == '-' {
			return false
		}
		for i := range label {
			c := label[i]
			isAlnum := (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')
			if !isAlnum && c != '-' && c != '_' {
				return false
			}
		}
	}

	// An all-numeric final label means this was meant as an IP address. Since
	// netip already rejected it as non-canonical, refuse rather than guess.
	last := labels[len(labels)-1]
	if _, err := strconv.ParseUint(last, 0, 64); err == nil {
		return false
	}

	return true
}

// curlResolvePin returns git configuration pinning the URL's host to the
// validated address, or nothing if the URL needs no pinning.
func curlResolvePin(u *url.URL, addr netip.Addr) []GitConfigEntry {
	// An IP literal is already pinned by construction.
	if !addr.IsValid() || isIPLiteral(u.Hostname()) {
		return nil
	}

	port := u.Port()
	if port == "" {
		port = "80"
		if u.Scheme == "https" {
			port = "443"
		}
	}

	return []GitConfigEntry{{
		Key:   "http.curloptResolve",
		Value: fmt.Sprintf("%s:%s:%s", u.Hostname(), port, addr.String()),
	}}
}

// isIPLiteral reports whether a host is already a canonical IP address.
func isIPLiteral(host string) bool {
	_, err := netip.ParseAddr(host)
	return err == nil
}

// isBareSSHRemote reports whether a remote uses scp-like syntax
// (user@host:path), which is an SSH remote rather than a malformed URL.
func isBareSSHRemote(remote string) bool {
	at := strings.Index(remote, "@")
	if at <= 0 {
		return false
	}
	return strings.Contains(remote[at:], ":")
}
