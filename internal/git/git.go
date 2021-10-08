package git

import (
	"errors"
	"log"
	"os"
	"sort"
	"sync"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
)

var ErrMissingRepo = errors.New("missing repo")

type Repo struct {
	Name        string
	Repository  *git.Repository
	Readme      string
	LastUpdated *time.Time
}

type RepoCommit struct {
	Name   string
	Commit *object.Commit
}

type CommitLog []RepoCommit

func (cl CommitLog) Len() int      { return len(cl) }
func (cl CommitLog) Swap(i, j int) { cl[i], cl[j] = cl[j], cl[i] }
func (cl CommitLog) Less(i, j int) bool {
	return cl[i].Commit.Author.When.After(cl[j].Commit.Author.When)
}

type RepoSource struct {
	Path    string
	mtx     sync.Mutex
	repos   []*Repo
	commits CommitLog
}

func NewRepoSource(repoPath string) *RepoSource {
	err := os.MkdirAll(repoPath, os.ModeDir|os.FileMode(0700))
	if err != nil {
		log.Fatal(err)
	}
	rs := &RepoSource{Path: repoPath}
	return rs
}

func (rs *RepoSource) AllRepos() []*Repo {
	rs.mtx.Lock()
	defer rs.mtx.Unlock()
	return rs.repos
}

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

func (rs *RepoSource) InitRepo(name string, bare bool) (*Repo, error) {
	rs.mtx.Lock()
	defer rs.mtx.Unlock()
	rg, err := git.PlainInit(rs.Path+string(os.PathSeparator)+name, bare)
	if err != nil {
		return nil, err
	}
	r := &Repo{
		Name:       name,
		Repository: rg,
	}
	rs.repos = append(rs.repos, r)
	return r, nil
}

func (rs *RepoSource) GetCommits(limit int) []RepoCommit {
	rs.mtx.Lock()
	defer rs.mtx.Unlock()
	if limit > len(rs.commits) {
		limit = len(rs.commits)
	}
	return rs.commits[:limit]
}

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
		rg, err := git.PlainOpen(rs.Path + string(os.PathSeparator) + rn)
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
