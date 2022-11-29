package config

import (
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
	anon := c.AnonAccess
	private := c.isPrivate(repo)
	// Find user
	if pk != nil {
		if u := c.findUser(pk); u != nil {
			if u.IsAdmin() {
				return proto.AdminAccess
			}
			if c.isCollab(repo, pk) {
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
	}
	// Don't restrict access to private repos if no users are configured.
	// Return anon access level.
	if private && c.countUsers() > 0 {
		return proto.NoAccess
	}
	return anon
}

func (c *Config) countUsers() int {
	count, err := c.db.CountUsers()
	if err != nil {
		return 0
	}
	return count
}

func (c *Config) findUser(pk ssh.PublicKey) proto.User {
	k := strings.TrimSpace(string(gossh.MarshalAuthorizedKey(pk)))
	u, err := c.DB().GetUserByPublicKey(k)
	if err != nil {
		return nil
	}
	ks, err := c.DB().GetUserPublicKeys(u)
	if err != nil {
		return nil
	}
	return &user{user: u, keys: ks}
}

func (c *Config) findRepo(repo string) proto.Repository {
	r, err := c.DB().Open(repo)
	if err != nil {
		return nil
	}
	return r
}

func (c *Config) isPrivate(repo string) bool {
	if r := c.findRepo(repo); r != nil {
		return r.IsPrivate()
	}
	return false
}

func (c *Config) isCollab(repo string, pk ssh.PublicKey) bool {
	pks, err := c.DB().ListRepoPublicKeys(repo)
	if err != nil {
		return false
	}
	for _, k := range pks {
		if ssh.KeysEqual(pk, k) {
			return true
		}
	}
	return false
}
