package git

import (
	"context"
	"errors"
	"fmt"
	"io"
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

// WritePktline encodes and writes a pktline to the given writer.
func WritePktline(w io.Writer, v ...interface{}) error {
	msg := fmt.Sprintln(v...)
	pkt := pktline.NewEncoder(w)
	if err := pkt.EncodeString(msg); err != nil {
		return fmt.Errorf("git: error writing pkt-line message: %w", err)
	}
	if err := pkt.Flush(); err != nil {
		return fmt.Errorf("git: error flushing pkt-line message: %w", err)
	}

	return nil
}

// WritePktlineErr writes an error pktline to the given writer.
func WritePktlineErr(w io.Writer, err error) error {
	return WritePktline(w, "ERR ", err.Error())
}

// EnsureWithin ensures the given repo is within the repos directory.
func EnsureWithin(reposDir string, repo string) error {
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

// EnsureDefaultBranch ensures the repo has a default branch.
// It will prefer choosing "main" or "master" if available.
func EnsureDefaultBranch(ctx context.Context, scmd ServiceCommand) error {
	r, err := git.Open(scmd.Dir)
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
		branch := brs[0]
		// Prefer "main" or "master" as the default branch
		for _, b := range brs {
			if b == "main" || b == "master" {
				branch = b
				break
			}
		}

		cmd := git.NewCommand("branch", "-M", branch).WithContext(ctx)
		if err := cmd.RunInDirWithOptions(scmd.Dir, git.RunInDirOptions{
			Stdin:  scmd.Stdin,
			Stdout: scmd.Stdout,
			Stderr: scmd.Stderr,
		}); err != nil {
			return err
		}
	}
	if err != nil && err != git.ErrReferenceNotExist {
		return err
	}
	return nil
}
