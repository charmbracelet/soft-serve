package git

import (
	"path/filepath"

	"github.com/gobwas/glob"
)

// LatestFile returns the contents of the first file at the specified path pattern in the repository and its file path.
func LatestFile(repo *Repository, pattern string) (string, string, error) {
	g := glob.MustCompile(pattern)
	dir := filepath.Dir(pattern)
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
		te := e
		fp := filepath.Join(dir, te.Name())
		if te.IsTree() {
			continue
		}
		if g.Match(fp) {
			if te.IsSymlink() {
				bts, err := te.Contents()
				if err != nil {
					return "", "", err
				}
				fp = string(bts)
				te, err = t.TreeEntry(fp)
				if err != nil {
					return "", "", err
				}
			}
			bts, err := te.Contents()
			if err != nil {
				return "", "", err
			}
			return string(bts), fp, nil
		}
	}
	return "", "", ErrFileNotFound
}
