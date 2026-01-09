package git

import (
	"bufio"
	"bytes"
	"io"
	"io/fs"
	"path/filepath"
	"sort"

	"github.com/aymanbagabas/git-module"
)

// Tree is a wrapper around git.Tree with helper methods.
type Tree struct {
	*git.Tree
	Path       string
	Repository *Repository
}

// TreeEntry is a wrapper around git.TreeEntry with helper methods.
type TreeEntry struct {
	*git.TreeEntry
	// path is the full path of the file
	path string
}

// Entries is a wrapper around git.Entries.
type Entries []*TreeEntry

var sorters = []func(t1, t2 *TreeEntry) bool{
	func(t1, t2 *TreeEntry) bool {
		return (t1.IsTree() || t1.IsCommit()) && !t2.IsTree() && !t2.IsCommit()
	},
	func(t1, t2 *TreeEntry) bool {
		return t1.Name() < t2.Name()
	},
}

// Len implements sort.Interface.
func (es Entries) Len() int { return len(es) }

// Swap implements sort.Interface.
func (es Entries) Swap(i, j int) { es[i], es[j] = es[j], es[i] }

// Less implements sort.Interface.
func (es Entries) Less(i, j int) bool {
	t1, t2 := es[i], es[j]
	var k int
	for k = 0; k < len(sorters)-1; k++ {
		sorter := sorters[k]
		switch {
		case sorter(t1, t2):
			return true
		case sorter(t2, t1):
			return false
		}
	}
	return sorters[k](t1, t2)
}

// Sort sorts the entries in the tree.
func (es Entries) Sort() {
	sort.Sort(es)
}

// File is a wrapper around git.Blob with helper methods.
type File struct {
	*git.Blob
	Entry *TreeEntry
}

// Name returns the name of the file.
func (f *File) Name() string {
	return f.Entry.Name()
}

// Path returns the full path of the file.
func (f *File) Path() string {
	return f.Entry.path
}

// SubTree returns the sub-tree at the given path.
func (t *Tree) SubTree(path string) (*Tree, error) {
	tree, err := t.Subtree(path)
	if err != nil {
		return nil, err
	}
	return &Tree{
		Tree:       tree,
		Path:       path,
		Repository: t.Repository,
	}, nil
}

// Entries returns the entries in the tree.
func (t *Tree) Entries() (Entries, error) {
	entries, err := t.Tree.Entries()
	if err != nil {
		return nil, err
	}
	ret := make(Entries, len(entries))
	for i, e := range entries {
		ret[i] = &TreeEntry{
			TreeEntry: e,
			path:      filepath.Join(t.Path, e.Name()),
		}
	}
	return ret, nil
}

// TreeEntry returns the TreeEntry for the file path.
func (t *Tree) TreeEntry(path string) (*TreeEntry, error) {
	entry, err := t.Tree.TreeEntry(path)
	if err != nil {
		return nil, err
	}
	return &TreeEntry{
		TreeEntry: entry,
		path:      filepath.Join(t.Path, entry.Name()),
	}, nil
}

const sniffLen = 8000

// IsBinary detects if data is a binary value based on:
// http://git.kernel.org/cgit/git/git.git/tree/xdiff-interface.c?id=HEAD#n198
func IsBinary(r io.Reader) (bool, error) {
	reader := bufio.NewReader(r)
	c := 0
	for c < sniffLen {
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
		return fs.FileMode(m) //nolint:gosec
	}
}

// File returns the file for the TreeEntry.
func (e *TreeEntry) File() *File {
	b := e.Blob()
	return &File{
		Blob:  b,
		Entry: e,
	}
}

// Contents returns the contents of the file.
func (e *TreeEntry) Contents() ([]byte, error) {
	return e.File().Contents()
}

// Contents returns the contents of the file.
func (f *File) Contents() ([]byte, error) {
	return f.Blob.Bytes()
}
