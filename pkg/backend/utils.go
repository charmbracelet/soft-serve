package backend

import (
	"errors"

	"github.com/charmbracelet/soft-serve/git"
	"github.com/charmbracelet/soft-serve/pkg/proto"
)

// LatestFile returns the contents of the latest file at the specified path in
// the repository and its file path.
func LatestFile(r proto.Repository, ref *git.Reference, pattern string) (string, string, error) {
	repo, err := r.Open()
	if err != nil {
		return "", "", err
	}
	return git.LatestFile(repo, ref, pattern)
}

// readmePatterns is the ordered list of glob patterns used to find a README.
// Root-level patterns are checked first; subdirectory paths are fallbacks,
// matching GitHub's README discovery behavior.
var readmePatterns = []string{
	"[rR][eE][aA][dD][mM][eE]*",
	"docs/[rR][eE][aA][dD][mM][eE]*",
	".github/[rR][eE][aA][dD][mM][eE]*",
	".gitlab/[rR][eE][aA][dD][mM][eE]*",
}

// Readme returns the repository's README.
// It checks the repository root first, then falls back to docs/, .github/,
// and .gitlab/ subdirectories.
// When no README is found in any location, it returns ("", "", nil).
// Callers should check whether path is empty to detect a missing README.
func Readme(r proto.Repository, ref *git.Reference) (readme string, path string, err error) {
	for _, pattern := range readmePatterns {
		readme, path, err = LatestFile(r, ref, pattern)
		if err == nil {
			return
		}
		if !errors.Is(err, git.ErrFileNotFound) && !errors.Is(err, git.ErrRevisionNotExist) {
			return
		}
	}
	// No README found in any location; return a clean sentinel rather than
	// leaking the error from the last pattern tried.
	return "", "", nil
}
