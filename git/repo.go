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
	return r.LsTree(ref.ID)
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
	diff, err := r.Repository.Diff(commit.ID.String(), DiffMaxFiles, DiffMaxFileLines, DiffMaxLineChars, git.DiffOptions{
		CommandOptions: git.CommandOptions{
			Envs: []string{"GIT_CONFIG_GLOBAL=/dev/null"},
		},
	})
	if err != nil {
		return nil, err
	}
	return toDiff(diff), nil
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
	return r.RevListCount([]string{ref.Name().String()})
}

// CommitsByPage returns the commits for a given page and size.
func (r *Repository) CommitsByPage(ref *Reference, page, size int) (Commits, error) {
	cs, err := r.Repository.CommitsByPage(ref.Name().String(), page, size)
	if err != nil {
		return nil, err
	}
	commits := make(Commits, len(cs))
	copy(commits, cs)
	return commits, nil
}

// SymbolicRef returns or updates the symbolic reference for the given name.
// Both name and ref can be empty.
func (r *Repository) SymbolicRef(name string, ref string, opts ...git.SymbolicRefOptions) (string, error) {
	var opt git.SymbolicRefOptions
	if len(opts) > 0 {
		opt = opts[0]
	}

	opt.Name = name
	opt.Ref = ref
	return r.Repository.SymbolicRef(opt)
}
