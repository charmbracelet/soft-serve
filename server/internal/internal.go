package internal

import (
	"context"
	"fmt"

	"github.com/charmbracelet/keygen"
	"github.com/charmbracelet/soft-serve/server/backend"
	"github.com/charmbracelet/soft-serve/server/config"
	"github.com/charmbracelet/soft-serve/server/hooks"
	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
)

// InternalServer is a internal interface to communicate with the server.
type InternalServer struct {
	cfg *config.Config
	s   *ssh.Server
	kp  *keygen.SSHKeyPair
	ckp *keygen.SSHKeyPair
}

// NewInternalServer returns a new internal server.
func NewInternalServer(cfg *config.Config, hooks hooks.Hooks) (*InternalServer, error) {
	i := &InternalServer{cfg: cfg}

	// Create internal key.
	ikp, err := keygen.New(
		cfg.Internal.InternalKeyPath,
		keygen.WithKeyType(keygen.Ed25519),
		keygen.WithWrite(),
	)
	if err != nil {
		return nil, fmt.Errorf("internal key: %w", err)
	}

	i.kp = ikp

	// Create client key.
	ckp, err := keygen.New(
		cfg.Internal.ClientKeyPath,
		keygen.WithKeyType(keygen.Ed25519),
		keygen.WithWrite(),
	)
	if err != nil {
		return nil, fmt.Errorf("client key: %w", err)
	}

	i.ckp = ckp

	s, err := wish.NewServer(
		wish.WithAddress(cfg.Internal.ListenAddr),
		wish.WithHostKeyPath(cfg.Internal.KeyPath),
		wish.WithPublicKeyAuth(i.PublicKeyHandler),
		wish.WithMiddleware(
			i.Middleware(hooks),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("wish: %w", err)
	}

	i.s = s

	return i, nil
}

// PublicKeyHandler handles public key authentication.
func (i *InternalServer) PublicKeyHandler(ctx ssh.Context, pk ssh.PublicKey) bool {
	return backend.KeysEqual(i.kp.PublicKey(), pk)
}

// Start starts the internal server.
func (i *InternalServer) Start() error {
	return i.s.ListenAndServe()
}

// Shutdown shuts down the internal server.
func (i *InternalServer) Shutdown(ctx context.Context) error {
	return i.s.Shutdown(ctx)
}

// Close closes the internal server.
func (i *InternalServer) Close() error {
	return i.s.Close()
}
