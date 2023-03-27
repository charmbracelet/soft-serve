package noop

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/charmbracelet/soft-serve/git"
	"github.com/charmbracelet/soft-serve/server/backend"
	"golang.org/x/crypto/ssh"
)

var ErrNotImpl = fmt.Errorf("not implemented")

var _ backend.Backend = (*Noop)(nil)

var _ backend.AccessMethod = (*Noop)(nil)

// Noop is a backend that does nothing. It's used for testing.
type Noop struct {
	Port string
}

// RepositoryStorePath implements backend.Backend
func (*Noop) RepositoryStorePath() string {
	return ""
}

// Admins implements backend.Backend
func (*Noop) Admins() ([]string, error) {
	return nil, nil
}

// Collaborators implements backend.Backend
func (*Noop) Collaborators(repo string) ([]string, error) {
	return nil, nil
}

// RemoveAdmin implements backend.Backend
func (*Noop) RemoveAdmin(pk ssh.PublicKey) error {
	return nil
}

// RemoveCollaborator implements backend.Backend
func (*Noop) RemoveCollaborator(pk ssh.PublicKey, repo string) error {
	return nil
}

// AccessLevel implements backend.AccessMethod
func (*Noop) AccessLevel(repo string, pk ssh.PublicKey) backend.AccessLevel {
	return backend.AdminAccess
}

// AddAdmin implements backend.Backend
func (*Noop) AddAdmin(pk ssh.PublicKey, memo string) error {
	return ErrNotImpl
}

// AddCollaborator implements backend.Backend
func (*Noop) AddCollaborator(pk ssh.PublicKey, memo string, repo string) error {
	return ErrNotImpl
}

// AllowKeyless implements backend.Backend
func (*Noop) AllowKeyless() bool {
	return true
}

// AnonAccess implements backend.Backend
func (*Noop) AnonAccess() backend.AccessLevel {
	return backend.AdminAccess
}

// CreateRepository implements backend.Backend
func (*Noop) CreateRepository(name string, private bool) (backend.Repository, error) {
	temp, err := os.MkdirTemp("", "soft-serve")
	if err != nil {
		return nil, err
	}

	rp := filepath.Join(temp, name)
	_, err = git.Init(rp, private)
	if err != nil {
		return nil, err
	}

	return &repo{path: rp}, nil
}

// DeleteRepository implements backend.Backend
func (*Noop) DeleteRepository(name string) error {
	return ErrNotImpl
}

// Description implements backend.Backend
func (*Noop) Description(repo string) string {
	return ""
}

// IsAdmin implements backend.Backend
func (*Noop) IsAdmin(pk ssh.PublicKey) bool {
	return true
}

// IsCollaborator implements backend.Backend
func (*Noop) IsCollaborator(pk ssh.PublicKey, repo string) bool {
	return true
}

// IsPrivate implements backend.Backend
func (*Noop) IsPrivate(repo string) bool {
	return false
}

// RenameRepository implements backend.Backend
func (*Noop) RenameRepository(oldName string, newName string) error {
	return ErrNotImpl
}

// Repositories implements backend.Backend
func (*Noop) Repositories() ([]backend.Repository, error) {
	return nil, ErrNotImpl
}

// Repository implements backend.Backend
func (*Noop) Repository(repo string) (backend.Repository, error) {
	return nil, ErrNotImpl
}

// SetAllowKeyless implements backend.Backend
func (*Noop) SetAllowKeyless(allow bool) error {
	return ErrNotImpl
}

// SetAnonAccess implements backend.Backend
func (*Noop) SetAnonAccess(level backend.AccessLevel) error {
	return ErrNotImpl
}

// SetDescription implements backend.Backend
func (*Noop) SetDescription(repo string, desc string) error {
	return ErrNotImpl
}

// SetPrivate implements backend.Backend
func (*Noop) SetPrivate(repo string, priv bool) error {
	return ErrNotImpl
}
