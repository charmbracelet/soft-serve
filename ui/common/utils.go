package common

import (
	"fmt"

	"github.com/muesli/reflow/truncate"
)

// TruncateString is a convenient wrapper around truncate.TruncateString.
func TruncateString(s string, max int) string {
	if max < 0 {
		max = 0
	}
	return truncate.StringWithTail(s, uint(max), "…")
}

// RepoURL returns the URL of the repository.
func RepoURL(host string, port string, name string) string {
	p := ""
	if port != "22" {
		p += ":" + port
	}
	return fmt.Sprintf("git clone ssh://%s/%s", host+p, name)
}
