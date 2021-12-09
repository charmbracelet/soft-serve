package git

import (
	"errors"
	"log"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/go-git/go-billy/v5/memfs"
	"github.com/go-git/go-git/v5"
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
}

// RepoCommit contains metadata for a Git commit.
type RepoCommit struct {
	Name   string
	Commit *object.Commit
}

// CommitLog is a series of Git commits.
type CommitLog []RepoCommit

func (cl CommitLog) Len() int      { return len(cl) }
func (cl CommitLog) Swap(i, j int) { cl[i], cl[j] = cl[j], cl[i] }
func (cl CommitLog) Less(i, j int) bool {
	return cl[i].Commit.Author.When.After(cl[j].Commit.Author.When)
}

// RepoSource is a reference to an on-disk repositories.
type RepoSource struct {
	Path    string
	mtx     sync.Mutex
	repos   []*Repo
	commits CommitLog
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

// GetCommits returns commits for the repository.
func (rs *RepoSource) GetCommits(limit int) []RepoCommit {
	rs.mtx.Lock()
	defer rs.mtx.Unlock()
	if limit > len(rs.commits) {
		limit = len(rs.commits)
	}
	return rs.commits[:limit]
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
	rs.commits = make([]RepoCommit, 0)
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
	r := &Repo{Name: name}
	r.Repository = rg
	l, err := rg.Log(&git.LogOptions{All: true})
	if err != nil {
		return nil, err
	}
	err = l.ForEach(func(c *object.Commit) error {
		if r.LastUpdated == nil {
			r.LastUpdated = &c.Author.When
			rf, err := c.File("README.md")
			if err == nil {
				rmd, err := rf.Contents()
				if err == nil {
					r.Readme = rmd
				}
			}
		}
		rs.commits = append(rs.commits, RepoCommit{Name: name, Commit: c})
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Sort(rs.commits)
	return r, nil
}

// LatestFile returns the latest file at the specified path in the repository.
func (r *Repo) LatestFile(path string) (string, error) {
	lg, err := r.Repository.Log(&git.LogOptions{})
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
