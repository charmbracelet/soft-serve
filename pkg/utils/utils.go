package utils

import (
	"fmt"
	"path"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/charmbracelet/x/ansi"
)

// SanitizeRepo returns a sanitized version of the given repository name.
// Returns empty string if the repository name contains path traversal sequences.
func SanitizeRepo(repo string) string {
	repo = Sanitize(repo)

	// Prevent path traversal - reject ../ and other dangerous sequences
	// These patterns could allow attackers to escape the repository root
	if strings.Contains(repo, "../") || strings.Contains(repo, "..\\") {
		return ""
	}
	if strings.HasPrefix(repo, "../") || strings.HasPrefix(repo, "..\\") {
		return ""
	}

	// Prevent absolute path escapes
	if strings.HasPrefix(repo, "/") && strings.Contains(repo[1:], "../") {
		return ""
	}

	// We need to use an absolute path for the path to be cleaned correctly.
	repo = strings.TrimPrefix(repo, "/")
	repo = "/" + repo

	// We're using path instead of filepath here because this is not OS dependent
	// looking at you Windows
	repo = path.Clean(repo)
	repo = strings.TrimSuffix(repo, ".git")

	// Final safety check: if path.Clean() resulted in path traversal, reject it
	cleaned := repo[1:]
	if strings.Contains(cleaned, "../") || strings.Contains(cleaned, "..\\") {
		return ""
	}

	return repo[1:]
}

// Sanitize strips ANSI escape codes from the given string.
func Sanitize(s string) string {
	return ansi.Strip(s)
}

// ValidateUsername returns an error if any of the given usernames are invalid.
func ValidateUsername(username string) error {
	if username == "" {
		return fmt.Errorf("username cannot be empty")
	}

	first, _ := utf8.DecodeRuneInString(username)
	if !unicode.IsLetter(first) {
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
