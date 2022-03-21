package git

import (
	"bufio"
	"bytes"
	"errors"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	gitypes "github.com/charmbracelet/soft-serve/internal/tui/bubbles/git/types"
	"github.com/go-git/go-git/v5/utils/binary"
	"github.com/gobwas/glob"
	"github.com/gogs/git-module"
	"github.com/golang/groupcache/lru"
)

// ErrMissingRepo indicates that the requested repository could not be found.
var ErrMissingRepo = errors.New("missing repo")

// Repo represents a Git repository.
type Repo struct {
	path         string
	repository   *git.Repository
	Readme       string
	ReadmePath   string
	head         string
	refs         []*git.Reference
	totalCommits int64
	patchCache   *lru.Cache
	commitCache  *lru.Cache
}

func (rs *RepoSource) Open(path string) (*git.Repository, error) {
	r, err := git.Open(path)
	if err != nil {
		return nil, err
	}
	return r, nil
}

// GetName returns the name of the repository.
func (r *Repo) Name() string {
	return filepath.Base(r.path)
}

// GetHEAD returns the reference for a repository.
func (r *Repo) GetHEAD() *git.Reference {
	ref := git.Reference{
		Refspec: r.head,
	}
	ref.ID, _ = git.ShowRefVerify(r.path, r.head)
	return &ref
}

// GetReferences returns the references for a repository.
func (r *Repo) GetReferences() []*git.Reference {
	return r.refs
}

// GetRepository returns the underlying go-git repository object.
func (r *Repo) Repository() *git.Repository {
	return r.repository
}

var treeEntryRe = regexp.MustCompile(`([0-9]{6})\s+(\w+)\s+([0-9a-z]+)\s+([0-9-]+)\s+(.*)`)

// Tree returns the git tree for a given path.
func (r *Repo) Tree(ref *git.Reference, path string) (*git.Tree, error) {
	path = filepath.Clean(path)
	if path == "." {
		path = ""
	}
	hash := ref.ID
	t, err := r.repository.LsTree(hash)
	if err != nil {
		return nil, err
	}
	return t, nil
}

func (r *Repo) TreeEntryFile(e *git.TreeEntry) (*git.Blob, error) {
	if e.IsTree() {
		return nil, git.ErrNotBlob
	}
	return e.Blob(), nil
}

func IsBinary(blob *git.Blob) (bool, error) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	err := blob.Pipeline(stdout, stderr)
	if err != nil {
		return false, err
	}
	r := bufio.NewReader(stdout)
	return binary.IsBinary(r)
}

// PatchCtx returns the patch for a given commit.
func (r *Repo) Patch(hash string) (*git.Diff, error) {
	commit, err := r.Commit(hash)
	if err != nil {
		return nil, err
	}
	diff, err := r.repository.Diff(commit.ID.String(), 1000, 1000, 1000)
	if err != nil {
		return nil, err
	}
	return diff, nil
}

func (r *Repo) Commit(hash string) (*git.Commit, error) {
	return r.repository.CommitByRevision(hash)
}

func (r *Repo) GetCommitsByPage(ref *git.Reference, page, size int) (gitypes.Commits, error) {
	return r.repository.CommitsByPage(ref.ID, page, size)
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
	err := git.Init(rp, git.InitOptions{Bare: bare})
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
		path:        path,
		repository:  rg,
		commitCache: lru.New(1000),
		patchCache:  lru.New(1000),
		refs:        make([]*git.Reference, 0),
	}
	r.head, err = rg.SymbolicRef()
	if err != nil {
		return nil, err
	}
	r.refs, err = rg.ShowRef()
	if err != nil {
		return nil, err
	}
	return
}

func (r *Repo) LsFiles(pattern string) ([]string, error) {
	g := glob.MustCompile(pattern)
	t, err := r.repository.LsTree(r.GetHEAD().ID)
	if err != nil {
		return nil, err
	}
	ents, err := t.Entries()
	if err != nil {
		return nil, err
	}
	files := make([]string, 0)
	for _, e := range ents {
		if g.Match(e.Name()) {
			files = append(files, e.Name())
		}
	}
	return files, nil
}

// LatestFile returns the contents of the latest file at the specified path in the repository.
func (r *Repo) LatestFile(pattern string) (string, error) {
	g := glob.MustCompile(pattern)
	c, err := r.repository.CommitByRevision(r.GetHEAD().ID)
	if err != nil {
		return "", err
	}
	ents, err := c.Tree.Entries()
	if err != nil {
		return "", err
	}
	for _, e := range ents {
		if g.Match(e.Name()) {
			bts, err := e.Blob().Bytes()
			if err != nil {
				return "", err
			}
			return string(bts), nil
		}
	}
	return "", git.ErrURLNotExist
}

// LatestTree returns the latest tree at the specified path in the repository.
func (r *Repo) LatestTree(path string) (*git.Tree, error) {
	path = filepath.Clean(path)
	t, err := r.repository.LsTree(r.GetHEAD().ID)
	if err != nil {
		return nil, err
	}
	if path == "." {
		return t, nil
	}
	return t.Subtree(path)
}

// UpdateServerInfo updates the server info for the repository.
func (r *Repo) UpdateServerInfo() error {
	return r.gitCmd("update-server-info").Run()
}

func (r *Repo) Count(ref *git.Reference) (int64, error) {
	if r.totalCommits != 0 {
		return r.totalCommits, nil
	}
	c, err := r.repository.CommitByRevision(r.GetHEAD().ID)
	if err != nil {
		return 0, err
	}
	return c.CommitsCount()
}

func (r *Repo) Show(args ...string) (string, error) {
	return r.gitCmdOutput(append([]string{"show"}, args...)...)
}

func (r *Repo) LogProcessLine(cb func(line string) (bool, error), args ...string) error {
	return r.gitCmdProcessLine(cb, append([]string{"log"}, args...)...)
}

func (r *Repo) LsTree(args ...string) (string, error) {
	return r.gitCmdOutput(append([]string{"ls-tree"}, args...)...)
}

func (r *Repo) gitCmdOutput(args ...string) (string, error) {
	out, err := r.gitCmd(args...).Output()
	if err != nil {
		log.Printf("%T", err)
		return "", err
	}
	return strings.TrimSpace(strings.ReplaceAll(string(out), "\r", "")), nil
}

func (r *Repo) gitCmdProcessLine(cb func(line string) (bool, error), args ...string) error {
	cmd := r.gitCmd(args...)
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	scanner := bufio.NewScanner(stdoutPipe)
	scanner.Split(bufio.ScanLines)
	if err := cmd.Start(); err != nil {
		return err
	}

	for scanner.Scan() {
		line := strings.ReplaceAll(scanner.Text(), "\r", "")
		stop, err := cb(line)
		if err != nil {
			return err
		}
		if stop {
			_ = cmd.Process.Kill()
			break
		}
	}

	_ = cmd.Wait()

	return nil
}

func (r *Repo) gitCmd(args ...string) *exec.Cmd {
	log.Printf("git %s", strings.Join(args, " "))
	cmd := exec.Command("git", args...)
	cmd.Dir = r.path
	return cmd
}
