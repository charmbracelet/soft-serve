package common

import (
	"fmt"
	"net/url"

	"github.com/charmbracelet/soft-serve/pkg/utils"
	"github.com/muesli/reflow/truncate"
)

// TruncateString is a convenient wrapper around truncate.TruncateString.
func TruncateString(s string, max int) string { //nolint:revive
	if max < 0 {
		max = 0 //nolint:revive
	}
	return truncate.StringWithTail(s, uint(max), "…") //nolint:gosec
}

// RepoURL returns the URL of the repository.
func RepoURL(publicURL, name string) string {
	name = utils.SanitizeRepo(name) + ".git"
	url, err := url.Parse(publicURL)
	if err == nil {
		switch url.Scheme {
		case "ssh":
			port := url.Port()
			if port == "" || port == "22" {
				return fmt.Sprintf("git@%s:%s", url.Hostname(), name)
			}
			return fmt.Sprintf("ssh://%s:%s/%s", url.Hostname(), url.Port(), name)
		}
	}

	return fmt.Sprintf("%s/%s", publicURL, name)
}
