package git

import (
	"bufio"
	"bytes"
	"io"
	"io/fs"
	"path/filepath"

	"github.com/gogs/git-module"
)

// Tree is a wrapper around git.Tree with helper methods.
type Tree struct {
	*git.Tree
	Path string
}

// TreeEntry is a wrapper around git.TreeEntry with helper methods.
type TreeEntry struct {
	*git.TreeEntry
}

// File is a wrapper around git.Blob with helper methods.
type File struct {
	*git.Blob
	// path is the full path of the file
	path  string
	Entry *TreeEntry
}

// File returns the file at the given path.
func (t *Tree) File(path string) (*File, error) {
	b, err := t.Blob(path)
	if err != nil {
		return nil, err
	}
	return &File{
		Blob:  b,
		Entry: &TreeEntry{TreeEntry: b.TreeEntry},
		path:  filepath.Join(t.Path, path),
	}, nil
}

// Name returns the name of the file.
func (f *File) Name() string {
	return f.Entry.Name()
}

// Path returns the full path of the file.
func (f *File) Path() string {
	return f.path
}

// SubTree returns the sub-tree at the given path.
func (t *Tree) SubTree(path string) (*Tree, error) {
	tree, err := t.Subtree(path)
	if err != nil {
		return nil, err
	}
	return &Tree{
		Tree: tree,
		Path: path,
	}, nil
}

const sniffLen = 8000

// IsBinary detects if data is a binary value based on:
// http://git.kernel.org/cgit/git/git.git/tree/xdiff-interface.c?id=HEAD#n198
func IsBinary(r io.Reader) (bool, error) {
	reader := bufio.NewReader(r)
	c := 0
	for {
		if c == sniffLen {
			break
		}

		b, err := reader.ReadByte()
		if err == io.EOF {
			break
		}
		if err != nil {
			return false, err
		}

		if b == byte(0) {
			return true, nil
		}

		c++
	}

	return false, nil
}

// IsBinary returns true if the file is binary.
func (f *File) IsBinary() (bool, error) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	err := f.Pipeline(stdout, stderr)
	if err != nil {
		return false, err
	}
	r := bufio.NewReader(stdout)
	return IsBinary(r)
}

// Mode returns the mode of the file in fs.FileMode format.
func (e *TreeEntry) Mode() fs.FileMode {
	m := e.Blob().Mode()
	switch m {
	case git.EntryTree:
		return fs.ModeDir | fs.ModePerm
	default:
		return fs.FileMode(m)
	}
}
