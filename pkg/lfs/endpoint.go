package lfs

import (
	"fmt"
	"net/url"
	"strings"
)

// Endpoint is a Git LFS endpoint.
type Endpoint = *url.URL

// NewEndpoint returns a new Git LFS endpoint.
func NewEndpoint(rawurl string) (Endpoint, error) {
	u, err := url.Parse(rawurl)
	if err != nil {
		e, err := endpointFromBareSSH(rawurl)
		if err != nil {
			return nil, err
		}
		u = e
	}

	u.Path = strings.TrimSuffix(u.Path, "/")

	switch u.Scheme {
	case "git":
		// Use https for git:// URLs and strip the port if it exists.
		u.Scheme = "https"
		if u.Port() != "" {
			u.Host = u.Hostname()
		}
		fallthrough
	case "http", "https":
		if strings.HasSuffix(u.Path, ".git") {
			u.Path += "/info/lfs"
		} else {
			u.Path += ".git/info/lfs"
		}
	case "ssh", "git+ssh", "ssh+git":
	default:
		return nil, fmt.Errorf("unknown url: %s", rawurl)
	}

	return u, nil
}

// endpointFromBareSSH creates a new endpoint from a bare ssh repo.
//
//	user@host.com:path/to/repo.git or
//	[user@host.com:port]:path/to/repo.git
func endpointFromBareSSH(rawurl string) (*url.URL, error) {
	parts := strings.Split(rawurl, ":")
	partsLen := len(parts)
	if partsLen < 2 {
		return url.Parse(rawurl)
	}

	// Treat presence of ':' as a bare URL
	var newPath string
	if len(parts) > 2 { // port included; really should only ever be 3 parts
		// Correctly handle [host:port]:path URLs
		parts[0] = strings.TrimPrefix(parts[0], "[")
		parts[1] = strings.TrimSuffix(parts[1], "]")
		newPath = fmt.Sprintf("%v:%v", parts[0], strings.Join(parts[1:], "/"))
	} else {
		newPath = strings.Join(parts, "/")
	}
	newrawurl := fmt.Sprintf("ssh://%v", newPath)
	return url.Parse(newrawurl)
}
