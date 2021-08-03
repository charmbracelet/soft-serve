package git

import (
	"log"
	"os"
	"sort"
	"sync"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
)

type RepoCommit struct {
	Name   string
	Repo   *git.Repository
	Commit *object.Commit
}

type CommitLog []RepoCommit

func (cl CommitLog) Len() int      { return len(cl) }
func (cl CommitLog) Swap(i, j int) { cl[i], cl[j] = cl[j], cl[i] }
func (cl CommitLog) Less(i, j int) bool {
	return cl[i].Commit.Author.When.After(cl[j].Commit.Author.When)
}

type RepoSource struct {
	mtx     sync.Mutex
	path    string
	repos   []*git.Repository
	commits CommitLog
}

func NewRepoSource(repoPath string) *RepoSource {
	rs := &RepoSource{path: repoPath}
	go func() {
		for {
			rs.loadRepos()
			time.Sleep(time.Second * 10)
		}
	}()
	return rs
}

func (rs *RepoSource) AllRepos() []*git.Repository {
	rs.mtx.Lock()
	defer rs.mtx.Unlock()
	return rs.repos
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
	rs.repos = make([]*git.Repository, 0)
	rs.commits = make([]RepoCommit, 0)
	for _, rd := range rd {
		r, err := git.PlainOpen(rs.path + string(os.PathSeparator) + rd.Name())
		if err != nil {
			log.Fatal(err)
		}
		l, err := r.Log(&git.LogOptions{All: true})
		if err != nil {
			log.Fatal(err)
		}
		l.ForEach(func(c *object.Commit) error {
			rs.commits = append(rs.commits, RepoCommit{Name: rd.Name(), Repo: r, Commit: c})
			return nil
		})
		sort.Sort(rs.commits)
		rs.repos = append(rs.repos, r)
	}
}
