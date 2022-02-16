package git

import (
	"errors"
	"log"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

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
	Name        string
	Repository  *git.Repository
	Readme      string
	LastUpdated *time.Time
	refCommits  map[plumbing.Hash]gitypes.Commits
	ref         *plumbing.Reference
}

// GetName returns the name of the repository.
func (r *Repo) GetName() string {
	return r.Name
}

// GetReference returns the reference for a repository.
func (r *Repo) GetReference() *plumbing.Reference {
	return r.ref
}

// SetReference sets the repository head reference.
func (r *Repo) SetReference(ref *plumbing.Reference) error {
	r.ref = ref
	return nil
}

// GetRepository returns the underlying go-git repository object.
func (r *Repo) GetRepository() *git.Repository {
	return r.Repository
}

// Tree returns the git tree for a given path.
func (r *Repo) Tree(ref *plumbing.Reference, path string) (*object.Tree, error) {
	path = filepath.Clean(path)
	hash, err := r.targetHash(ref)
	if err != nil {
		return nil, err
	}
	c, err := r.Repository.CommitObject(hash)
	if err != nil {
		return nil, err
	}
	t, err := c.Tree()
	if err != nil {
		return nil, err
	}
	if path == "." {
		return t, nil
	}
	return t.Tree(path)
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
	log.Printf("caching commits for %s/%s: %s", r.Name, ref.Name(), ref.Hash())
	commits = gitypes.Commits{}
	co, err := r.Repository.CommitObject(hash)
	if err != nil {
		return nil, err
	}
	// traverse the commit tree to get all commits
	commits = append(commits, &gitypes.Commit{Commit: co})
	for {
		co, err = co.Parent(0)
		if err != nil {
			if err == object.ErrParentNotFound {
				err = nil
			}
			break
		}
		commits = append(commits, &gitypes.Commit{Commit: co})
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
		to, err := r.Repository.TagObject(hash)
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

// loadCommits loads the commits for a repository.
func (r *Repo) loadCommits(ref *plumbing.Reference) (gitypes.Commits, error) {
	commits := gitypes.Commits{}
	hash, err := r.targetHash(ref)
	if err != nil {
		return nil, err
	}
	l, err := r.Repository.Log(&git.LogOptions{
		Order: git.LogOrderCommitterTime,
		From:  hash,
	})
	if err != nil {
		return nil, err
	}
	defer l.Close()
	err = l.ForEach(func(c *object.Commit) error {
		commits = append(commits, &gitypes.Commit{Commit: c})
		return nil
	})
	if err != nil {
		return nil, err
	}
	return commits, nil
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
		if r.Name == name {
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
		Name:       name,
		Repository: rg,
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
		rn := de.Name()
		rg, err := git.PlainOpen(filepath.Join(rs.Path, rn))
		if err != nil {
			return err
		}
		r, err := rs.loadRepo(rn, rg)
		if err != nil {
			return err
		}
		rs.repos = append(rs.repos, r)
	}
	return nil
}

func (rs *RepoSource) loadRepo(name string, rg *git.Repository) (*Repo, error) {
	r := &Repo{
		Name:       name,
		Repository: rg,
	}
	r.refCommits = make(map[plumbing.Hash]gitypes.Commits)
	ref, err := rg.Head()
	if err != nil {
		return nil, err
	}
	r.ref = ref
	rm, err := r.LatestFile("README.md")
	if err != nil {
		return nil, err
	}
	r.Readme = rm
	return r, nil
}

// LatestFile returns the latest file at the specified path in the repository.
func (r *Repo) LatestFile(path string) (string, error) {
	lg, err := r.Repository.Log(&git.LogOptions{
		From: r.GetReference().Hash(),
	})
	if err != nil {
		return "", err
	}
	c, err := lg.Next()
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
