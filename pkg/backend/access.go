package backend

import (
	"context"

	"github.com/charmbracelet/soft-serve/pkg/access"
	"github.com/charmbracelet/soft-serve/pkg/proto"
	"github.com/charmbracelet/soft-serve/pkg/sshutils"
	"golang.org/x/crypto/ssh"
)

// AccessLevel returns the access level of a user for a repository.
//
// It implements backend.Backend.
func (d *Backend) AccessLevel(ctx context.Context, repo string, username string) access.AccessLevel {
	user, _ := d.User(ctx, username)
	return d.AccessLevelForUser(ctx, repo, user)
}

// AccessLevelByPublicKey returns the access level of a user's public key for a repository.
//
// It implements backend.Backend.
func (d *Backend) AccessLevelByPublicKey(ctx context.Context, repo string, pk ssh.PublicKey) access.AccessLevel {
	for _, k := range d.cfg.AdminKeys() {
		if sshutils.KeysEqual(pk, k) {
			return access.AdminAccess
		}
	}

	user, _ := d.UserByPublicKey(ctx, pk)
	if user != nil {
		return d.AccessLevel(ctx, repo, user.Username())
	}

	return d.AccessLevel(ctx, repo, "")
}

// AccessLevelForUser returns the access level of a user for a repository.
// TODO: user repository ownership
func (d *Backend) AccessLevelForUser(ctx context.Context, repo string, user proto.User) access.AccessLevel {
	var username string
	anon := d.AnonAccess(ctx)
	if user != nil {
		username = user.Username()
	}

	// If the user is an admin, they have admin access.
	if user != nil && user.IsAdmin() {
		return access.AdminAccess
	}

	// If the repository exists, check if the user is a collaborator.
	r := proto.RepositoryFromContext(ctx)
	if r == nil {
		r, _ = d.Repository(ctx, repo)
	}

	if r != nil {
		if user != nil {
			// If the user is the owner, they have admin access.
			if r.UserID() == user.ID() {
				return access.AdminAccess
			}
		}

		// If the user is a collaborator, they have return their access level.
		collabAccess, isCollab, _ := d.IsCollaborator(ctx, repo, username)
		if isCollab {
			if anon > collabAccess {
				return anon
			}
			return collabAccess
		}

		// If the repository is private, the user has no access.
		if r.IsPrivate() {
			return access.NoAccess
		}

		// Otherwise, the user has read-only access.
		return access.ReadOnlyAccess
	}

	if user != nil {
		// If the repository doesn't exist, the user has read/write access.
		if anon > access.ReadWriteAccess {
			return anon
		}

		return access.ReadWriteAccess
	}

	// If the user doesn't exist, give them the anonymous access level.
	return anon
}
