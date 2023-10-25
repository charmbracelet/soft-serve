package git

import (
	"os"
	"path/filepath"

	"github.com/gobwas/glob"
)

// LatestFile returns the contents of the first file at the specified path pattern in the repository and its file path.
func LatestFile(repo *Repository, ref *Reference, pattern string) (string, string, error) {
	g := glob.MustCompile(pattern)
	dir := filepath.Dir(pattern)
	if ref == nil {
		head, err := repo.HEAD()
		if err != nil {
			return "", "", err
		}
		ref = head
	}
	t, err := repo.TreePath(ref, dir)
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

// Returns true if path is a directory containing an `objects` directory and a
// `HEAD` file.
func isGitDir(path string) bool {
	stat, err := os.Stat(filepath.Join(path, "objects"))
	if err != nil {
		return false
	}
	if !stat.IsDir() {
		return false
	}

	stat, err = os.Stat(filepath.Join(path, "HEAD"))
	if err != nil {
		return false
	}
	if stat.IsDir() {
		return false
	}

	return true
}
