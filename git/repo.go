package git

import (
	"path/filepath"
	"strings"

	"github.com/gogs/git-module"
)

var (
	// DiffMaxFile is the maximum number of files to show in a diff.
	DiffMaxFiles = 1000
	// DiffMaxFileLines is the maximum number of lines to show in a file diff.
	DiffMaxFileLines = 1000
	// DiffMaxLineChars is the maximum number of characters to show in a line diff.
	DiffMaxLineChars = 1000
)

// Repository is a wrapper around git.Repository with helper methods.
type Repository struct {
	*git.Repository
	Path   string
	IsBare bool
}

// Clone clones a repository.
func Clone(src, dst string, opts ...git.CloneOptions) error {
	return git.Clone(src, dst, opts...)
}

// Init initializes and opens a new git repository.
func Init(path string, bare bool) (*Repository, error) {
	if bare {
		path = strings.TrimSuffix(path, ".git") + ".git"
	}

	err := git.Init(path, git.InitOptions{Bare: bare})
	if err != nil {
		return nil, err
	}
	return Open(path)
}

func isInsideWorkTree(r *git.Repository) bool {
	out, err := r.RevParse("--is-inside-work-tree")
	return err == nil && out == "true"
}

func isInsideGitDir(r *git.Repository) bool {
	out, err := r.RevParse("--is-inside-git-dir")
	return err == nil && out == "true"
}

func gitDir(r *git.Repository) (string, error) {
	return r.RevParse("--git-dir")
}

// Open opens a git repository at the given path.
func Open(path string) (*Repository, error) {
	repo, err := git.Open(path)
	if err != nil {
		return nil, err
	}
	gp, err := gitDir(repo)
	if err != nil || (gp != "." && gp != ".git") {
		return nil, ErrNotAGitRepository
	}
	return &Repository{
		Repository: repo,
		Path:       path,
		IsBare:     gp == ".",
	}, nil
}

// Name returns the name of the repository.
func (r *Repository) Name() string {
	return filepath.Base(r.Path)
}

// HEAD returns the HEAD reference for a repository.
func (r *Repository) HEAD() (*Reference, error) {
	rn, err := r.Repository.SymbolicRef(git.SymbolicRefOptions{Name: "HEAD"})
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

// LsTree returns the tree for the given reference.
func (r *Repository) LsTree(ref string) (*Tree, error) {
	tree, err := r.Repository.LsTree(ref)
	if err != nil {
		return nil, err
	}
	return &Tree{
		Tree:       tree,
		Path:       "",
		Repository: r,
	}, nil
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
	return r.LsTree(ref.Hash.String())
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

// Config returns the config value for the given key.
func (r *Repository) Config(key string, opts ...ConfigOptions) (string, error) {
	dir, err := gitDir(r.Repository)
	if err != nil {
		return "", err
	}
	var opt ConfigOptions
	if len(opts) > 0 {
		opt = opts[0]
	}
	opt.File = filepath.Join(dir, "config")
	return Config(key, opt)
}

// SetConfig sets the config value for the given key.
func (r *Repository) SetConfig(key, value string, opts ...ConfigOptions) error {
	dir, err := gitDir(r.Repository)
	if err != nil {
		return err
	}
	var opt ConfigOptions
	if len(opts) > 0 {
		opt = opts[0]
	}
	opt.File = filepath.Join(dir, "config")
	return SetConfig(key, value, opt)
}

// SymbolicRef returns or updates the symbolic reference for the given name.
// Both name and ref can be empty.
func (r *Repository) SymbolicRef(name string, ref string) (string, error) {
	opt := git.SymbolicRefOptions{
		Name: name,
		Ref:  ref,
	}
	return r.Repository.SymbolicRef(opt)
}
