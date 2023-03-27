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
func RepoURL(publicURL, name string) string {
	return fmt.Sprintf("git clone %s/%s", publicURL, name)
}
