package tui

import (
	"log"
	"os"
	"sort"
	"sync"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
)

type commitLog []*object.Commit

func (cl commitLog) Len() int           { return len(cl) }
func (cl commitLog) Swap(i, j int)      { cl[i], cl[j] = cl[j], cl[i] }
func (cl commitLog) Less(i, j int) bool { return cl[i].Author.When.After(cl[j].Author.When) }

type repoSource struct {
	mtx     sync.Mutex
	path    string
	repos   []*git.Repository
	commits commitLog
}

func newRepoSource(repoPath string) *repoSource {
	rs := &repoSource{path: repoPath}
	go func() {
		for {
			rs.loadRepos()
			time.Sleep(time.Second * 10)
		}
	}()
	return rs
}

func (rs *repoSource) allRepos() []*git.Repository {
	rs.mtx.Lock()
	defer rs.mtx.Unlock()
	return rs.repos
}

func (rs *repoSource) getCommits(limit int) []*object.Commit {
	rs.mtx.Lock()
	defer rs.mtx.Unlock()
	if limit > len(rs.commits) {
		limit = len(rs.commits)
	}
	return rs.commits[:limit]
}

func (rs *repoSource) loadRepos() {
	rs.mtx.Lock()
	defer rs.mtx.Unlock()
	rd, err := os.ReadDir(rs.path)
	if err != nil {
		return
	}
	rs.repos = make([]*git.Repository, 0)
	rs.commits = make([]*object.Commit, 0)
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
			rs.commits = append(rs.commits, c)
			return nil
		})
		sort.Sort(rs.commits)
		rs.repos = append(rs.repos, r)
	}
}
