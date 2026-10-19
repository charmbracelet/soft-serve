package ssrf

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/netip"
	"strconv"
	"strings"
)

var (
	// ErrSSHHostNotAllowed is returned when an ssh remote names a host that is
	// not on the operator's allowlist.
	ErrSSHHostNotAllowed = errors.New("remote host is not allowed")
	// ErrSSHRequiresAdmin is returned when no ssh allowlist is configured and
	// the caller is not allowed to pick ssh hosts on their own.
	ErrSSHRequiresAdmin = errors.New("ssh remotes require a server admin when no allowed hosts are configured")
)

// GitRemotePolicy is the operator policy applied on top of the address
// validation in ValidateGitRemote.
type GitRemotePolicy struct {
	// AllowedSSHHosts lists the hosts an ssh remote may name. The server's
	// client key is presented to every ssh host a user imports from, so this
	// is the set of hosts the operator trusts that key with. When non-empty
	// it binds everyone, admins included.
	AllowedSSHHosts AllowedHosts
	// AllowAnySSHHost permits ssh remotes to any host when AllowedSSHHosts is
	// empty. It is set for server admins, who may make that call themselves;
	// everyone else gets ErrSSHRequiresAdmin.
	AllowAnySSHHost bool
}

// checkSSH applies the policy to an ssh host and port ("" meaning the ssh
// default).
func (p GitRemotePolicy) checkSSH(host, port string) error {
	if len(p.AllowedSSHHosts) > 0 {
		return p.AllowedSSHHosts.Match(host, port)
	}
	if p.AllowAnySSHHost {
		return nil
	}
	return ErrSSHRequiresAdmin
}

// AllowedHosts is a parsed host allowlist. Build one with ParseAllowedHosts.
type AllowedHosts []allowedHost

// allowedHost is one allowlist entry: either a hostname or a prefix, plus an
// optional port.
type allowedHost struct {
	name   string
	prefix netip.Prefix
	port   string
}

// ParseAllowedHosts parses allowlist entries. Each is a hostname, IP address,
// or CIDR range, with an optional port:
//
//	git.internal.example, gitlab.corp:2222, 10.0.0.0/8, [fd00::/8]:22
//
// It fails on the first entry it cannot read, so a typo in the config is
// reported when the config loads instead of silently matching nothing.
func ParseAllowedHosts(entries []string) (AllowedHosts, error) {
	hosts := make(AllowedHosts, 0, len(entries))
	for _, raw := range entries {
		h, err := parseAllowedHost(raw)
		if err != nil {
			return nil, err
		}
		hosts = append(hosts, h)
	}
	return hosts, nil
}

func parseAllowedHost(raw string) (allowedHost, error) {
	entry := strings.TrimSpace(raw)
	if entry == "" {
		return allowedHost{}, fmt.Errorf("allowed host %q: empty entry", raw)
	}

	host, port := splitHostPort(entry)
	if port != "" {
		if _, err := strconv.ParseUint(port, 10, 16); err != nil {
			return allowedHost{}, fmt.Errorf("allowed host %q: invalid port", raw)
		}
	}
	if host == "" {
		return allowedHost{}, fmt.Errorf("allowed host %q: missing host", raw)
	}

	if strings.Contains(host, "/") {
		p, err := netip.ParsePrefix(host)
		if err != nil {
			return allowedHost{}, fmt.Errorf("allowed host %q: invalid network", raw)
		}
		return allowedHost{prefix: p.Masked(), port: port}, nil
	}
	if addr, err := netip.ParseAddr(host); err == nil {
		return allowedHost{prefix: netip.PrefixFrom(addr.Unmap(), addr.Unmap().BitLen()), port: port}, nil
	}
	if !isDNSName(host) {
		return allowedHost{}, fmt.Errorf("allowed host %q: not a hostname, address, or network", raw)
	}
	return allowedHost{name: canonicalHostname(host), port: port}, nil
}

// lookupIPAddr resolves hostnames for CIDR matching. Tests replace it so the
// matcher can be exercised without DNS.
var lookupIPAddr = net.DefaultResolver.LookupIPAddr

// Match reports whether host (and port, "" meaning the ssh default) is
// covered by the allowlist. Hostnames match entries by name; otherwise the
// host must resolve, and every address it resolves to must fall inside an
// allowed range. A hostname that cannot be resolved fails closed.
//
// ssh resolves the name again itself, so a CIDR match only says where the
// name pointed at validation time. Hostname entries are the robust form.
func (a AllowedHosts) Match(host, port string) error {
	if port == "" {
		port = "22"
	}
	notAllowed := fmt.Errorf("%w: %q", ErrSSHHostNotAllowed, net.JoinHostPort(host, port))

	var candidates []allowedHost
	for _, h := range a {
		if h.port == "" || h.port == port {
			candidates = append(candidates, h)
		}
	}
	if len(candidates) == 0 {
		return notAllowed
	}

	if addr, err := netip.ParseAddr(host); err == nil {
		if containsAddr(candidates, addr) {
			return nil
		}
		return notAllowed
	}

	name := canonicalHostname(host)
	hasPrefix := false
	for _, h := range candidates {
		if h.name != "" && h.name == name {
			return nil
		}
		hasPrefix = hasPrefix || h.prefix.IsValid()
	}
	if !hasPrefix {
		return notAllowed
	}

	ips, err := lookupIPAddr(context.Background(), host)
	if err != nil || len(ips) == 0 {
		return fmt.Errorf("%w: %q does not resolve", ErrSSHHostNotAllowed, host)
	}
	for _, ip := range ips {
		addr, ok := netip.AddrFromSlice(ip.IP)
		if !ok || !containsAddr(candidates, addr) {
			return fmt.Errorf("%w: %q resolves outside the allowed networks", ErrSSHHostNotAllowed, host)
		}
	}
	return nil
}

func containsAddr(hosts []allowedHost, addr netip.Addr) bool {
	addr = addr.Unmap()
	for _, h := range hosts {
		if h.prefix.IsValid() && h.prefix.Contains(addr) {
			return true
		}
	}
	return false
}

// canonicalHostname lower-cases a hostname and drops a trailing dot so the
// same name written two ways compares equal.
func canonicalHostname(host string) string {
	return strings.TrimSuffix(strings.ToLower(host), ".")
}

// splitHostPort splits an optional trailing ":port" off a host, address,
// network, or bracketed IPv6 form. Anything with more than one colon and no
// brackets is an IPv6 literal and has no port.
func splitHostPort(s string) (host, port string) {
	if strings.HasPrefix(s, "[") {
		end := strings.Index(s, "]")
		if end < 0 {
			return s, ""
		}
		host = s[1:end]
		rest := s[end+1:]
		if strings.HasPrefix(rest, ":") {
			return host, rest[1:]
		}
		return host, ""
	}

	colon := strings.LastIndex(s, ":")
	if colon < 0 || strings.Count(s, ":") > 1 {
		return s, ""
	}
	return s[:colon], s[colon+1:]
}
