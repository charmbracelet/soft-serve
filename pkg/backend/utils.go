package backend

import (
	"fmt"

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

// Readme returns the repository's README.
func Readme(r proto.Repository, ref *git.Reference) (readme string, path string, err error) {
	pattern := "[rR][eE][aA][dD][mM][eE]*"
	directories := []string{"", "docs", ".github", ".gitlab"}
	for _, dir := range directories {
		pattern := fmt.Sprintf("%s/%s", dir, pattern)
		readme, path, err = LatestFile(r, ref, pattern)
		if err == nil {
			break
		}
	}
	return
}
