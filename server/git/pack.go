package git

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/soft-serve/git"
	"github.com/go-git/go-git/v5/plumbing/format/pktline"
)

// GitPack runs the git pack protocol against the provided repo.
func GitPack(out io.Writer, in io.Reader, er io.Writer, gitCmd string, repoDir string, repo string) error {
	cmd := strings.TrimPrefix(gitCmd, "git-")
	rp := filepath.Join(repoDir, repo)
	switch gitCmd {
	case "git-upload-archive", "git-upload-pack":
		exists, err := fileExists(rp)
		if !exists {
			return ErrInvalidRepo
		}
		if err != nil {
			return err
		}
		return RunGit(out, in, er, "", cmd, rp)
	case "git-receive-pack":
		err := ensureRepo(repoDir, repo)
		if err != nil {
			return err
		}
		err = RunGit(out, in, er, "", cmd, rp)
		if err != nil {
			return err
		}
		err = ensureDefaultBranch(out, in, er, rp)
		if err != nil {
			return err
		}
		// Needed for git dumb http server
		return RunGit(out, in, er, rp, "update-server-info")
	default:
		return fmt.Errorf("unknown git command: %s", gitCmd)
	}
}

// RunGit runs a git command in the given repo.
func RunGit(out io.Writer, in io.Reader, err io.Writer, dir string, args ...string) error {
	c := git.NewCommand(args...)
	return c.RunInDirWithOptions(dir, git.RunInDirOptions{
		Stdout: out,
		Stdin:  in,
		Stderr: err,
	})
}

// WritePktline encodes and writes a pktline to the given writer.
func WritePktline(w io.Writer, v ...interface{}) {
	msg := fmt.Sprint(v...)
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
	if !exists {
		_, err := git.Init(rp, true)
		if err != nil {
			return err
		}
	}
	return nil
}

func ensureDefaultBranch(out io.Writer, in io.Reader, er io.Writer, repoPath string) error {
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
		err = RunGit(out, in, er, repoPath, "branch", "-M", brs[0])
		if err != nil {
			return err
		}
	}
	if err != nil && err != git.ErrReferenceNotExist {
		return err
	}
	return nil
}
