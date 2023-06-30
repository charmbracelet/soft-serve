package auth

import (
	"context"
	"fmt"

	"github.com/charmbracelet/soft-serve/server/sshutils"
	"golang.org/x/crypto/ssh"
)

// Auth is an interface that represents a auth store.
type Auth interface {
	// Authenticate returns the user for the given auth method.
	Authenticate(ctx context.Context, method AuthMethod) (User, error)
}

// AuthMethod is an interface that represents a auth method.
type AuthMethod interface {
	fmt.Stringer

	// Name returns the name of the auth method.
	Name() string
}

// PublicKey is a public-key auth method.
type PublicKey struct {
	ssh.PublicKey
}

// NewPublicKey returns a new PublicKey auth method from ssh.PublicKey.
func NewPublicKey(pk ssh.PublicKey) PublicKey {
	return PublicKey{PublicKey: pk}
}

var _ AuthMethod = PublicKey{}

// String implements AuthMethod.
func (pk PublicKey) String() string {
	return sshutils.MarshalAuthorizedKey(pk.PublicKey)
}

// Name implements AuthMethod.
func (PublicKey) Name() string {
	return "ssh-public-key"
}
