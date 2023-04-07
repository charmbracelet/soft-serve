package utils

import (
	"fmt"
	"path/filepath"
	"strings"
	"unicode"
)

// SanitizeRepo returns a sanitized version of the given repository name.
func SanitizeRepo(repo string) string {
	repo = strings.TrimPrefix(repo, "/")
	repo = filepath.Clean(repo)
	repo = strings.TrimSuffix(repo, ".git")
	return repo
}

// ValidateUsername returns an error if any of the given usernames are invalid.
func ValidateUsername(username string) error {
	if username == "" {
		return fmt.Errorf("username cannot be empty")
	}

	if !unicode.IsLetter(rune(username[0])) {
		return fmt.Errorf("username must start with a letter")
	}

	for _, r := range username {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '-' {
			return fmt.Errorf("username can only contain letters, numbers, and hyphens")
		}
	}

	return nil
}

// ValidateRepo returns an error if the given repository name is invalid.
func ValidateRepo(repo string) error {
	if repo == "" {
		return fmt.Errorf("repo cannot be empty")
	}

	for _, r := range repo {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '-' && r != '_' && r != '.' && r != '/' {
			return fmt.Errorf("repo can only contain letters, numbers, hyphens, underscores, periods, and slashes")
		}
	}

	return nil
}
