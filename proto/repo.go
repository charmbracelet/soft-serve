package proto

import (
	"path/filepath"

	"github.com/charmbracelet/soft-serve/git"
	"github.com/gobwas/glob"
)

// Metadata is a repository's metadata.
type Metadata interface {
	Name() string
	ProjectName() string
	Description() string
	IsPrivate() bool
	Collabs() []User
	Open() (Repository, error)
}

// Repository is Git repository.
type Repository interface {
	Name() string
	Repository() *git.Repository
}

// LatestFile returns the contents of the latest file at the specified path in
// the repository and its file path.
func LatestFile(r Repository, pattern string) (string, string, error) {
	g := glob.MustCompile(pattern)
	dir := filepath.Dir(pattern)
	head, err := r.Repository().HEAD()
	if err != nil {
		return "", "", err
	}
	t, err := r.Repository().TreePath(head, dir)
	if err != nil {
		return "", "", err
	}
	ents, err := t.Entries()
	if err != nil {
		return "", "", err
	}
	for _, e := range ents {
		fp := filepath.Join(dir, e.Name())
		if e.IsTree() {
			continue
		}
		if g.Match(fp) {
			bts, err := e.Contents()
			if err != nil {
				return "", "", err
			}
			return string(bts), fp, nil
		}
	}
	return "", "", git.ErrFileNotFound
}
