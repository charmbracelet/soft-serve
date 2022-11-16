package git

import (
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/soft-serve/git"
	"github.com/charmbracelet/wish"
	"github.com/gliderlabs/ssh"
)

// ErrNotAuthed represents unauthorized access.
var ErrNotAuthed = errors.New("you are not authorized to do this")

// ErrSystemMalfunction represents a general system error returned to clients.
var ErrSystemMalfunction = errors.New("something went wrong")

// ErrInvalidRepo represents an attempt to access a non-existent repo.
var ErrInvalidRepo = errors.New("invalid repo")

// AccessLevel is the level of access allowed to a repo.
type AccessLevel int

const (
	// NoAccess does not allow access to the repo.
	NoAccess AccessLevel = iota

	// ReadOnlyAccess allows read-only access to the repo.
	ReadOnlyAccess

	// ReadWriteAccess allows read and write access to the repo.
	ReadWriteAccess

	// AdminAccess allows read, write, and admin access to the repo.
	AdminAccess
)

// String implements the Stringer interface for AccessLevel.
func (a AccessLevel) String() string {
	switch a {
	case NoAccess:
		return "no-access"
	case ReadOnlyAccess:
		return "read-only"
	case ReadWriteAccess:
		return "read-write"
	case AdminAccess:
		return "admin-access"
	default:
		return ""
	}
}

// Hooks is an interface that allows for custom authorization
// implementations and post push/fetch notifications. Prior to git access,
// AuthRepo will be called with the ssh.Session public key and the repo name.
// Implementers return the appropriate AccessLevel.
type Hooks interface {
	AuthRepo(string, ssh.PublicKey) AccessLevel
	Push(string, ssh.PublicKey)
	Fetch(string, ssh.PublicKey)
}

// Middleware adds Git server functionality to the ssh.Server. Repos are stored
// in the specified repo directory. The provided Hooks implementation will be
// checked for access on a per repo basis for a ssh.Session public key.
// Hooks.Push and Hooks.Fetch will be called on successful completion of
// their commands.
func Middleware(repoDir string, gh Hooks) wish.Middleware {
	return func(sh ssh.Handler) ssh.Handler {
		return func(s ssh.Session) {
			func() {
				cmd := s.Command()
				if len(cmd) == 2 {
					gc := cmd[0]
					// repo should be in the form of "repo.git"
					repo := strings.TrimPrefix(cmd[1], "/")
					repo = filepath.Clean(repo)
					if strings.Contains(repo, "/") {
						log.Printf("invalid repo: %s", repo)
						Fatal(s, fmt.Errorf("%s: %s", ErrInvalidRepo, "user repos not supported"))
						return
					}
					pk := s.PublicKey()
					access := gh.AuthRepo(strings.TrimSuffix(repo, ".git"), pk)
					// git bare repositories should end in ".git"
					// https://git-scm.com/docs/gitrepository-layout
					if !strings.HasSuffix(repo, ".git") {
						repo += ".git"
					}
					switch gc {
					case "git-receive-pack":
						switch access {
						case ReadWriteAccess, AdminAccess:
							err := gitPack(s, gc, repoDir, repo)
							if err != nil {
								Fatal(s, ErrSystemMalfunction)
							} else {
								gh.Push(repo, pk)
							}
						default:
							Fatal(s, ErrNotAuthed)
						}
						return
					case "git-upload-archive", "git-upload-pack":
						switch access {
						case ReadOnlyAccess, ReadWriteAccess, AdminAccess:
							// try to upload <repo>.git first, then <repo>
							err := gitPack(s, gc, repoDir, repo)
							if err != nil {
								err = gitPack(s, gc, repoDir, strings.TrimSuffix(repo, ".git"))
							}
							switch err {
							case ErrInvalidRepo:
								Fatal(s, ErrInvalidRepo)
							case nil:
								gh.Fetch(repo, pk)
							default:
								log.Printf("unknown git error: %s", err)
								Fatal(s, ErrSystemMalfunction)
							}
						default:
							Fatal(s, ErrNotAuthed)
						}
						return
					}
				}
			}()
			sh(s)
		}
	}
}

func gitPack(s ssh.Session, gitCmd string, repoDir string, repo string) error {
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
		return runGit(s, "", cmd, rp)
	case "git-receive-pack":
		err := ensureRepo(repoDir, repo)
		if err != nil {
			return err
		}
		err = runGit(s, "", cmd, rp)
		if err != nil {
			return err
		}
		err = ensureDefaultBranch(s, rp)
		if err != nil {
			return err
		}
		// Needed for git dumb http server
		return runGit(s, rp, "update-server-info")
	default:
		return fmt.Errorf("unknown git command: %s", gitCmd)
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

// Fatal prints to the session's STDOUT as a git response and exit 1.
func Fatal(s ssh.Session, v ...interface{}) {
	msg := fmt.Sprint(v...)
	// hex length includes 4 byte length prefix and ending newline
	pktLine := fmt.Sprintf("%04x%s\n", len(msg)+5, msg)
	_, _ = wish.WriteString(s, pktLine)
	s.Exit(1) // nolint: errcheck
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

func runGit(s ssh.Session, dir string, args ...string) error {
	usi := exec.CommandContext(s.Context(), "git", args...)
	usi.Dir = dir
	usi.Stdout = s
	usi.Stdin = s
	if err := usi.Run(); err != nil {
		return err
	}
	return nil
}

func ensureDefaultBranch(s ssh.Session, repoPath string) error {
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
		err = runGit(s, repoPath, "branch", "-M", brs[0])
		if err != nil {
			return err
		}
	}
	if err != nil && err != git.ErrReferenceNotExist {
		return err
	}
	return nil
}
