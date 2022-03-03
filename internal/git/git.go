package git

import (
	"context"
	"errors"
	"log"
	"os"
	"path/filepath"
	"sort"
	"sync"

	gitypes "github.com/charmbracelet/soft-serve/internal/tui/bubbles/git/types"
	"github.com/go-git/go-billy/v5/memfs"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/storage/memory"
)

// ErrMissingRepo indicates that the requested repository could not be found.
var ErrMissingRepo = errors.New("missing repo")

// Repo represents a Git repository.
type Repo struct {
	path       string
	repository *git.Repository
	Readme     string
	refCommits map[plumbing.Hash]gitypes.Commits
	head       *plumbing.Reference
	refs       []*plumbing.Reference
	trees      map[plumbing.Hash]*object.Tree
	commits    map[plumbing.Hash]*object.Commit
	patch      map[plumbing.Hash]*object.Patch
}

// GetName returns the name of the repository.
func (r *Repo) Name() string {
	return filepath.Base(r.path)
}

// GetHEAD returns the reference for a repository.
func (r *Repo) GetHEAD() *plumbing.Reference {
	return r.head
}

// SetHEAD sets the repository head reference.
func (r *Repo) SetHEAD(ref *plumbing.Reference) error {
	r.head = ref
	return nil
}

// GetReferences returns the references for a repository.
func (r *Repo) GetReferences() []*plumbing.Reference {
	return r.refs
}

// GetRepository returns the underlying go-git repository object.
func (r *Repo) Repository() *git.Repository {
	return r.repository
}

// Tree returns the git tree for a given path.
func (r *Repo) Tree(ref *plumbing.Reference, path string) (*object.Tree, error) {
	path = filepath.Clean(path)
	hash, err := r.targetHash(ref)
	if err != nil {
		return nil, err
	}
	c, err := r.commitForHash(hash)
	if err != nil {
		return nil, err
	}
	t, err := r.treeForHash(c.TreeHash)
	if err != nil {
		return nil, err
	}
	if path == "." {
		return t, nil
	}
	return t.Tree(path)
}

func (r *Repo) treeForHash(treeHash plumbing.Hash) (*object.Tree, error) {
	var err error
	t, ok := r.trees[treeHash]
	if !ok {
		t, err = r.repository.TreeObject(treeHash)
		if err != nil {
			return nil, err
		}
		r.trees[treeHash] = t
	}
	return t, nil
}

func (r *Repo) commitForHash(hash plumbing.Hash) (*object.Commit, error) {
	var err error
	co, ok := r.commits[hash]
	if !ok {
		co, err = r.repository.CommitObject(hash)
		if err != nil {
			return nil, err
		}
		r.commits[hash] = co
	}
	return co, nil
}

func (r *Repo) PatchCtx(ctx context.Context, commit *object.Commit) (*object.Patch, error) {
	hash := commit.Hash
	p, ok := r.patch[hash]
	if !ok {
		c, err := r.commitForHash(hash)
		if err != nil {
			return nil, err
		}
		// Using commit trees fixes the issue when generating diff for the first commit
		// https://github.com/go-git/go-git/issues/281
		tree, err := r.treeForHash(c.TreeHash)
		if err != nil {
			return nil, err
		}
		var parent *object.Commit
		parentTree := &object.Tree{}
		if c.NumParents() > 0 {
			parent, err = r.commitForHash(c.ParentHashes[0])
			if err != nil {
				return nil, err
			}
			parentTree, err = r.treeForHash(parent.TreeHash)
			if err != nil {
				return nil, err
			}
		}
		p, err = parentTree.PatchContext(ctx, tree)
		if err != nil {
			return nil, err
		}
	}
	return p, nil
}

// GetCommits returns the commits for a repository.
func (r *Repo) GetCommits(ref *plumbing.Reference) (gitypes.Commits, error) {
	hash, err := r.targetHash(ref)
	if err != nil {
		return nil, err
	}
	// return cached commits if available
	commits, ok := r.refCommits[hash]
	if ok {
		return commits, nil
	}
	commits = gitypes.Commits{}
	co, err := r.commitForHash(hash)
	if err != nil {
		return nil, err
	}
	// traverse the commit tree to get all commits
	commits = append(commits, co)
	for co.NumParents() > 0 {
		co, err = r.commitForHash(co.ParentHashes[0])
		if err != nil {
			return nil, err
		}
		commits = append(commits, co)
	}
	if err != nil {
		return nil, err
	}
	sort.Sort(commits)
	// cache the commits in the repo
	r.refCommits[hash] = commits
	return commits, nil
}

// targetHash returns the target hash for a given reference. If reference is an
// annotated tag, find the target hash for that tag.
func (r *Repo) targetHash(ref *plumbing.Reference) (plumbing.Hash, error) {
	hash := ref.Hash()
	if ref.Type() != plumbing.HashReference {
		return plumbing.ZeroHash, plumbing.ErrInvalidType
	}
	if ref.Name().IsTag() {
		to, err := r.repository.TagObject(hash)
		switch err {
		case nil:
			// annotated tag (object has a target hash)
			hash = to.Target
		case plumbing.ErrObjectNotFound:
			// lightweight tag (hash points to a commit)
		default:
			return plumbing.ZeroHash, err
		}
	}
	return hash, nil
}

// GetReadme returns the readme for a repository.
func (r *Repo) GetReadme() string {
	if r.Readme != "" {
		return r.Readme
	}
	md, err := r.LatestFile("README.md")
	if err != nil {
		return ""
	}
	return md
}

// RepoSource is a reference to an on-disk repositories.
type RepoSource struct {
	Path  string
	mtx   sync.Mutex
	repos []*Repo
}

// NewRepoSource creates a new RepoSource.
func NewRepoSource(repoPath string) *RepoSource {
	err := os.MkdirAll(repoPath, os.ModeDir|os.FileMode(0700))
	if err != nil {
		log.Fatal(err)
	}
	rs := &RepoSource{Path: repoPath}
	return rs
}

// AllRepos returns all repositories for the given RepoSource.
func (rs *RepoSource) AllRepos() []*Repo {
	rs.mtx.Lock()
	defer rs.mtx.Unlock()
	return rs.repos
}

// GetRepo returns a repository by name.
func (rs *RepoSource) GetRepo(name string) (*Repo, error) {
	rs.mtx.Lock()
	defer rs.mtx.Unlock()
	for _, r := range rs.repos {
		if filepath.Base(r.path) == name {
			return r, nil
		}
	}
	return nil, ErrMissingRepo
}

// InitRepo initializes a new Git repository.
func (rs *RepoSource) InitRepo(name string, bare bool) (*Repo, error) {
	rs.mtx.Lock()
	defer rs.mtx.Unlock()
	rp := filepath.Join(rs.Path, name)
	rg, err := git.PlainInit(rp, bare)
	if err != nil {
		return nil, err
	}
	if bare {
		// Clone repo into memory storage
		ar, err := git.Clone(memory.NewStorage(), memfs.New(), &git.CloneOptions{
			URL: rp,
		})
		if err != nil && err != transport.ErrEmptyRemoteRepository {
			return nil, err
		}
		rg = ar
	}
	r := &Repo{
		path:       rp,
		repository: rg,
	}
	rs.repos = append(rs.repos, r)
	return r, nil
}

// LoadRepos opens Git repositories.
func (rs *RepoSource) LoadRepos() error {
	rs.mtx.Lock()
	defer rs.mtx.Unlock()
	rd, err := os.ReadDir(rs.Path)
	if err != nil {
		return err
	}
	rs.repos = make([]*Repo, 0)
	for _, de := range rd {
		rp := filepath.Join(rs.Path, de.Name())
		rg, err := git.PlainOpen(rp)
		if err != nil {
			return err
		}
		r, err := rs.loadRepo(rp, rg)
		if err != nil {
			return err
		}
		rs.repos = append(rs.repos, r)
	}
	return nil
}

func (rs *RepoSource) loadRepo(path string, rg *git.Repository) (*Repo, error) {
	r := &Repo{
		path:       path,
		repository: rg,
		patch:      make(map[plumbing.Hash]*object.Patch),
	}
	r.commits = make(map[plumbing.Hash]*object.Commit)
	r.trees = make(map[plumbing.Hash]*object.Tree)
	r.refCommits = make(map[plumbing.Hash]gitypes.Commits)
	ref, err := rg.Head()
	if err != nil {
		return nil, err
	}
	r.head = ref
	rm, err := r.LatestFile("README.md")
	if err == object.ErrFileNotFound {
		rm = ""
	} else if err != nil {
		return nil, err
	}
	r.Readme = rm
	l, err := r.repository.Log(&git.LogOptions{All: true})
	if err != nil {
		return nil, err
	}
	err = l.ForEach(func(c *object.Commit) error {
		r.commits[c.Hash] = c
		return nil
	})
	if err != nil {
		return nil, err
	}
	refs := make([]*plumbing.Reference, 0)
	ri, err := rg.References()
	if err != nil {
		return nil, err
	}
	ri.ForEach(func(r *plumbing.Reference) error {
		refs = append(refs, r)
		return nil
	})
	r.refs = refs
	return r, nil
}

// LatestFile returns the latest file at the specified path in the repository.
func (r *Repo) LatestFile(path string) (string, error) {
	c, err := r.commitForHash(r.head.Hash())
	if err != nil {
		return "", err
	}
	f, err := c.File(path)
	if err != nil {
		return "", err
	}
	content, err := f.Contents()
	if err != nil {
		return "", err
	}
	return content, nil
}

// LatestTree returns the latest tree at the specified path in the repository.
func (r *Repo) LatestTree(path string) (*object.Tree, error) {
	return r.Tree(r.head, path)
}

