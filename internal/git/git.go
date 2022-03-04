package git

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"sync"

	gitypes "github.com/charmbracelet/soft-serve/internal/tui/bubbles/git/types"
	"github.com/go-git/go-billy/v5/memfs"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/storer"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/storage/memory"
	"github.com/gobwas/glob"
	"github.com/golang/groupcache/lru"
)

// ErrMissingRepo indicates that the requested repository could not be found.
var ErrMissingRepo = errors.New("missing repo")

// Repo represents a Git repository.
type Repo struct {
	path       string
	repository *git.Repository
	Readme     string
	ReadmePath string
	head       *plumbing.Reference
	refs       []*plumbing.Reference
	trees      *lru.Cache
	commits    *lru.Cache
	patch      *lru.Cache
	mtx        sync.Mutex
	pmtx       sync.Mutex
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
	r.mtx.Lock()
	defer r.mtx.Unlock()
	var err error
	if r.trees == nil {
		t, err := r.repository.TreeObject(treeHash)
		if err != nil {
			return nil, err
		}
		return t, nil
	}
	var t *object.Tree
	i, ok := r.trees.Get(treeHash)
	if !ok {
		t, err = r.repository.TreeObject(treeHash)
		if err != nil {
			return nil, err
		}
		r.trees.Add(treeHash, t)
	} else {
		t, ok = i.(*object.Tree)
		if !ok {
			return nil, fmt.Errorf("error casting interface to tree")
		}
	}
	return t, nil
}

func (r *Repo) commitForHash(hash plumbing.Hash) (*object.Commit, error) {
	r.mtx.Lock()
	defer r.mtx.Unlock()
	var err error
	if r.commits == nil {
		co, err := r.repository.CommitObject(hash)
		if err != nil {
			return nil, err
		}
		return co, nil
	}
	var c *object.Commit
	i, ok := r.commits.Get(hash)
	if !ok {
		c, err = r.repository.CommitObject(hash)
		if err != nil {
			return nil, err
		}
		r.commits.Add(hash, c)
	} else {
		c, ok = i.(*object.Commit)
		if !ok {
			return nil, fmt.Errorf("error casting interface to commit")
		}
	}
	return c, nil
}

func (r *Repo) patchForHashCtx(ctx context.Context, hash plumbing.Hash) (*object.Patch, error) {
	r.pmtx.Lock()
	defer r.pmtx.Unlock()
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
	p, err := parentTree.PatchContext(ctx, tree)
	if err != nil {
		return nil, err
	}
	return p, nil
}

// PatchCtx returns the patch for a given commit.
func (r *Repo) PatchCtx(ctx context.Context, commit *object.Commit) (*object.Patch, error) {
	var err error
	hash := commit.Hash
	if r.patch == nil {
		return r.patchForHashCtx(ctx, hash)
	}
	var p *object.Patch
	i, ok := r.patch.Get(hash)
	if !ok {
		p, err = r.patchForHashCtx(ctx, hash)
		if err != nil {
			return nil, err
		}
		r.patch.Add(hash, p)
	} else {
		p, ok = i.(*object.Patch)
		if !ok {
			return nil, fmt.Errorf("error casting interface to patch")
		}
	}
	return p, nil
}

// GetCommits returns the commits for a repository.
func (r *Repo) GetCommits(ref *plumbing.Reference) (gitypes.Commits, error) {
	var err error
	hash, err := r.targetHash(ref)
	if err != nil {
		return nil, err
	}
	commits := gitypes.Commits{}
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
	return r.Readme
}

// GetReadmePath returns the path to the readme for a repository.
func (r *Repo) GetReadmePath() string {
	return r.ReadmePath
}

// RepoSource is a reference to an on-disk repositories.
type RepoSource struct {
	Path      string
	CacheSize int
	mtx       sync.Mutex
	repos     map[string]*Repo
}

// NewRepoSource creates a new RepoSource.
func NewRepoSource(repoPath string) *RepoSource {
	err := os.MkdirAll(repoPath, os.ModeDir|os.FileMode(0700))
	if err != nil {
		log.Fatal(err)
	}
	rs := &RepoSource{Path: repoPath}
	rs.repos = make(map[string]*Repo, 0)
	return rs
}

// AllRepos returns all repositories for the given RepoSource.
func (rs *RepoSource) AllRepos() []*Repo {
	rs.mtx.Lock()
	defer rs.mtx.Unlock()
	repos := make([]*Repo, 0, len(rs.repos))
	for _, r := range rs.repos {
		repos = append(repos, r)
	}
	return repos
}

// GetRepo returns a repository by name.
func (rs *RepoSource) GetRepo(name string) (*Repo, error) {
	rs.mtx.Lock()
	defer rs.mtx.Unlock()
	r, ok := rs.repos[name]
	if !ok {
		return nil, ErrMissingRepo
	}
	return r, nil
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
	rs.repos[name] = r
	return r, nil
}

// LoadRepo loads a repository from disk.
func (rs *RepoSource) LoadRepo(name string) error {
	rs.mtx.Lock()
	defer rs.mtx.Unlock()
	rp := filepath.Join(rs.Path, name)
	rg, err := git.PlainOpen(rp)
	if err != nil {
		return err
	}
	r, err := rs.loadRepo(rp, rg)
	if err != nil {
		return err
	}
	rs.repos[name] = r
	return nil
}

// LoadRepos opens Git repositories.
func (rs *RepoSource) LoadRepos() error {
	rd, err := os.ReadDir(rs.Path)
	if err != nil {
		return err
	}
	for _, de := range rd {
		err = rs.LoadRepo(de.Name())
		if err != nil {
			return err
		}
	}
	return nil
}

func (rs *RepoSource) loadRepo(path string, rg *git.Repository) (*Repo, error) {
	r := &Repo{
		path:       path,
		repository: rg,
	}
	if rs.CacheSize > 0 {
		r.commits = lru.New(rs.CacheSize)
		r.trees = lru.New(rs.CacheSize)
		r.patch = lru.New(rs.CacheSize)
	}
	ref, err := rg.Head()
	if err != nil {
		return nil, err
	}
	r.head = ref
	refs := make([]*plumbing.Reference, 0)
	ri, err := rg.References()
	if err != nil {
		return nil, err
	}
	err = ri.ForEach(func(r *plumbing.Reference) error {
		refs = append(refs, r)
		return nil
	})
	if err != nil {
		return nil, err
	}
	r.refs = refs
	return r, nil
}

// FindLatestFile returns the latest file for a given path.
func (r *Repo) FindLatestFile(pattern string) (*object.File, error) {
	g, err := glob.Compile(pattern)
	if err != nil {
		return nil, err
	}
	c, err := r.commitForHash(r.head.Hash())
	if err != nil {
		return nil, err
	}
	fi, err := c.Files()
	if err != nil {
		return nil, err
	}
	var f *object.File
	err = fi.ForEach(func(ff *object.File) error {
		if g.Match(ff.Name) {
			f = ff
			return storer.ErrStop
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if f == nil {
		return nil, object.ErrFileNotFound
	}
	return f, nil
}

// LatestFile returns the contents of the latest file at the specified path in the repository.
func (r *Repo) LatestFile(pattern string) (string, error) {
	f, err := r.FindLatestFile(pattern)
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

// UpdateServerInfo updates the server info for the repository.
func (r *Repo) UpdateServerInfo() error {
	cmd := exec.Command("git", "update-server-info")
	cmd.Dir = r.path
	return cmd.Run()
}
