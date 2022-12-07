package config

import (
	"log"
	"strings"

	"github.com/charmbracelet/soft-serve/proto"
	"github.com/gliderlabs/ssh"
	gossh "golang.org/x/crypto/ssh"
)

var _ proto.Access = &Config{}

// AuthRepo grants repo authorization to the given key.
func (c *Config) AuthRepo(repo string, pk ssh.PublicKey) proto.AccessLevel {
	return c.accessForKey(repo, pk)
}

// User returns the user for the given public key. Can return nil if no user is
// found.
func (c *Config) User(pk ssh.PublicKey) (proto.User, error) {
	k := authorizedKey(pk)
	u, err := c.DB().GetUserByPublicKey(k)
	if err != nil {
		log.Printf("error getting user for key: %s", err)
		return nil, err
	}
	if u == nil {
		return nil, nil
	}
	return &user{
		cfg:  c,
		user: u,
	}, nil
}

// IsCollab returns whether or not the given key is a collaborator on the given
// repository.
func (c *Config) IsCollab(repo string, pk ssh.PublicKey) bool {
	if c.isInitialAdminKey(pk) {
		return true
	}

	isCollab, err := c.DB().IsRepoPublicKeyCollab(repo, authorizedKey(pk))
	if err != nil {
		log.Printf("error checking if key is repo collab: %v", err)
		return false
	}
	if isCollab {
		return true
	}
	return false
}

// IsAdmin returns whether or not the given key is an admin.
func (c *Config) IsAdmin(pk ssh.PublicKey) bool {
	if c.isInitialAdminKey(pk) {
		return true
	}

	u, err := c.User(pk)
	if err != nil {
		log.Printf("error getting user for key: %s", err)
		return false
	}
	return u.IsAdmin()
}

// PasswordHandler returns whether or not password access is allowed.
func (c *Config) PasswordHandler(ctx ssh.Context, password string) bool {
	return (c.AnonAccess != proto.NoAccess) && c.SSH.AllowKeyless &&
		c.SSH.AllowPassword && (c.SSH.Password == password)
}

// KeyboardInteractiveHandler returns whether or not keyboard interactive is allowed.
func (c *Config) KeyboardInteractiveHandler(ctx ssh.Context, _ gossh.KeyboardInteractiveChallenge) bool {
	return (c.AnonAccess != proto.NoAccess) && c.SSH.AllowKeyless
}

// PublicKeyHandler returns whether or not the given public key may access the
// repo.
func (c *Config) PublicKeyHandler(ctx ssh.Context, pk ssh.PublicKey) bool {
	return c.accessForKey("", pk) != proto.NoAccess
}

// accessForKey returns the access level for the given repo.
//
// If repo doesn't exist, then access is based on user's admin privileges, or
// config.AnonAccess.
// If repo exists, and private, then admins and collabs are allowed access.
// If repo exists, and not private, then access is based on config.AnonAccess.
func (c *Config) accessForKey(repo string, pk ssh.PublicKey) proto.AccessLevel {
	if c.isInitialAdminKey(pk) {
		return proto.AdminAccess
	}

	anon := c.AnonAccess
	info, err := c.Metadata(repo)
	if err != nil || info == nil {
		log.Printf("error getting repo info: %v", err)
		return anon
	}
	private := info.IsPrivate()
	log.Printf("auth key %s", authorizedKey(pk))
	if pk != nil {
		isAdmin := c.IsAdmin(pk)
		if isAdmin {
			return proto.AdminAccess
		}
		isCollab := c.IsCollab(repo, pk)
		if isCollab {
			if anon > proto.ReadWriteAccess {
				return anon
			}
			return proto.ReadWriteAccess
		}
		if !private {
			if anon > proto.ReadOnlyAccess {
				return anon
			}
			return proto.ReadOnlyAccess
		}
	}
	// Don't restrict access to private repos if no users are configured.
	// Return anon access level.
	if private {
		return proto.NoAccess
	}
	return anon
}

func (c *Config) countUsers() int {
	count, err := c.DB().CountUsers()
	if err != nil {
		return 0
	}
	return count
}

func (c *Config) isInitialAdminKey(key ssh.PublicKey) bool {
	for _, k := range c.InitialAdminKeys {
		pk, _, _, _, err := ssh.ParseAuthorizedKey([]byte(k))
		if err != nil {
			log.Printf("error parsing initial admin key: %v", err)
			continue
		}
		if ssh.KeysEqual(key, pk) {
			return true
		}
	}
	return false
}

func authorizedKey(key ssh.PublicKey) string {
	if key == nil {
		return ""
	}
	return strings.TrimSpace(string(gossh.MarshalAuthorizedKey(key)))
}
