package server

import (
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

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
	ReceivePackBin   = "git-receive-pack"
	UploadPackBin    = "git-upload-pack"
	UploadArchiveBin = "git-upload-archive"
)

// UploadPack runs the git upload-pack protocol against the provided repo.
func UploadPack(in io.Reader, out io.Writer, er io.Writer, repoDir string) error {
	exists, err := fileExists(repoDir)
	if !exists {
		return ErrInvalidRepo
	}
	if err != nil {
		return err
	}
	return RunGit(in, out, er, "", UploadPackBin[4:], repoDir)
}

// UploadArchive runs the git upload-archive protocol against the provided repo.
func UploadArchive(in io.Reader, out io.Writer, er io.Writer, repoDir string) error {
	exists, err := fileExists(repoDir)
	if !exists {
		return ErrInvalidRepo
	}
	if err != nil {
		return err
	}
	return RunGit(in, out, er, "", UploadArchiveBin[4:], repoDir)
}

// ReceivePack runs the git receive-pack protocol against the provided repo.
func ReceivePack(in io.Reader, out io.Writer, er io.Writer, repoDir string) error {
	if err := ensureRepo(repoDir, ""); err != nil {
		return err
	}
	if err := RunGit(in, out, er, "", ReceivePackBin[4:], repoDir); err != nil {
		return err
	}
	return ensureDefaultBranch(in, out, er, repoDir)
}

// RunGit runs a git command in the given repo.
func RunGit(in io.Reader, out io.Writer, err io.Writer, dir string, args ...string) error {
	c := git.NewCommand(args...)
	return c.RunInDirWithOptions(dir, git.RunInDirOptions{
		Stdin:  in,
		Stdout: out,
		Stderr: err,
	})
}

// WritePktline encodes and writes a pktline to the given writer.
func WritePktline(w io.Writer, v ...interface{}) {
	msg := fmt.Sprintln(v...)
	pkt := pktline.NewEncoder(w)
	if err := pkt.EncodeString(msg); err != nil {
		log.Printf("git: error writing pkt-line message: %s", err)
	}
	if err := pkt.Flush(); err != nil {
		log.Printf("git: error flushing pkt-line message: %s", err)
	}
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

func ensureRepo(dir string, repo string) error {
	exists, err := fileExists(dir)
	if err != nil {
		return err
	}
	if !exists {
		err = os.MkdirAll(dir, os.ModeDir|os.FileMode(0700))
		if err != nil {
			return err
		}
	}
	rp := filepath.Join(dir, repo)
	exists, err = fileExists(rp)
	if err != nil {
		return err
	}
	// FIXME: use backend.CreateRepository
	if !exists {
		_, err := git.Init(rp, true)
		if err != nil {
			return err
		}
	}
	return nil
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
		// FIXME: use backend.SetDefaultBranch
		err = RunGit(in, out, er, repoPath, "branch", "-M", brs[0])
		if err != nil {
			return err
		}
	}
	if err != nil && err != git.ErrReferenceNotExist {
		return err
	}
	return nil
}
