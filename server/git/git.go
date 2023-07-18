package git

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/log"
	"github.com/charmbracelet/soft-serve/git"
	"github.com/charmbracelet/soft-serve/server/config"
	"github.com/go-git/go-git/v5/plumbing/format/pktline"
	"golang.org/x/sync/errgroup"
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
	ReceivePackBin   = "git-receive-pack"
	UploadPackBin    = "git-upload-pack"
	UploadArchiveBin = "git-upload-archive"
)

// UploadPack runs the git upload-pack protocol against the provided repo.
func UploadPack(ctx context.Context, in io.Reader, out io.Writer, er io.Writer, repoDir string, envs ...string) error {
	exists, err := fileExists(repoDir)
	if !exists {
		return ErrInvalidRepo
	}
	if err != nil {
		return err
	}
	return RunGit(ctx, in, out, er, "", envs, UploadPackBin[4:], repoDir)
}

// UploadArchive runs the git upload-archive protocol against the provided repo.
func UploadArchive(ctx context.Context, in io.Reader, out io.Writer, er io.Writer, repoDir string, envs ...string) error {
	exists, err := fileExists(repoDir)
	if !exists {
		return ErrInvalidRepo
	}
	if err != nil {
		return err
	}
	return RunGit(ctx, in, out, er, "", envs, UploadArchiveBin[4:], repoDir)
}

// ReceivePack runs the git receive-pack protocol against the provided repo.
func ReceivePack(ctx context.Context, in io.Reader, out io.Writer, er io.Writer, repoDir string, envs ...string) error {
	if err := RunGit(ctx, in, out, er, "", envs, ReceivePackBin[4:], repoDir); err != nil {
		return err
	}
	return EnsureDefaultBranch(ctx, in, out, er, repoDir)
}

// RunGit runs a git command in the given repo.
func RunGit(ctx context.Context, in io.Reader, out io.Writer, er io.Writer, dir string, envs []string, args ...string) error {
	cfg := config.FromContext(ctx)
	logger := log.FromContext(ctx).WithPrefix("rungit")
	c := exec.CommandContext(ctx, "git", args...)
	c.Dir = dir
	c.Env = append(os.Environ(), envs...)
	c.Env = append(c.Env, "PATH="+os.Getenv("PATH"))
	c.Env = append(c.Env, "SOFT_SERVE_DEBUG="+os.Getenv("SOFT_SERVE_DEBUG"))
	if cfg != nil {
		c.Env = append(c.Env, "SOFT_SERVE_LOG_FORMAT="+cfg.Log.Format)
		c.Env = append(c.Env, "SOFT_SERVE_LOG_TIME_FORMAT="+cfg.Log.TimeFormat)
	}

	stdin, err := c.StdinPipe()
	if err != nil {
		logger.Error("failed to get stdin pipe", "err", err)
		return err
	}

	stdout, err := c.StdoutPipe()
	if err != nil {
		logger.Error("failed to get stdout pipe", "err", err)
		return err
	}

	stderr, err := c.StderrPipe()
	if err != nil {
		logger.Error("failed to get stderr pipe", "err", err)
		return err
	}

	if err := c.Start(); err != nil {
		logger.Error("failed to start command", "err", err)
		return err
	}

	errg, ctx := errgroup.WithContext(ctx)

	// stdin
	errg.Go(func() error {
		defer stdin.Close()

		_, err := io.Copy(stdin, in)
		return err
	})

	// stdout
	errg.Go(func() error {
		_, err := io.Copy(out, stdout)
		return err
	})

	// stderr
	errg.Go(func() error {
		_, err := io.Copy(er, stderr)
		return err
	})

	if err := errg.Wait(); err != nil {
		logger.Error("while copying output", "err", err)
	}

	// Wait for the command to finish
	return c.Wait()
}

// WritePktline encodes and writes a pktline to the given writer.
func WritePktline(w io.Writer, v ...interface{}) {
	msg := fmt.Sprintln(v...)
	pkt := pktline.NewEncoder(w)
	if err := pkt.EncodeString(msg); err != nil {
		log.Debugf("git: error writing pkt-line message: %s", err)
	}
	if err := pkt.Flush(); err != nil {
		log.Debugf("git: error flushing pkt-line message: %s", err)
	}
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

func EnsureDefaultBranch(ctx context.Context, in io.Reader, out io.Writer, er io.Writer, repoPath string) error {
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
		if _, err := r.SymbolicRef(git.HEAD, git.RefsHeads+brs[0]); err != nil {
			return err
		}
	}
	if err != nil && err != git.ErrReferenceNotExist {
		return err
	}
	return nil
}
