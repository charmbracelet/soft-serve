package utils

import (
	"path/filepath"
	"strings"
)

// SanitizeRepo returns a sanitized version of the given repository name.
func SanitizeRepo(repo string) string {
	repo = strings.TrimPrefix(repo, "/")
	repo = filepath.Clean(repo)
	repo = strings.TrimSuffix(repo, ".git")
	return repo
}
