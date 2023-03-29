package server

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/log"
	"github.com/charmbracelet/soft-serve/git"
	"github.com/go-git/go-git/v5/plumbing/format/pktline"
)

var (

	// ErrNotAuthed represents unauthorized access.
	ErrNotAuthed = errors.New("you are not authorized to do this")

	// ErrSystemMalfunction represents a general system error returned to clients.
	ErrSystemMalfunction = errors.New("something went wrong")

	// ErrInvalidRepo represents an attempt to access a non-existent repo.
	ErrInvalidRepo = errors.New("invalid repo")

	// ErrInvalidRequest represents an invalid request.
	ErrInvalidRequest = errors.New("invalid request")

	// ErrMaxConnections represents a maximum connection limit being reached.
	ErrMaxConnections = errors.New("too many connections, try again later")

	// ErrTimeout is returned when the maximum read timeout is exceeded.
	ErrTimeout = errors.New("I/O timeout reached")
)

// Git protocol commands.
const (
	receivePackBin   = "git-receive-pack"
	uploadPackBin    = "git-upload-pack"
	uploadArchiveBin = "git-upload-archive"
)

// uploadPack runs the git upload-pack protocol against the provided repo.
func uploadPack(in io.Reader, out io.Writer, er io.Writer, repoDir string) error {
	exists, err := fileExists(repoDir)
	if !exists {
		return ErrInvalidRepo
	}
	if err != nil {
		return err
	}
	return runGit(in, out, er, "", uploadPackBin[4:], repoDir)
}

// uploadArchive runs the git upload-archive protocol against the provided repo.
func uploadArchive(in io.Reader, out io.Writer, er io.Writer, repoDir string) error {
	exists, err := fileExists(repoDir)
	if !exists {
		return ErrInvalidRepo
	}
	if err != nil {
		return err
	}
	return runGit(in, out, er, "", uploadArchiveBin[4:], repoDir)
}

// receivePack runs the git receive-pack protocol against the provided repo.
func receivePack(in io.Reader, out io.Writer, er io.Writer, repoDir string) error {
	if err := runGit(in, out, er, "", receivePackBin[4:], repoDir); err != nil {
		return err
	}
	return ensureDefaultBranch(in, out, er, repoDir)
}

// runGit runs a git command in the given repo.
func runGit(in io.Reader, out io.Writer, err io.Writer, dir string, args ...string) error {
	c := git.NewCommand(args...)
	return c.RunInDirWithOptions(dir, git.RunInDirOptions{
		Stdin:  in,
		Stdout: out,
		Stderr: err,
	})
}

// writePktline encodes and writes a pktline to the given writer.
func writePktline(w io.Writer, v ...interface{}) {
	msg := fmt.Sprintln(v...)
	pkt := pktline.NewEncoder(w)
	if err := pkt.EncodeString(msg); err != nil {
		log.Debugf("git: error writing pkt-line message: %s", err)
	}
	if err := pkt.Flush(); err != nil {
		log.Debugf("git: error flushing pkt-line message: %s", err)
	}
}

// ensureWithin ensures the given repo is within the repos directory.
func ensureWithin(reposDir string, repo string) error {
	repoDir := filepath.Join(reposDir, repo)
	absRepos, err := filepath.Abs(reposDir)
	if err != nil {
		log.Debugf("failed to get absolute path for repo: %s", err)
		return ErrSystemMalfunction
	}
	absRepo, err := filepath.Abs(repoDir)
	if err != nil {
		log.Debugf("failed to get absolute path for repos: %s", err)
		return ErrSystemMalfunction
	}

	// ensure the repo is within the repos directory
	if !strings.HasPrefix(absRepo, absRepos) {
		log.Debugf("repo path is outside of repos directory: %s", absRepo)
		return ErrInvalidRepo
	}

	return nil
}

func fileExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return true, err
}

func ensureDefaultBranch(in io.Reader, out io.Writer, er io.Writer, repoPath string) error {
	r, err := git.Open(repoPath)
	if err != nil {
		return err
	}
	brs, err := r.Branches()
	if err != nil {
		return err
	}
	if len(brs) == 0 {
		return fmt.Errorf("no branches found")
	}
	// Rename the default branch to the first branch available
	_, err = r.HEAD()
	if err == git.ErrReferenceNotExist {
		err = runGit(in, out, er, repoPath, "branch", "-M", brs[0])
		if err != nil {
			return err
		}
	}
	if err != nil && err != git.ErrReferenceNotExist {
		return err
	}
	return nil
}
