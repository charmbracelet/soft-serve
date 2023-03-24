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

// AccessLevel implements backend.AccessMethod
func (*Noop) AccessLevel(repo string, pk ssh.PublicKey) backend.AccessLevel {
	return backend.AdminAccess
}

// AddAdmin implements backend.Backend
func (*Noop) AddAdmin(pk ssh.PublicKey) error {
	return ErrNotImpl
}

// AddCollaborator implements backend.Backend
func (*Noop) AddCollaborator(pk ssh.PublicKey, repo string) error {
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

// ServerHost implements backend.Backend
func (*Noop) ServerHost() string {
	return "localhost"
}

// ServerName implements backend.Backend
func (*Noop) ServerName() string {
	return "Soft Serve"
}

// ServerPort implements backend.Backend
func (n *Noop) ServerPort() string {
	return n.Port
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

// SetServerHost implements backend.Backend
func (*Noop) SetServerHost(host string) error {
	return ErrNotImpl
}

// SetServerName implements backend.Backend
func (*Noop) SetServerName(name string) error {
	return ErrNotImpl
}

// SetServerPort implements backend.Backend
func (*Noop) SetServerPort(port string) error {
	return ErrNotImpl
}
