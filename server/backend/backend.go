package backend

import (
	"context"

	"github.com/charmbracelet/log"
	"github.com/charmbracelet/soft-serve/server/access"
	"github.com/charmbracelet/soft-serve/server/auth"
	"github.com/charmbracelet/soft-serve/server/config"
	"github.com/charmbracelet/soft-serve/server/settings"
	"github.com/charmbracelet/soft-serve/server/sshutils"
	"github.com/charmbracelet/soft-serve/server/store"
	"github.com/charmbracelet/ssh"
	gossh "golang.org/x/crypto/ssh"
)

// Backend handles repository management, server settings, authentication, and
// authorization.
type Backend struct {
	settings.Settings
	store.Store
	auth.Auth
	access.Access

	ctx    context.Context
	logger *log.Logger
}

// NewBackendStore returns a new BackendStore.
func NewBackend(ctx context.Context, s settings.Settings, st store.Store, a auth.Auth, ac access.Access) (*Backend, error) {
	ba := &Backend{
		Settings: s,
		Store:    st,
		Auth:     a,
		Access:   ac,
		ctx:      ctx,
		logger:   log.FromContext(ctx).WithPrefix("backend"),
	}

	return ba, nil
}

// AccessLevel returns the access level for the given user and repository.
// It will also check if the SSH connection is using an admin key.
func (b *Backend) AccessLevel(ctx context.Context, repo string, user auth.User) (access.AccessLevel, error) {
	cfg := config.FromContext(ctx)
	if cfg != nil {
		log.Printf("found cfg")
		if pk, ok := ctx.Value(ssh.ContextKeyPublicKey).(gossh.PublicKey); ok {
			log.Printf("pk: %v", sshutils.MarshalAuthorizedKey(pk))
			for _, k := range cfg.AdminKeys() {
				if sshutils.KeysEqual(pk, k) {
					return access.AdminAccess, nil
				}
			}
		}
		log.Printf("no pk")
	}

	log.Printf("no cfg")

	return b.Access.AccessLevel(ctx, repo, user)
}
