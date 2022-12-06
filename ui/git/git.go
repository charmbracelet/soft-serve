package git

import (
	"errors"
	"fmt"

	"github.com/charmbracelet/soft-serve/proto"
)

// ErrMissingRepo indicates that the requested repository could not be found.
var ErrMissingRepo = errors.New("missing repo")

// Repository is a Git repository with its metadata.
type Repository struct {
	Repo proto.Repository
	Info proto.Metadata
}

// Readme returns the repository's README.
func (r *Repository) Readme() (readme string, path string) {
	readme, path, _ = r.LatestFile("README*")
	return
}

// LatestFile returns the contents of the latest file at the specified path in
// the repository and its file path.
func (r *Repository) LatestFile(pattern string) (string, string, error) {
	return proto.LatestFile(r.Repo, pattern)
}

// RepoURL returns the URL of the repository.
func RepoURL(host string, port int, name string) string {
	p := ""
	if port != 22 {
		p += fmt.Sprintf(":%d", port)
	}
	return fmt.Sprintf("git clone ssh://%s/%s", host+p, name)
}
