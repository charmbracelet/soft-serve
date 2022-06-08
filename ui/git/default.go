package git

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/soft-serve/git"
	"github.com/gobwas/glob"
)

type gitRepo struct {
	repo *git.Repository
	desc string
	rm   string
	rp   string
}

// NewRepo returns a new git repository that conforms to the GitRepo interface.
func NewRepo(path string) (GitRepo, error) {
	repo, err := git.Open(path)
	if err != nil {
		return nil, err
	}
	return &gitRepo{repo: repo}, nil
}

func (r *gitRepo) Name() string {
	return r.repo.Name()
}

func (r *gitRepo) Repo() string {
	return r.repo.Name()
}

func (r *gitRepo) Description() string {
	if r.desc != "" {
		return r.desc
	}
	gd, err := r.repo.RevParse("--git-dir")
	if err != nil {
		return ""
	}
	gp := filepath.Join(r.repo.Path, gd, "description")
	desc, err := os.ReadFile(gp)
	if err != nil {
		return ""
	}
	r.desc = strings.TrimSpace(string(desc))
	return r.desc
}

func (r *gitRepo) IsPrivate() bool {
	return false
}

func (r *gitRepo) Readme() (string, string) {
	if r.rm != "" && r.rp != "" {
		return r.rm, r.rp
	}
	pattern := "README*"
	g := glob.MustCompile(pattern)
	dir := filepath.Dir(pattern)
	head, err := r.HEAD()
	if err != nil {
		return "", ""
	}
	t, err := r.repo.TreePath(head, dir)
	if err != nil {
		return "", ""
	}
	ents, err := t.Entries()
	if err != nil {
		return "", ""
	}
	for _, e := range ents {
		fp := filepath.Join(dir, e.Name())
		if e.IsTree() {
			continue
		}
		if g.Match(fp) {
			bts, err := e.Contents()
			if err != nil {
				return "", ""
			}
			r.rm = string(bts)
			r.rp = fp
			return r.rm, r.rp
		}
	}
	return "", ""
}

func (r *gitRepo) Tree(ref *git.Reference, path string) (*git.Tree, error) {
	return r.repo.TreePath(ref, path)
}

func (r *gitRepo) CommitsByPage(ref *git.Reference, page, size int) (git.Commits, error) {
	return r.repo.CommitsByPage(ref, page, size)
}

func (r *gitRepo) CountCommits(ref *git.Reference) (int64, error) {
	tc, err := r.repo.CountCommits(ref)
	if err != nil {
		return 0, err
	}
	return tc, nil
}

func (r *gitRepo) Diff(commit *git.Commit) (*git.Diff, error) {
	diff, err := r.repo.Diff(commit)
	if err != nil {
		return nil, err
	}
	return diff, nil
}

func (r *gitRepo) HEAD() (*git.Reference, error) {
	h, err := r.repo.HEAD()
	if err != nil {
		return nil, err
	}
	return h, nil
}

func (r *gitRepo) References() ([]*git.Reference, error) {
	return r.repo.References()
}
