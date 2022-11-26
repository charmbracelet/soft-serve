package file

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/charmbracelet/soft-serve/git"
	"github.com/charmbracelet/soft-serve/proto"
)

type key int

const (
	projectName key = iota
	description
	private
)

var keys = map[key]string{
	projectName: "soft-serve.projectName",
	description: "soft-serve.description",
	private:     "soft-serve.private",
}

var _ proto.Provider = &File{}

// File is a file-based repository provider.
type File struct {
	repoPath string
}

// New returns a new File provider.
func New(repoPath string) *File {
	f := &File{
		repoPath: repoPath,
	}
	return f
}

// Open opens a new repository and returns a new FileRepo.
func (f *File) Open(name string) (proto.RepositoryService, error) {
	fp := filepath.Join(f.repoPath, name)
	r, err := git.Open(fp)
	if errors.Is(err, os.ErrNotExist) {
		r, err = git.Open(fp + ".git")
	}
	if err != nil {
		return nil, err
	}
	return &FileRepo{r}, nil
}

var _ proto.Repository = &FileRepo{}

// FileRepo is a file-based repository.
type FileRepo struct { // nolint:revive
	repo *git.Repository
}

// Name returns the name of the repository.
func (r *FileRepo) Name() string {
	return strings.TrimSuffix(r.repo.Name(), ".git")
}

// ProjectName returns the project name of the repository.
func (r *FileRepo) ProjectName() string {
	pn, err := r.repo.Config(keys[projectName])
	if err != nil {
		return ""
	}
	return pn
}

// SetProjectName sets the project name of the repository.
func (r *FileRepo) SetProjectName(name string) error {
	return r.repo.SetConfig(keys[projectName], name)
}

// Description returns the description of the repository.
func (r *FileRepo) Description() string {
	desc, err := r.repo.Config(keys[description])
	if err != nil {
		return ""
	}
	return desc
}

// SetDescription sets the description of the repository.
func (r *FileRepo) SetDescription(desc string) error {
	return r.repo.SetConfig(keys[description], desc)
}

// IsPrivate returns whether the repository is private.
func (r *FileRepo) IsPrivate() bool {
	p, err := r.repo.Config(keys[private])
	if err != nil {
		return false
	}
	return p == "true"
}

// SetPrivate sets whether the repository is private.
func (r *FileRepo) SetPrivate(p bool) error {
	return r.repo.SetConfig(keys[private], strconv.FormatBool(p))
}
