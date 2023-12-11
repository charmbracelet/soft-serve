package utils

import (
	"fmt"
	"path"
	"strings"
	"unicode"
)

// SanitizeRepo returns a sanitized version of the given repository name.
func SanitizeRepo(repo string) string {
	repo = strings.TrimPrefix(repo, "/")
	// We're using path instead of filepath here because this is not OS dependent
	// looking at you Windows
	repo = path.Clean(repo)
	repo = strings.TrimSuffix(repo, ".git")
	return repo
}

// ValidateHandle returns an error if any of the given usernames are invalid.
func ValidateHandle(handle string) error {
	if handle == "" {
		return fmt.Errorf("cannot be empty")
	}

	if !unicode.IsLetter(rune(handle[0])) {
		return fmt.Errorf("must start with a letter")
	}

	for _, r := range handle {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '-' {
			return fmt.Errorf("can only contain letters, numbers, and hyphens")
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
