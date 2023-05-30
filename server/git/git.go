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
	"time"

	"github.com/charmbracelet/log"
	"github.com/charmbracelet/soft-serve/git"
	"github.com/charmbracelet/soft-serve/server/config"
	"github.com/go-git/go-git/v5/plumbing/format/pktline"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
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

var uploadPackCounter = promauto.NewCounter(prometheus.CounterOpts{
	Namespace: "soft_serve",
	Subsystem: "git",
	Name:      "upload_pack_total",
	Help:      "Total times git-upload-pack was run",
})

var uploadPackDuration = promauto.NewCounter(prometheus.CounterOpts{
	Namespace: "soft_serve",
	Subsystem: "git",
	Name:      "upload_pack_seconds_total",
	Help:      "Total time spent running git-upload-pack was run",
})

var uploadArchiveCounter = promauto.NewCounter(prometheus.CounterOpts{
	Namespace: "soft_serve",
	Subsystem: "git",
	Name:      "upload_archive_total",
	Help:      "Total times git-upload-archive was run",
})

var uploadArchiveDuration = promauto.NewCounter(prometheus.CounterOpts{
	Namespace: "soft_serve",
	Subsystem: "git",
	Name:      "upload_archive_seconds_total",
	Help:      "Total time spent running git-upload-archive was run",
})

var receivePackCounter = promauto.NewCounter(prometheus.CounterOpts{
	Namespace: "soft_serve",
	Subsystem: "git",
	Name:      "receive_pack_total",
	Help:      "Total times git-receive-pack was run",
})

var receivePackDuration = promauto.NewCounter(prometheus.CounterOpts{
	Namespace: "soft_serve",
	Subsystem: "git",
	Name:      "receive_pack_seconds_total",
	Help:      "Total time spent running git-receive-pack was run",
})

// Git protocol commands.
const (
	ReceivePackBin   = "git-receive-pack"
	UploadPackBin    = "git-upload-pack"
	UploadArchiveBin = "git-upload-archive"
)

// UploadPack runs the git upload-pack protocol against the provided repo.
func UploadPack(ctx context.Context, in io.Reader, out io.Writer, er io.Writer, repoDir string, envs ...string) error {
	start := time.Now()
	defer uploadPackDuration.Add(time.Since(start).Seconds())
	uploadPackCounter.Inc()
	exists, err := fileExists(repoDir)
	if !exists {
		return ErrInvalidRepo
	}
	if err != nil {
		return err
	}
	return runGit(ctx, in, out, er, "", envs, UploadPackBin[4:], repoDir)
}

// UploadArchive runs the git upload-archive protocol against the provided repo.
func UploadArchive(ctx context.Context, in io.Reader, out io.Writer, er io.Writer, repoDir string, envs ...string) error {
	start := time.Now()
	defer uploadArchiveDuration.Add(time.Since(start).Seconds())
	uploadArchiveCounter.Inc()
	exists, err := fileExists(repoDir)
	if !exists {
		return ErrInvalidRepo
	}
	if err != nil {
		return err
	}
	return runGit(ctx, in, out, er, "", envs, UploadArchiveBin[4:], repoDir)
}

// ReceivePack runs the git receive-pack protocol against the provided repo.
func ReceivePack(ctx context.Context, in io.Reader, out io.Writer, er io.Writer, repoDir string, envs ...string) error {
	start := time.Now()
	defer receivePackDuration.Add(time.Since(start).Seconds())
	receivePackCounter.Inc()
	if err := runGit(ctx, in, out, er, "", envs, ReceivePackBin[4:], repoDir); err != nil {
		return err
	}
	return ensureDefaultBranch(ctx, in, out, er, repoDir)
}

// runGit runs a git command in the given repo.
func runGit(ctx context.Context, in io.Reader, out io.Writer, er io.Writer, dir string, envs []string, args ...string) error {
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

	errg, _ := errgroup.WithContext(ctx)

	// stdin
	errg.Go(func() error {
		defer stdin.Close() // nolint:errcheck

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

func ensureDefaultBranch(ctx context.Context, in io.Reader, out io.Writer, er io.Writer, repoPath string) error {
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
		err = runGit(ctx, in, out, er, repoPath, []string{}, "branch", "-M", brs[0])
		if err != nil {
			return err
		}
	}
	if err != nil && err != git.ErrReferenceNotExist {
		return err
	}
	return nil
}
