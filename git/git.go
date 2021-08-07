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

type ReadmeTransform func(string) string

func (cl CommitLog) Len() int      { return len(cl) }
func (cl CommitLog) Swap(i, j int) { cl[i], cl[j] = cl[j], cl[i] }
func (cl CommitLog) Less(i, j int) bool {
	return cl[i].Commit.Author.When.After(cl[j].Commit.Author.When)
}

type RepoSource struct {
	mtx             sync.Mutex
	path            string
	repos           []*Repo
	commits         CommitLog
	readmeTransform ReadmeTransform
}

func NewRepoSource(repoPath string, poll time.Duration, rf ReadmeTransform) *RepoSource {
	rs := &RepoSource{path: repoPath, readmeTransform: rf}
	go func() {
		for {
			rs.loadRepos()
			time.Sleep(poll)
		}
	}()
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

func (rs *RepoSource) GetCommits(limit int) []RepoCommit {
	rs.mtx.Lock()
	defer rs.mtx.Unlock()
	if limit > len(rs.commits) {
		limit = len(rs.commits)
	}
	return rs.commits[:limit]
}

func (rs *RepoSource) loadRepos() {
	rs.mtx.Lock()
	defer rs.mtx.Unlock()
	rd, err := os.ReadDir(rs.path)
	if err != nil {
		return
	}
	rs.repos = make([]*Repo, 0)
	rs.commits = make([]RepoCommit, 0)
	for _, de := range rd {
		rn := de.Name()
		r := &Repo{Name: rn}
		rg, err := git.PlainOpen(rs.path + string(os.PathSeparator) + rn)
		if err != nil {
			log.Fatal(err)
		}
		r.Repository = rg
		l, err := rg.Log(&git.LogOptions{All: true})
		if err != nil {
			log.Fatal(err)
		}
		l.ForEach(func(c *object.Commit) error {
			if r.LastUpdated == nil {
				r.LastUpdated = &c.Author.When
				rf, err := c.File("README.md")
				if err == nil {
					rmd, err := rf.Contents()
					if err == nil {
						r.Readme = rs.readmeTransform(rmd)
					}
				}
			}
			rs.commits = append(rs.commits, RepoCommit{Name: rn, Commit: c})
			return nil
		})
		sort.Sort(rs.commits)
		rs.repos = append(rs.repos, r)
	}
}
