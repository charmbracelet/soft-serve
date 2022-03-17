package types

import (
	"bytes"
	"fmt"
	"io/fs"
	"io/ioutil"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/filemode"
	"github.com/go-git/go-git/v5/utils/binary"
)

const (
	DirMode fs.FileMode = 040000
)

type Repo interface {
	Name() string
	GetHEAD() *plumbing.Reference
	SetHEAD(*plumbing.Reference) error
	GetReferences() []*plumbing.Reference
	GetReadme() string
	GetReadmePath() string
	Count(*plumbing.Reference) (int, error)
	GetCommits(*plumbing.Reference) (Commits, error)
	GetCommitsLimit(*plumbing.Reference, int) (Commits, error)
	GetCommitsSkip(*plumbing.Reference, int, int) (Commits, error)
	Repository() *git.Repository
	Tree(*plumbing.Reference, string) (*Tree, error)
	TreeEntryFile(*TreeEntry) (*File, error)
	Patch(*Commit) (*Patch, error)
}

type Commits []*Commit

func (cl Commits) Len() int      { return len(cl) }
func (cl Commits) Swap(i, j int) { cl[i], cl[j] = cl[j], cl[i] }
func (cl Commits) Less(i, j int) bool {
	return cl[i].Author.When.After(cl[j].Author.When)
}

type Signature struct {
	Name  string
	Email string
	When  time.Time
}

func (s Signature) String() string {
	return fmt.Sprintf("%s <%s>", s.Name, s.Email)
}

type CommitMessage string

func (cm CommitMessage) Title() string {
	lines := strings.Split(cm.String(), "\n")
	if len(lines) > 0 {
		return lines[0]
	}
	return ""
}

func (cm CommitMessage) String() string {
	return string(cm)
}

type Commit struct {
	Hash      string
	TreeHash  string
	Message   CommitMessage
	Author    Signature
	Committer Signature
	// TODO implement PGP signature
	// TODO implement mergetag

	// SHAs of parent commits
	Parents []string
}

type Patch struct {
	Stat string
	Diff string
}

type TreeEntry struct {
	EntryName    string
	EntryMode    fs.FileMode
	EntrySize    int64
	EntryModTime time.Time
	EntryHash    string
	EntryType    string
}

func (e *TreeEntry) Name() string {
	return filepath.Base(e.EntryName)
}

func (e *TreeEntry) Mode() fs.FileMode {
	m := fs.FileMode(e.EntryMode)
	if m&fs.FileMode(filemode.Dir) != 0 {
		m |= fs.ModeDir | fs.ModePerm
	}
	return m
}

func (e *TreeEntry) IsDir() bool {
	return e.Mode().IsDir()
}

func (e *TreeEntry) Size() int64 {
	return e.EntrySize
}

func (e *TreeEntry) ModTime() time.Time {
	return e.EntryModTime
}

func (e *TreeEntry) Sys() interface{} {
	return e.Hash()
}

func (e *TreeEntry) Hash() plumbing.Hash {
	return plumbing.NewHash(e.EntryHash)
}

func (e *TreeEntry) Type() string {
	return e.EntryType
}

func (e *TreeEntry) Path() string {
	return e.EntryName
}

type Tree struct {
	Entries TreeEntries
	Hash    string
}

type TreeEntries []*TreeEntry

func (cl TreeEntries) Len() int      { return len(cl) }
func (cl TreeEntries) Swap(i, j int) { cl[i], cl[j] = cl[j], cl[i] }
func (cl TreeEntries) Less(i, j int) bool {
	if cl[i].IsDir() && cl[j].IsDir() {
		return cl[i].Name() < cl[j].Name()
	} else if cl[i].IsDir() {
		return true
	} else if cl[j].IsDir() {
		return false
	} else {
		return cl[i].Name() < cl[j].Name()
	}
}

type File struct {
	Entry *TreeEntry
	Blob  []byte

	fs.File
}

func (f *File) Read(p []byte) (int, error) {
	reader := bytes.NewReader(f.Blob)
	return reader.Read(p)
}

func (f *File) Stats() (fs.FileInfo, error) {
	return f.Entry, nil
}

func (f *File) Close() error {
	return nil
}

func (f *File) IsBinary() (bool, error) {
	return binary.IsBinary(f)
}

func (f *File) Contents() (string, error) {
	reader := bytes.NewReader(f.Blob)
	contents, err := ioutil.ReadAll(reader)
	if err != nil {
		return "", err
	}
	return string(contents), nil
}
