// Package file implements a backend that uses the filesystem to store non-Git related data
//
// The following files and directories are used:
//
//   - anon-access: contains the access level for anonymous users
//   - allow-keyless: contains a boolean value indicating whether or not keyless access is allowed
//   - admins: contains a list of authorized keys for admin users
//   - host: contains the server's server hostname
//   - name: contains the server's name
//   - port: contains the server's port
//   - repos: is a the directory containing all Git repositories
//
// Each repository has the following files and directories:
//   - collaborators: contains a list of authorized keys for collaborators
//   - description: contains the repository's description
//   - private: when present, indicates that the repository is private
//   - git-daemon-export-ok: when present, indicates that the repository is public
//   - project-name: contains the repository's project name
package file

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/charmbracelet/log"
	"github.com/charmbracelet/soft-serve/git"
	"github.com/charmbracelet/soft-serve/server/backend"
	"github.com/charmbracelet/ssh"
	gitm "github.com/gogs/git-module"
	gossh "golang.org/x/crypto/ssh"
)

// sub file and directory names.
const (
	anonAccess   = "anon-access"
	allowKeyless = "allow-keyless"
	admins       = "admins"
	serverHost   = "host"
	serverName   = "name"
	serverPort   = "port"
	repos        = "repos"
	collabs      = "collaborators"
	description  = "description"
	private      = "private"
)

var (
	logger = log.WithPrefix("backend.file")

	defaults = map[string]string{
		serverName:   "Soft Serve",
		serverHost:   "localhost",
		serverPort:   "23231",
		anonAccess:   backend.ReadOnlyAccess.String(),
		allowKeyless: "true",
	}
)

var _ backend.Backend = &FileBackend{}

var _ backend.AccessMethod = &FileBackend{}

// FileBackend is a backend that uses the filesystem.
type FileBackend struct { // nolint:revive
	// path is the path to the directory containing the repositories and config
	// files.
	path string
	// AdditionalAdmins additional admins to the server.
	AdditionalAdmins []string
}

func (fb *FileBackend) reposPath() string {
	return filepath.Join(fb.path, repos)
}

func (fb *FileBackend) adminsPath() string {
	return filepath.Join(fb.path, admins)
}

func (fb *FileBackend) collabsPath(repo string) string {
	return filepath.Join(fb.reposPath(), repo, collabs)
}

func sanatizeRepo(repo string) string {
	return strings.TrimSuffix(repo, ".git")
}

func readOneLine(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close() // nolint:errcheck
	s := bufio.NewScanner(f)
	s.Scan()
	return s.Text(), s.Err()
}

func readAll(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}

	bts, err := io.ReadAll(f)
	return string(bts), err
}

// NewFileBackend creates a new FileBackend.
func NewFileBackend(path string) (*FileBackend, error) {
	fb := &FileBackend{path: path}
	for _, dir := range []string{repos} {
		if err := os.MkdirAll(filepath.Join(path, dir), 0755); err != nil {
			return nil, err
		}
	}
	for _, file := range []string{admins, anonAccess, allowKeyless, serverHost, serverName, serverPort} {
		fp := filepath.Join(path, file)
		_, err := os.Stat(fp)
		if errors.Is(err, fs.ErrNotExist) {
			f, err := os.Create(fp)
			if err != nil {
				return nil, err
			}
			if c, ok := defaults[file]; ok {
				io.WriteString(f, c) // nolint:errcheck
			}
			_ = f.Close()
		}
	}
	return fb, nil
}

// AccessLevel returns the access level for the given public key and repo.
//
// It implements backend.AccessMethod.
func (fb *FileBackend) AccessLevel(repo string, pk gossh.PublicKey) backend.AccessLevel {
	private := fb.IsPrivate(repo)
	anon := fb.AnonAccess()
	if pk != nil {
		// Check if the key is an admin.
		if fb.IsAdmin(pk) {
			return backend.AdminAccess
		}

		// Check if the key is a collaborator.
		if fb.IsCollaborator(pk, repo) {
			if anon > backend.ReadWriteAccess {
				return anon
			}
			return backend.ReadWriteAccess
		}

		// Check if repo is private.
		if !private {
			if anon > backend.ReadOnlyAccess {
				return anon
			}
			return backend.ReadOnlyAccess
		}
	}

	if private {
		return backend.NoAccess
	}

	return anon
}

// AddAdmin adds a public key to the list of server admins.
//
// It implements backend.Backend.
func (fb *FileBackend) AddAdmin(pk gossh.PublicKey) error {
	// Skip if the key already exists.
	if fb.IsAdmin(pk) {
		return nil
	}

	ak := backend.MarshalAuthorizedKey(pk)
	f, err := os.OpenFile(fb.adminsPath(), os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		logger.Debug("failed to open admin keys file", "err", err, "path", fb.adminsPath())
		return err
	}

	defer f.Close() //nolint:errcheck
	_, err = fmt.Fprintln(f, ak)
	return err
}

// AddCollaborator adds a public key to the list of collaborators for the given repo.
//
// It implements backend.Backend.
func (fb *FileBackend) AddCollaborator(pk gossh.PublicKey, repo string) error {
	// Skip if the key already exists.
	if fb.IsCollaborator(pk, repo) {
		return nil
	}

	ak := backend.MarshalAuthorizedKey(pk)
	repo = sanatizeRepo(repo) + ".git"
	f, err := os.OpenFile(fb.collabsPath(repo), os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		logger.Debug("failed to open collaborators file", "err", err, "path", fb.collabsPath(repo))
		return err
	}

	defer f.Close() //nolint:errcheck
	_, err = fmt.Fprintln(f, ak)
	return err
}

// AllowKeyless returns true if keyless access is allowed.
//
// It implements backend.Backend.
func (fb *FileBackend) AllowKeyless() bool {
	line, err := readOneLine(filepath.Join(fb.path, allowKeyless))
	if err != nil {
		logger.Debug("failed to read allow-keyless file", "err", err)
		return false
	}

	return line == "true"
}

// AnonAccess returns the level of anonymous access allowed.
//
// It implements backend.Backend.
func (fb *FileBackend) AnonAccess() backend.AccessLevel {
	line, err := readOneLine(filepath.Join(fb.path, anonAccess))
	if err != nil {
		logger.Debug("failed to read anon-access file", "err", err)
		return backend.NoAccess
	}

	switch line {
	case backend.NoAccess.String():
		return backend.NoAccess
	case backend.ReadOnlyAccess.String():
		return backend.ReadOnlyAccess
	case backend.ReadWriteAccess.String():
		return backend.ReadWriteAccess
	case backend.AdminAccess.String():
		return backend.AdminAccess
	default:
		return backend.NoAccess
	}
}

// Description returns the description of the given repo.
//
// It implements backend.Backend.
func (fb *FileBackend) Description(repo string) string {
	repo = sanatizeRepo(repo) + ".git"
	r := &Repo{path: filepath.Join(fb.reposPath(), repo)}
	return r.Description()
}

// IsAdmin checks if the given public key is a server admin.
//
// It implements backend.Backend.
func (fb *FileBackend) IsAdmin(pk gossh.PublicKey) bool {
	// Check if the key is an additional admin.
	ak := backend.MarshalAuthorizedKey(pk)
	for _, admin := range fb.AdditionalAdmins {
		if ak == admin {
			return true
		}
	}

	f, err := os.Open(fb.adminsPath())
	if err != nil {
		logger.Debug("failed to open admins file", "err", err, "path", fb.adminsPath())
		return false
	}

	defer f.Close() //nolint:errcheck
	s := bufio.NewScanner(f)
	for s.Scan() {
		apk, _, err := backend.ParseAuthorizedKey(s.Text())
		if err != nil {
			continue
		}
		if ssh.KeysEqual(apk, pk) {
			return true
		}
	}

	return false
}

// IsCollaborator returns true if the given public key is a collaborator on the
// given repo.
//
// It implements backend.Backend.
func (fb *FileBackend) IsCollaborator(pk gossh.PublicKey, repo string) bool {
	repo = sanatizeRepo(repo) + ".git"
	f, err := os.Open(fb.collabsPath(repo))
	if err != nil {
		logger.Debug("failed to open collaborators file", "err", err, "path", fb.collabsPath(repo))
		return false
	}

	defer f.Close() //nolint:errcheck
	s := bufio.NewScanner(f)
	for s.Scan() {
		apk, _, err := backend.ParseAuthorizedKey(s.Text())
		if err != nil {
			continue
		}
		if ssh.KeysEqual(apk, pk) {
			return true
		}
	}

	return false
}

// IsPrivate returns true if the given repo is private.
//
// It implements backend.Backend.
func (fb *FileBackend) IsPrivate(repo string) bool {
	repo = sanatizeRepo(repo) + ".git"
	r := &Repo{path: filepath.Join(fb.reposPath(), repo)}
	return r.IsPrivate()
}

// ServerHost returns the server host.
//
// It implements backend.Backend.
func (fb *FileBackend) ServerHost() string {
	line, err := readOneLine(filepath.Join(fb.path, serverHost))
	if err != nil {
		logger.Debug("failed to read server-host file", "err", err)
		return ""
	}

	return line
}

// ServerName returns the server name.
//
// It implements backend.Backend.
func (fb *FileBackend) ServerName() string {
	line, err := readOneLine(filepath.Join(fb.path, serverName))
	if err != nil {
		logger.Debug("failed to read server-name file", "err", err)
		return ""
	}

	return line
}

// ServerPort returns the server port.
//
// It implements backend.Backend.
func (fb *FileBackend) ServerPort() string {
	line, err := readOneLine(filepath.Join(fb.path, serverPort))
	if err != nil {
		logger.Debug("failed to read server-port file", "err", err)
		return ""
	}

	if _, err := strconv.Atoi(line); err != nil {
		logger.Debug("failed to parse server-port file", "err", err)
		return ""
	}

	return line
}

// SetAllowKeyless sets whether or not to allow keyless access.
//
// It implements backend.Backend.
func (fb *FileBackend) SetAllowKeyless(allow bool) error {
	f, err := os.OpenFile(filepath.Join(fb.path, allowKeyless), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("failed to open allow-keyless file: %w", err)
	}

	defer f.Close() //nolint:errcheck
	_, err = fmt.Fprintln(f, allow)
	return err
}

// SetAnonAccess sets the anonymous access level.
//
// It implements backend.Backend.
func (fb *FileBackend) SetAnonAccess(level backend.AccessLevel) error {
	f, err := os.OpenFile(filepath.Join(fb.path, anonAccess), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("failed to open anon-access file: %w", err)
	}

	defer f.Close() //nolint:errcheck
	_, err = fmt.Fprintln(f, level.String())
	return err
}

// SetDescription sets the description of the given repo.
//
// It implements backend.Backend.
func (fb *FileBackend) SetDescription(repo string, desc string) error {
	f, err := os.OpenFile(filepath.Join(fb.reposPath(), repo, description), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("failed to open description file: %w", err)
	}

	defer f.Close() //nolint:errcheck
	_, err = fmt.Fprintln(f, desc)
	return err
}

// SetPrivate sets the private status of the given repo.
//
// It implements backend.Backend.
func (fb *FileBackend) SetPrivate(repo string, priv bool) error {
	repo = sanatizeRepo(repo) + ".git"
	daemonExport := filepath.Join(fb.reposPath(), repo, "git-daemon-export-ok")
	if priv {
		_ = os.Remove(daemonExport)
		f, err := os.Create(filepath.Join(fb.reposPath(), repo, private))
		if err != nil {
			return fmt.Errorf("failed to create private file: %w", err)
		}

		_ = f.Close() //nolint:errcheck
	} else {
		// Create git-daemon-export-ok file if repo is public.
		f, err := os.Create(daemonExport)
		if err != nil {
			logger.Warn("failed to create git-daemon-export-ok file", "err", err)
		} else {
			_ = f.Close() //nolint:errcheck
		}
	}
	return nil
}

// SetServerHost sets the server host.
//
// It implements backend.Backend.
func (fb *FileBackend) SetServerHost(host string) error {
	f, err := os.Create(filepath.Join(fb.path, serverHost))
	if err != nil {
		return fmt.Errorf("failed to create server-host file: %w", err)
	}

	defer f.Close() //nolint:errcheck
	_, err = fmt.Fprintln(f, host)
	return err
}

// SetServerName sets the server name.
//
// It implements backend.Backend.
func (fb *FileBackend) SetServerName(name string) error {
	f, err := os.Create(filepath.Join(fb.path, serverName))
	if err != nil {
		return fmt.Errorf("failed to create server-name file: %w", err)
	}

	defer f.Close() //nolint:errcheck
	_, err = fmt.Fprintln(f, name)
	return err
}

// SetServerPort sets the server port.
//
// It implements backend.Backend.
func (fb *FileBackend) SetServerPort(port string) error {
	f, err := os.Create(filepath.Join(fb.path, serverPort))
	if err != nil {
		return fmt.Errorf("failed to create server-port file: %w", err)
	}

	defer f.Close() //nolint:errcheck
	_, err = fmt.Fprintln(f, port)
	return err
}

// CreateRepository creates a new repository.
//
// Created repositories are always bare.
//
// It implements backend.Backend.
func (fb *FileBackend) CreateRepository(name string, private bool) (backend.Repository, error) {
	name = sanatizeRepo(name) + ".git"
	rp := filepath.Join(fb.reposPath(), name)
	if _, err := git.Init(rp, true); err != nil {
		logger.Debug("failed to create repository", "err", err)
		return nil, err
	}

	fb.SetPrivate(name, private)

	return &Repo{path: rp}, nil
}

// DeleteRepository deletes the given repository.
//
// It implements backend.Backend.
func (fb *FileBackend) DeleteRepository(name string) error {
	name = sanatizeRepo(name) + ".git"
	return os.RemoveAll(filepath.Join(fb.reposPath(), name))
}

// RenameRepository renames the given repository.
//
// It implements backend.Backend.
func (fb *FileBackend) RenameRepository(oldName string, newName string) error {
	oldName = sanatizeRepo(oldName) + ".git"
	newName = sanatizeRepo(newName) + ".git"
	return os.Rename(filepath.Join(fb.reposPath(), oldName), filepath.Join(fb.reposPath(), newName))
}

// Repository finds the given repository.
//
// It implements backend.Backend.
func (fb *FileBackend) Repository(repo string) (backend.Repository, error) {
	repo = sanatizeRepo(repo) + ".git"
	rp := filepath.Join(fb.reposPath(), repo)
	_, err := os.Stat(rp)
	if !errors.Is(err, os.ErrExist) {
		return nil, err
	}

	return &Repo{path: rp}, nil
}

// Repositories returns a list of all repositories.
//
// It implements backend.Backend.
func (fb *FileBackend) Repositories() ([]backend.Repository, error) {
	repos := make([]backend.Repository, 0)
	err := filepath.WalkDir(fb.reposPath(), func(path string, d fs.DirEntry, _ error) error {
		// Skip non-directories.
		if !d.IsDir() {
			return nil
		}

		// Skip non-repositories.
		if !strings.HasSuffix(path, ".git") {
			return nil
		}

		repos = append(repos, &Repo{path: path})

		return nil
	})
	if err != nil {
		return nil, err
	}

	return repos, nil
}

// DefaultBranch returns the default branch of the given repository.
//
// It implements backend.Backend.
func (fb *FileBackend) DefaultBranch(repo string) (string, error) {
	rr, err := fb.Repository(repo)
	if err != nil {
		logger.Debug("failed to get default branch", "err", err)
		return "", err
	}

	r, err := rr.Repository()
	if err != nil {
		logger.Debug("failed to open repository for default branch", "err", err)
		return "", err
	}

	head, err := r.HEAD()
	if err != nil {
		logger.Debug("failed to get HEAD for default branch", "err", err)
		return "", err
	}

	return head.Name().Short(), nil
}

// SetDefaultBranch sets the default branch for the given repository.
//
// It implements backend.Backend.
func (fb *FileBackend) SetDefaultBranch(repo string, branch string) error {
	rr, err := fb.Repository(repo)
	if err != nil {
		logger.Debug("failed to get repository for default branch", "err", err)
		return err
	}

	r, err := rr.Repository()
	if err != nil {
		logger.Debug("failed to open repository for default branch", "err", err)
		return err
	}

	if _, err := r.SymbolicRef("HEAD", gitm.RefsHeads+branch); err != nil {
		logger.Debug("failed to set default branch", "err", err)
		return err
	}

	return nil
}
