package backend

import (
	"path/filepath"

	"github.com/charmbracelet/soft-serve/git"
	"github.com/gobwas/glob"
)

// LatestFile returns the contents of the latest file at the specified path in
// the repository and its file path.
func LatestFile(r Repository, pattern string) (string, string, error) {
	g := glob.MustCompile(pattern)
	dir := filepath.Dir(pattern)
	repo, err := r.Open()
	if err != nil {
		return "", "", err
	}
	head, err := repo.HEAD()
	if err != nil {
		return "", "", err
	}
	t, err := repo.TreePath(head, dir)
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

// Readme returns the repository's README.
func Readme(r Repository) (readme string, path string, err error) {
	readme, path, err = LatestFile(r, "README*")
	return
}
