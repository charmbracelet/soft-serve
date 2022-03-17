package git

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	gitypes "github.com/charmbracelet/soft-serve/internal/tui/bubbles/git/types"
	"github.com/go-git/go-billy/v5/memfs"
	"github.com/go-git/go-billy/v5/osfs"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/storage/filesystem"
	"github.com/go-git/go-git/v5/storage/memory"
	"github.com/gobwas/glob"
)

// ErrMissingRepo indicates that the requested repository could not be found.
var ErrMissingRepo = errors.New("missing repo")

// Repo represents a Git repository.
type Repo struct {
	path         string
	repository   *git.Repository
	Readme       string
	ReadmePath   string
	head         *plumbing.Reference
	refs         []*plumbing.Reference
	totalCommits int
}

type noCache struct{}

func (n *noCache) Put(o plumbing.EncodedObject) {}
func (n *noCache) Get(k plumbing.Hash) (plumbing.EncodedObject, bool) {
	return nil, false
}
func (n *noCache) Clear() {}

func (rs *RepoSource) Open(path string) (*git.Repository, error) {
	path = filepath.Clean(path)
	b := osfs.New(path)
	s := filesystem.NewStorage(b, &noCache{})
	return git.Open(s, nil)
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

var treeEntryRe = regexp.MustCompile(`([0-9]{6})\s+(\w+)\s+([0-9a-z]+)\s+([0-9-]+)\s+(.*)`)

// Tree returns the git tree for a given path.
func (r *Repo) Tree(ref *plumbing.Reference, path string) (*gitypes.Tree, error) {
	t := &gitypes.Tree{
		Entries: make(gitypes.TreeEntries, 0),
	}
	path = filepath.Clean(path)
	if path == "." {
		path = ""
	}
	hash, err := r.targetHash(ref)
	if err != nil {
		return nil, err
	}
	c, err := r.Commit(hash.String())
	if err != nil {
		return nil, err
	}
	t.Hash = c.TreeHash
	lstree, err := r.LsTree("-l", "--full-tree", fmt.Sprintf("%s:%s", c.TreeHash, path))
	if err != nil {
		return nil, err
	}
	lines := strings.Split(lstree, "\n")
	for _, line := range lines {
		m := treeEntryRe.FindStringSubmatch(line)
		mo, err := strconv.ParseUint(m[1], 8, 32)
		if err != nil {
			return nil, err
		}
		var si int64
		if m[4] != "-" {
			si, err = strconv.ParseInt(m[4], 10, 64)
			if err != nil {
				return nil, err
			}
		}
		e := &gitypes.TreeEntry{
			EntryMode: fs.FileMode(mo),
			EntryType: m[2],
			EntryHash: m[3],
			EntrySize: si,
			EntryName: filepath.Join(path, m[5]),
		}
		t.Entries = append(t.Entries, e)
	}
	return t, nil
}

func (r *Repo) TreeEntryFile(e *gitypes.TreeEntry) (*gitypes.File, error) {
	if e.IsDir() {
		return nil, object.ErrFileNotFound
	}
	f, err := r.Show("-q", "--text", "--format=", fmt.Sprintf("%s:%s", r.head.Hash().String(), e.Path()))
	if err != nil {
		return nil, object.ErrFileNotFound
	}
	return &gitypes.File{
		Entry: e,
		Blob:  []byte(f),
	}, nil
}

func (r *Repo) treeForHash(treeHash plumbing.Hash) (*object.Tree, error) {
	var err error
	t, err := r.repository.TreeObject(treeHash)
	if err != nil {
		return nil, err
	}
	return t, nil
}

func (r *Repo) commitForHash(hash plumbing.Hash) (*object.Commit, error) {
	var err error
	co, err := r.repository.CommitObject(hash)
	if err != nil {
		return nil, err
	}
	return co, nil
}

func (r *Repo) patchForHashCtx(ctx context.Context, hash plumbing.Hash) (*object.Patch, error) {
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
func (r *Repo) Patch(commit *gitypes.Commit) (*gitypes.Patch, error) {
	hash := commit.Hash
	p := &gitypes.Patch{}
	ps, err := r.Show("--format=", "--stat", "--patch", hash)
	if err != nil {
		return nil, err
	}
	diff := false
	for _, l := range strings.Split(ps, "\n") {
		l = strings.TrimSpace(l)
		if diff {
			p.Diff += l + "\n"
		} else {
			p.Stat += l + "\n"
		}
		if l == "" {
			diff = true
		}
	}
	return p, nil
}

// Matches the "raw" format of a git commit.
// commit ([0-9a-f]{40})\ntree ([0-9a-f]{40})\nparent ([0-9a-f]{40})\nauthor (.*) <(.*)> (.*)\ncommitter (.*) <(.*)> (.*)\n((?:.*\n?)*)?
// (?:gpgsig ((?:-----BEGIN PGP SIGNATURE-----\n)\n(?: (?:.*\n?))*(?:-----END PGP SIGNATURE-----))\n)?\n((?:.*\n?)*)
// commit ([0-9a-f]{40}) ?\(?(.*[^\)])?\)?
// Author:\s+(.*) <(.*)>
// AuthorDate:\s+(.*)
// Commit:\s+(.*) <(.*)>
// CommitDate:\s+(.*)
// \s+(.*)\n?\n?((.*\n?)*)
// var commitMessageRe = regexp.MustCompile(`(?:gpgsig ((?:-----BEGIN PGP SIGNATURE-----\n)\n?(?: (?:.*\n?))*(?:-----END PGP SIGNATURE-----))\n)?\n((?:.*\n?)*)`)
// var commitRe = regexp.MustCompile(`commit ([0-9a-f]{40})\ntree ([0-9a-f]{40})\nparent ([0-9a-f]{40})\nauthor (.*) <(.*)> (.*)\ncommitter (.*) <(.*)> (.*)\n((?:.*\n?)*)`)

var authorRe = regexp.MustCompile(`(.*) <(.*)> ([0-9]+) ([-+][0-9]{4})`)

func (r *Repo) parseCommitDate(timestamp, offset string) (*time.Time, error) {
	ts, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return nil, err
	}
	date := time.Unix(ts, 0).UTC()
	if offset != "" {
		of, err := strconv.ParseInt(offset, 10, 32)
		if err != nil {
			return nil, err
		}
		loc := time.FixedZone("UTC"+offset, int(of))
		date = date.In(loc)
	}
	return &date, nil
}

var commitMessageRe = regexp.MustCompile(`(?:\t| {4})(.*)`)

func (r *Repo) parseCommitString(cs string) (*gitypes.Commit, error) {
	c := &gitypes.Commit{}
	for _, l := range strings.Split(cs, "\n") {
		switch {
		case strings.HasPrefix(l, "commit "):
			c.Hash = strings.TrimPrefix(l, "commit ")
		case strings.HasPrefix(l, "tree "):
			c.TreeHash = strings.TrimPrefix(l, "tree ")
		case strings.HasPrefix(l, "parent "):
			c.Parents = append(c.Parents, strings.TrimPrefix(l, "parent "))
		case strings.HasPrefix(l, "author "):
			m := authorRe.FindStringSubmatch(strings.TrimPrefix(l, "author "))
			date, err := r.parseCommitDate(m[3], m[4])
			if err != nil {
				return nil, err
			}
			c.Author = gitypes.Signature{
				Name:  m[1],
				Email: m[2],
				When:  *date,
			}
		case strings.HasPrefix(l, "committer "):
			m := authorRe.FindStringSubmatch(strings.TrimPrefix(l, "committer "))
			date, err := r.parseCommitDate(m[3], m[4])
			if err != nil {
				return nil, err
			}
			c.Committer = gitypes.Signature{
				Name:  m[1],
				Email: m[2],
				When:  *date,
			}
		case commitMessageRe.MatchString(l):
			c.Message += gitypes.CommitMessage(strings.TrimSpace(l) + "\n")
		// TODO implement mergetag and PGP signature parsing
		default:
		}
	}
	return c, nil
}

func (r *Repo) Commit(hash string) (*gitypes.Commit, error) {
	s, err := r.Show("-q", "--pretty=raw", hash)
	if err != nil {
		return nil, err
	}
	return r.parseCommitString(s)
}

// GetCommits returns the commits for a repository.
func (r *Repo) GetCommits(ref *plumbing.Reference) (gitypes.Commits, error) {
	count, err := r.Count(ref)
	if err != nil {
		return nil, err
	}
	return r.GetCommitsLimit(ref, count)
}

func (r *Repo) GetCommitsLimit(ref *plumbing.Reference, limit int) (gitypes.Commits, error) {
	return r.GetCommitsSkip(ref, 0, limit)
}

func (r *Repo) GetCommitsSkip(ref *plumbing.Reference, skip, limit int) (gitypes.Commits, error) {
	var err error
	hash, err := r.targetHash(ref)
	if err != nil {
		return nil, err
	}
	commits := gitypes.Commits{}
	var cs string
	err = r.LogProcessLine(func(line string) (bool, error) {
		if strings.HasPrefix(line, "commit ") && cs != "" {
			c, err := r.parseCommitString(cs)
			if err != nil {
				return true, err
			}
			commits = append(commits, c)
			cs = ""
		}
		cs += line + "\n"
		return false, nil
	},
		"--pretty=raw", "--topo-order", fmt.Sprintf("--skip=%d", skip),
		fmt.Sprintf("--max-count=%d", limit), hash.String())
	if err != nil {
		return nil, err
	}
	c, err := r.parseCommitString(cs)
	if err != nil {
		return nil, err
	}
	commits = append(commits, c)
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

func (rs *RepoSource) loadRepo(path string, rg *git.Repository) (*Repo, error) {
	r := &Repo{
		path:       path,
		repository: rg,
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

func (r *Repo) LsFiles(pattern string) ([]string, error) {
	out, err := r.LsTree("--name-only", "-r", r.head.Hash().String())
	if err != nil {
		return nil, err
	}
	lines := strings.Split(out, "\n")
	g, err := glob.Compile(pattern)
	if err != nil {
		return nil, err
	}
	match := make([]string, 0)
	for _, l := range lines {
		if g.Match(l) {
			match = append(match, l)
		}
	}
	return match, nil
}

// LatestFile returns the contents of the latest file at the specified path in the repository.
func (r *Repo) LatestFile(pattern string) (string, error) {
	lines, err := r.LsFiles(pattern)
	if err != nil {
		return "", err
	}
	if len(lines) == 0 {
		return "", object.ErrFileNotFound
	}
	return r.Show("-q", "--format=", fmt.Sprintf("%s:%s", r.head.Hash().String(), lines[0]))
}

// LatestTree returns the latest tree at the specified path in the repository.
func (r *Repo) LatestTree(path string) (*gitypes.Tree, error) {
	return r.Tree(r.head, path)
}

// UpdateServerInfo updates the server info for the repository.
func (r *Repo) UpdateServerInfo() error {
	return r.gitCmd("update-server-info").Run()
}

func (r *Repo) Count(ref *plumbing.Reference) (int, error) {
	if r.totalCommits != 0 {
		return r.totalCommits, nil
	}
	hash, err := r.targetHash(ref)
	if err != nil {
		return 0, err
	}
	out, err := r.gitCmdOutput("rev-list", "--count", hash.String())
	if err != nil {
		return 0, err
	}
	tc, err := strconv.Atoi(string(out))
	if err != nil {
		return 0, err
	}
	r.totalCommits = tc
	return tc, nil
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
