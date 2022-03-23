package git

import (
	"errors"
	"log"
	"os"
	"path/filepath"
	"sync"

	"github.com/charmbracelet/soft-serve/pkg/git"
	"github.com/gobwas/glob"
	"github.com/golang/groupcache/lru"
)

// ErrMissingRepo indicates that the requested repository could not be found.
var ErrMissingRepo = errors.New("missing repo")

// Repo represents a Git repository.
type Repo struct {
	path       string
	repository *git.Repository
	readme     string
	readmePath string
	head       *git.Reference
	refs       []*git.Reference
	patchCache *lru.Cache
}

func (rs *RepoSource) Open(path string) (*git.Repository, error) {
	return git.Open(path)
}

// Path returns the path to the repository.
func (r *Repo) Path() string {
	return r.path
}

// GetName returns the name of the repository.
func (r *Repo) Name() string {
	return filepath.Base(r.path)
}

func (r *Repo) Readme() (readme string, path string) {
	return r.readme, r.readmePath
}

func (r *Repo) SetReadme(readme, path string) {
	r.readme = readme
	r.readmePath = path
}

// HEAD returns the reference for a repository.
func (r *Repo) HEAD() (*git.Reference, error) {
	if r.head != nil {
		return r.head, nil
	}
	h, err := r.repository.HEAD()
	if err != nil {
		return nil, err
	}
	r.head = h
	return h, nil
}

// GetReferences returns the references for a repository.
func (r *Repo) References() ([]*git.Reference, error) {
	if r.refs != nil {
		return r.refs, nil
	}
	refs, err := r.repository.References()
	if err != nil {
		return nil, err
	}
	r.refs = refs
	return refs, nil
}

// Tree returns the git tree for a given path.
func (r *Repo) Tree(ref *git.Reference, path string) (*git.Tree, error) {
	return r.repository.TreePath(ref, path)
}

// Diff returns the diff for a given commit.
func (r *Repo) Diff(commit *git.Commit) (*git.Diff, error) {
	hash := commit.Hash.String()
	c, ok := r.patchCache.Get(hash)
	if ok {
		return c.(*git.Diff), nil
	}
	diff, err := r.repository.Diff(commit)
	if err != nil {
		return nil, err
	}
	r.patchCache.Add(hash, diff)
	return diff, nil
}

// CountCommits returns the number of commits for a repository.
func (r *Repo) CountCommits(ref *git.Reference) (int64, error) {
	tc, err := r.repository.CountCommits(ref)
	if err != nil {
		return 0, err
	}
	return tc, nil
}

// CommitsByPage returns the commits for a repository.
func (r *Repo) CommitsByPage(ref *git.Reference, page, size int) (git.Commits, error) {
	return r.repository.CommitsByPage(ref, page, size)
}

// Push pushes the repository to the remote.
func (r *Repo) Push(remote, branch string) error {
	return r.repository.Push(remote, branch)
}

// RepoSource is a reference to an on-disk repositories.
type RepoSource struct {
	Path  string
	mtx   sync.Mutex
	repos map[string]*Repo
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
	_, err := git.Init(rp, bare)
	if err != nil {
		return nil, err
	}
	if bare {
		temp, err := os.MkdirTemp("", name)
		if err != nil {
			return nil, err
		}
		err = git.Clone(rp, temp)
		if err != nil {
			return nil, err
		}
		rp = temp
	}
	rg, err := git.Open(rp)
	if err != nil {
		return nil, err
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
	rg, err := rs.Open(rp)
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

func (rs *RepoSource) loadRepo(path string, rg *git.Repository) (r *Repo, err error) {
	r = &Repo{
		path:       path,
		repository: rg,
		patchCache: lru.New(1000),
	}
	_, err = r.HEAD()
	if err != nil {
		return nil, err
	}
	_, err = r.References()
	if err != nil {
		return nil, err
	}
	return
}

// LatestFile returns the contents of the latest file at the specified path in the repository.
func (r *Repo) LatestFile(pattern string) (string, string, error) {
	g := glob.MustCompile(pattern)
	dir := filepath.Dir(pattern)
	t, err := r.repository.TreePath(r.head, dir)
	if err != nil {
		return "", "", err
	}
	ents, err := t.Entries()
	if err != nil {
		return "", "", err
	}
	for _, e := range ents {
		fp := filepath.Join(dir, e.Name())
		if g.Match(fp) {
			bts, err := e.Contents()
			if err != nil {
				return "", "", err
			}
			return string(bts), fp, nil
		}
	}
	return "", "", git.ErrFileNotFound
}

// UpdateServerInfo updates the server info for the repository.
func (r *Repo) UpdateServerInfo() error {
	return r.repository.UpdateServerInfo()
}
