package utils

import (
	"fmt"
	"path"
	"strings"
	"unicode"

	"github.com/charmbracelet/x/ansi"
)

// SanitizeRepo returns the canonical repository name; the result is a fixed point.
func SanitizeRepo(repo string) string {
	// We're using path instead of filepath here because this is not OS dependent
	// looking at you Windows
	segs := strings.Split(path.Clean("/" + Sanitize(repo))[1:], "/")
	for len(segs) > 0 {
		last := segs[len(segs)-1]
		for strings.HasSuffix(last, ".git") {
			last = strings.TrimSuffix(last, ".git")
		}
		switch last {
		case "", ".":
			segs = segs[:len(segs)-1]
		case "..":
			segs = segs[:max(len(segs)-2, 0)]
		default:
			segs[len(segs)-1] = last
			return strings.Join(segs, "/")
		}
	}
	return ""
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

// ValidateRepo rejects names that are not canonical or whose segments end in
// ".git", which would nest one repository inside another.
func ValidateRepo(repo string) error {
	if repo == "" {
		return fmt.Errorf("repo cannot be empty")
	}

	for _, r := range repo {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '-' && r != '_' && r != '.' && r != '/' {
			return fmt.Errorf("repo can only contain letters, numbers, hyphens, underscores, periods, and slashes")
		}
	}

	if SanitizeRepo(repo) != repo {
		return fmt.Errorf("repo name is not in its canonical form")
	}

	for _, segment := range strings.Split(repo, "/") {
		if strings.HasSuffix(strings.ToLower(segment), ".git") {
			return fmt.Errorf("repo path segments cannot end in .git")
		}
	}

	return nil
}
