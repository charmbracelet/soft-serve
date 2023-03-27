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
	"strings"

	"github.com/charmbracelet/log"
	"github.com/charmbracelet/soft-serve/git"
	"github.com/charmbracelet/soft-serve/server/backend"
	"github.com/charmbracelet/ssh"
	gossh "golang.org/x/crypto/ssh"
)

// sub file and directory names.
const (
	anonAccess   = "anon-access"
	allowKeyless = "allow-keyless"
	admins       = "admins"
	repos        = "repos"
	collabs      = "collaborators"
	description  = "description"
	exportOk     = "git-daemon-export-ok"
	private      = "private"
	settings     = "settings"
)

var (
	logger = log.WithPrefix("backend.file")

	defaults = map[string]string{
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

	// repos is a map of repositories.
	repos map[string]*Repo

	// AdditionalAdmins additional admins to the server.
	AdditionalAdmins []string
}

func (fb *FileBackend) reposPath() string {
	return filepath.Join(fb.path, repos)
}

// RepositoryStorePath returns the path to the repository store.
func (fb *FileBackend) RepositoryStorePath() string {
	return fb.reposPath()
}

func (fb *FileBackend) settingsPath() string {
	return filepath.Join(fb.path, settings)
}

func (fb *FileBackend) adminsPath() string {
	return filepath.Join(fb.settingsPath(), admins)
}

func (fb *FileBackend) collabsPath(repo string) string {
	return filepath.Join(fb.path, collabs, repo, collabs)
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

// exists returns true if the given path exists.
func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// NewFileBackend creates a new FileBackend.
func NewFileBackend(path string) (*FileBackend, error) {
	fb := &FileBackend{path: path}
	for _, dir := range []string{repos, settings, collabs} {
		if err := os.MkdirAll(filepath.Join(path, dir), 0755); err != nil {
			return nil, err
		}
	}

	for _, file := range []string{admins, anonAccess, allowKeyless} {
		fp := filepath.Join(fb.settingsPath(), file)
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

	if err := fb.initRepos(); err != nil {
		return nil, err
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
func (fb *FileBackend) AddAdmin(pk gossh.PublicKey, memo string) error {
	// Skip if the key already exists.
	if fb.IsAdmin(pk) {
		return fmt.Errorf("key already exists")
	}

	ak := backend.MarshalAuthorizedKey(pk)
	f, err := os.OpenFile(fb.adminsPath(), os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		logger.Debug("failed to open admin keys file", "err", err, "path", fb.adminsPath())
		return err
	}

	defer f.Close() //nolint:errcheck
	if memo != "" {
		memo = " " + memo
	}
	_, err = fmt.Fprintf(f, "%s%s\n", ak, memo)
	return err
}

// AddCollaborator adds a public key to the list of collaborators for the given repo.
//
// It implements backend.Backend.
func (fb *FileBackend) AddCollaborator(pk gossh.PublicKey, memo string, repo string) error {
	name := sanatizeRepo(repo)
	repo = name + ".git"
	// Check if repo exists
	if !exists(filepath.Join(fb.reposPath(), repo)) {
		return fmt.Errorf("repository %s does not exist", repo)
	}

	// Skip if the key already exists.
	if fb.IsCollaborator(pk, repo) {
		return fmt.Errorf("key already exists")
	}

	ak := backend.MarshalAuthorizedKey(pk)
	if err := os.MkdirAll(filepath.Dir(fb.collabsPath(repo)), 0755); err != nil {
		logger.Debug("failed to create collaborators directory",
			"err", err, "path", filepath.Dir(fb.collabsPath(repo)))
		return err
	}

	f, err := os.OpenFile(fb.collabsPath(repo), os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		logger.Debug("failed to open collaborators file", "err", err, "path", fb.collabsPath(repo))
		return err
	}

	defer f.Close() //nolint:errcheck
	if memo != "" {
		memo = " " + memo
	}
	_, err = fmt.Fprintf(f, "%s%s\n", ak, memo)
	return err
}

// Admins returns a list of public keys that are admins.
//
// It implements backend.Backend.
func (fb *FileBackend) Admins() ([]string, error) {
	admins := make([]string, 0)
	f, err := os.Open(fb.adminsPath())
	if err != nil {
		logger.Debug("failed to open admin keys file", "err", err, "path", fb.adminsPath())
		return nil, err
	}

	defer f.Close() //nolint:errcheck
	s := bufio.NewScanner(f)
	for s.Scan() {
		admins = append(admins, s.Text())
	}

	return admins, s.Err()
}

// Collaborators returns a list of public keys that are collaborators for the given repo.
//
// It implements backend.Backend.
func (fb *FileBackend) Collaborators(repo string) ([]string, error) {
	name := sanatizeRepo(repo)
	repo = name + ".git"
	// Check if repo exists
	if !exists(filepath.Join(fb.reposPath(), repo)) {
		return nil, fmt.Errorf("repository %s does not exist", repo)
	}

	collabs := make([]string, 0)
	f, err := os.Open(fb.collabsPath(repo))
	if err != nil && errors.Is(err, os.ErrNotExist) {
		return collabs, nil
	}
	if err != nil {
		logger.Debug("failed to open collaborators file", "err", err, "path", fb.collabsPath(repo))
		return nil, err
	}

	defer f.Close() //nolint:errcheck
	s := bufio.NewScanner(f)
	for s.Scan() {
		collabs = append(collabs, s.Text())
	}

	return collabs, s.Err()
}

// RemoveAdmin removes a public key from the list of server admins.
//
// It implements backend.Backend.
func (fb *FileBackend) RemoveAdmin(pk gossh.PublicKey) error {
	f, err := os.OpenFile(fb.adminsPath(), os.O_RDWR, 0644)
	if err != nil {
		logger.Debug("failed to open admin keys file", "err", err, "path", fb.adminsPath())
		return err
	}

	defer f.Close() //nolint:errcheck
	s := bufio.NewScanner(f)
	lines := make([]string, 0)
	for s.Scan() {
		apk, _, err := backend.ParseAuthorizedKey(s.Text())
		if err != nil {
			logger.Debug("failed to parse admin key", "err", err, "path", fb.adminsPath())
			continue
		}

		if !ssh.KeysEqual(apk, pk) {
			lines = append(lines, s.Text())
		}
	}

	if err := s.Err(); err != nil {
		logger.Debug("failed to scan admin keys file", "err", err, "path", fb.adminsPath())
		return err
	}

	if err := f.Truncate(0); err != nil {
		logger.Debug("failed to truncate admin keys file", "err", err, "path", fb.adminsPath())
		return err
	}

	if _, err := f.Seek(0, 0); err != nil {
		logger.Debug("failed to seek admin keys file", "err", err, "path", fb.adminsPath())
		return err
	}

	w := bufio.NewWriter(f)
	for _, line := range lines {
		if _, err := fmt.Fprintln(w, line); err != nil {
			logger.Debug("failed to write admin keys file", "err", err, "path", fb.adminsPath())
			return err
		}
	}

	return w.Flush()
}

// RemoveCollaborator removes a public key from the list of collaborators for the given repo.
//
// It implements backend.Backend.
func (fb *FileBackend) RemoveCollaborator(pk gossh.PublicKey, repo string) error {
	name := sanatizeRepo(repo)
	repo = name + ".git"
	// Check if repo exists
	if !exists(filepath.Join(fb.reposPath(), repo)) {
		return fmt.Errorf("repository %s does not exist", repo)
	}

	f, err := os.OpenFile(fb.collabsPath(repo), os.O_RDWR, 0644)
	if err != nil && errors.Is(err, os.ErrNotExist) {
		return nil
	}

	if err != nil {
		logger.Debug("failed to open collaborators file", "err", err, "path", fb.collabsPath(repo))
		return err
	}

	defer f.Close() //nolint:errcheck
	s := bufio.NewScanner(f)
	lines := make([]string, 0)
	for s.Scan() {
		apk, _, err := backend.ParseAuthorizedKey(s.Text())
		if err != nil {
			logger.Debug("failed to parse collaborator key", "err", err, "path", fb.collabsPath(repo))
			continue
		}

		if !ssh.KeysEqual(apk, pk) {
			lines = append(lines, s.Text())
		}
	}

	if err := s.Err(); err != nil {
		logger.Debug("failed to scan collaborators file", "err", err, "path", fb.collabsPath(repo))
		return err
	}

	if err := f.Truncate(0); err != nil {
		logger.Debug("failed to truncate collaborators file", "err", err, "path", fb.collabsPath(repo))
		return err
	}

	if _, err := f.Seek(0, 0); err != nil {
		logger.Debug("failed to seek collaborators file", "err", err, "path", fb.collabsPath(repo))
		return err
	}

	w := bufio.NewWriter(f)
	for _, line := range lines {
		if _, err := fmt.Fprintln(w, line); err != nil {
			logger.Debug("failed to write collaborators file", "err", err, "path", fb.collabsPath(repo))
			return err
		}
	}

	return w.Flush()
}

// AllowKeyless returns true if keyless access is allowed.
//
// It implements backend.Backend.
func (fb *FileBackend) AllowKeyless() bool {
	line, err := readOneLine(filepath.Join(fb.settingsPath(), allowKeyless))
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
	line, err := readOneLine(filepath.Join(fb.settingsPath(), anonAccess))
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
	r := &Repo{path: filepath.Join(fb.reposPath(), repo), root: fb.reposPath()}
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
	_, err := os.Stat(fb.collabsPath(repo))
	if err != nil {
		return false
	}

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
	r := &Repo{path: filepath.Join(fb.reposPath(), repo), root: fb.reposPath()}
	return r.IsPrivate()
}

// SetAllowKeyless sets whether or not to allow keyless access.
//
// It implements backend.Backend.
func (fb *FileBackend) SetAllowKeyless(allow bool) error {
	f, err := os.OpenFile(filepath.Join(fb.settingsPath(), allowKeyless), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
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
	f, err := os.OpenFile(filepath.Join(fb.settingsPath(), anonAccess), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
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
	repo = sanatizeRepo(repo) + ".git"
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
	daemonExport := filepath.Join(fb.reposPath(), repo, exportOk)
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

// CreateRepository creates a new repository.
//
// Created repositories are always bare.
//
// It implements backend.Backend.
func (fb *FileBackend) CreateRepository(repo string, private bool) (backend.Repository, error) {
	name := sanatizeRepo(repo)
	repo = name + ".git"
	rp := filepath.Join(fb.reposPath(), repo)
	if _, err := os.Stat(rp); err == nil {
		return nil, os.ErrExist
	}

	if _, err := git.Init(rp, true); err != nil {
		logger.Debug("failed to create repository", "err", err)
		return nil, err
	}

	fb.SetPrivate(repo, private)
	fb.SetDescription(repo, "")

	r := &Repo{path: rp, root: fb.reposPath()}
	// Add to cache.
	fb.repos[name] = r
	return r, nil
}

// DeleteRepository deletes the given repository.
//
// It implements backend.Backend.
func (fb *FileBackend) DeleteRepository(repo string) error {
	name := sanatizeRepo(repo)
	delete(fb.repos, name)
	repo = name + ".git"
	return os.RemoveAll(filepath.Join(fb.reposPath(), repo))
}

// RenameRepository renames the given repository.
//
// It implements backend.Backend.
func (fb *FileBackend) RenameRepository(oldName string, newName string) error {
	oldName = filepath.Join(fb.reposPath(), sanatizeRepo(oldName)+".git")
	newName = filepath.Join(fb.reposPath(), sanatizeRepo(newName)+".git")
	if _, err := os.Stat(oldName); errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("repository %q does not exist", strings.TrimSuffix(filepath.Base(oldName), ".git"))
	}
	if _, err := os.Stat(newName); err == nil {
		return fmt.Errorf("repository %q already exists", strings.TrimSuffix(filepath.Base(newName), ".git"))
	}

	return os.Rename(oldName, newName)
}

// Repository finds the given repository.
//
// It implements backend.Backend.
func (fb *FileBackend) Repository(repo string) (backend.Repository, error) {
	name := sanatizeRepo(repo)
	if r, ok := fb.repos[name]; ok {
		return r, nil
	}

	repo = name + ".git"
	rp := filepath.Join(fb.reposPath(), repo)
	_, err := os.Stat(rp)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, os.ErrNotExist
		}
		return nil, err
	}

	return &Repo{path: rp, root: fb.reposPath()}, nil
}

// Returns true if path is a directory containing an `objects` directory and a
// `HEAD` file.
func isGitDir(path string) bool {
	stat, err := os.Stat(filepath.Join(path, "objects"))
	if err != nil {
		return false
	}
	if !stat.IsDir() {
		return false
	}

	stat, err = os.Stat(filepath.Join(path, "HEAD"))
	if err != nil {
		return false
	}
	if stat.IsDir() {
		return false
	}

	return true
}

// initRepos initializes the repository cache.
func (fb *FileBackend) initRepos() error {
	fb.repos = make(map[string]*Repo)
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

		if isGitDir(path) {
			r := &Repo{path: path, root: fb.reposPath()}
			fb.repos[r.Name()] = r
			repos = append(repos, r)
		}

		return nil
	})
	if err != nil {
		return err
	}

	return nil
}

// Repositories returns a list of all repositories.
//
// It implements backend.Backend.
func (fb *FileBackend) Repositories() ([]backend.Repository, error) {
	repos := make([]backend.Repository, 0)
	for _, r := range fb.repos {
		repos = append(repos, r)
	}

	return repos, nil
}
