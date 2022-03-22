package git

import (
	"path/filepath"

	"github.com/gogs/git-module"
)

var (
	DiffMaxFiles     = 1000
	DiffMaxFileLines = 1000
	DiffMaxLineChars = 1000
)

// Repository is a wrapper around git.Repository with helper methods.
type Repository struct {
	*git.Repository
	Path string
}

// Clone clones a repository.
func Clone(src, dst string, opts ...git.CloneOptions) error {
	return git.Clone(src, dst, opts...)
}

// Init initializes and opens a new git repository.
func Init(path string, bare bool) (*Repository, error) {
	err := git.Init(path, git.InitOptions{Bare: bare})
	if err != nil {
		return nil, err
	}
	return Open(path)
}

// Open opens a git repository at the given path.
func Open(path string) (*Repository, error) {
	repo, err := git.Open(path)
	if err != nil {
		return nil, err
	}
	return &Repository{
		Repository: repo,
		Path:       path,
	}, nil
}

// Name returns the name of the repository.
func (r *Repository) Name() string {
	return filepath.Base(r.Path)
}

// HEAD returns the HEAD reference for a repository.
func (r *Repository) HEAD() (*Reference, error) {
	rn, err := r.SymbolicRef()
	if err != nil {
		return nil, err
	}
	hash, err := r.ShowRefVerify(rn)
	if err != nil {
		return nil, err
	}
	return &Reference{
		Reference: &git.Reference{
			ID:      hash,
			Refspec: rn,
		},
		Hash: Hash(hash),
		path: r.Path,
	}, nil
}

// References returns the references for a repository.
func (r *Repository) References() ([]*Reference, error) {
	refs, err := r.ShowRef()
	if err != nil {
		return nil, err
	}
	rrefs := make([]*Reference, 0, len(refs))
	for _, ref := range refs {
		rrefs = append(rrefs, &Reference{
			Reference: ref,
			Hash:      Hash(ref.ID),
			path:      r.Path,
		})
	}
	return rrefs, nil
}

// Tree returns the tree for the given reference.
func (r *Repository) Tree(ref *Reference) (*Tree, error) {
	if ref == nil {
		rref, err := r.HEAD()
		if err != nil {
			return nil, err
		}
		ref = rref
	}
	tree, err := r.LsTree(ref.Hash.String())
	if err != nil {
		return nil, err
	}
	return &Tree{
		Tree: tree,
		Path: "",
	}, nil
}

// TreePath returns the tree for the given path.
func (r *Repository) TreePath(ref *Reference, path string) (*Tree, error) {
	path = filepath.Clean(path)
	if path == "." {
		path = ""
	}
	if path == "" {
		return r.Tree(ref)
	}
	t, err := r.Tree(ref)
	if err != nil {
		return nil, err
	}
	return t.SubTree(path)
}

// Diff returns the diff for the given commit.
func (r *Repository) Diff(commit *Commit) (*Diff, error) {
	ddiff, err := r.Repository.Diff(commit.Hash.String(), DiffMaxFiles, DiffMaxFileLines, DiffMaxLineChars)
	if err != nil {
		return nil, err
	}
	files := make([]*DiffFile, 0, len(ddiff.Files))
	for _, df := range ddiff.Files {
		sections := make([]*DiffSection, 0, len(df.Sections))
		for _, ds := range df.Sections {
			sections = append(sections, &DiffSection{
				DiffSection: ds,
			})
		}
		files = append(files, &DiffFile{
			DiffFile: df,
			Sections: sections,
		})
	}
	diff := &Diff{
		Diff:  ddiff,
		Files: files,
	}
	return diff, nil
}

// Patch returns the patch for the given reference.
func (r *Repository) Patch(commit *Commit) (string, error) {
	diff, err := r.Diff(commit)
	if err != nil {
		return "", err
	}
	return diff.Patch(), err
}

// CountCommits returns the number of commits in the repository.
func (r *Repository) CountCommits(ref *Reference) (int64, error) {
	return r.Repository.RevListCount([]string{ref.Name().String()})
}

// CommitsByPage returns the commits for a given page and size.
func (r *Repository) CommitsByPage(ref *Reference, page, size int) (Commits, error) {
	cs, err := r.Repository.CommitsByPage(ref.Name().String(), page, size)
	if err != nil {
		return nil, err
	}
	commits := make(Commits, len(cs))
	for i, c := range cs {
		commits[i] = &Commit{
			Commit: c,
			Hash:   Hash(c.ID.String()),
		}
	}
	return commits, nil
}

// UpdateServerInfo updates the repository server info.
func (r *Repository) UpdateServerInfo() error {
	cmd := git.NewCommand("update-server-info")
	_, err := cmd.RunInDir(r.Path)
	return err
}
