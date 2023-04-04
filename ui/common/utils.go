package common

import (
	"fmt"
	"net/url"

	"github.com/charmbracelet/soft-serve/server/utils"
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
	name = utils.SanitizeRepo(name) + ".git"
	url, err := url.Parse(publicURL)
	if err == nil {
		switch url.Scheme {
		case "ssh":
			port := url.Port()
			if port == "" || port == "22" {
				return fmt.Sprintf("git clone git@%s:%s", url.Hostname(), name)
			} else {
				return fmt.Sprintf("git clone ssh://%s:%s/%s", url.Hostname(), url.Port(), name)
			}
		}
	}

	return fmt.Sprintf("git clone %s/%s", publicURL, name)
}
