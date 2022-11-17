package config

import (
	"log"
	"strings"

	gm "github.com/charmbracelet/soft-serve/server/git"
	"github.com/gliderlabs/ssh"
	gossh "golang.org/x/crypto/ssh"
)

// Push registers Git push functionality for the given repo and key.
func (cfg *Config) Push(repo string, pk ssh.PublicKey) {
	go func() {
		err := cfg.Reload()
		if err != nil {
			log.Printf("error reloading after push: %s", err)
		}
		if cfg.Cfg.Callbacks != nil {
			cfg.Cfg.Callbacks.Push(repo)
		}
		r, err := cfg.Source.GetRepo(repo)
		if err != nil {
			log.Printf("error getting repo after push: %s", err)
			return
		}
		err = r.UpdateServerInfo()
		if err != nil {
			log.Printf("error updating server info after push: %s", err)
		}
	}()
}

// Fetch registers Git fetch functionality for the given repo and key.
func (cfg *Config) Fetch(repo string, pk ssh.PublicKey) {
	if cfg.Cfg.Callbacks != nil {
		cfg.Cfg.Callbacks.Fetch(repo)
	}
}

// AuthRepo grants repo authorization to the given key.
func (cfg *Config) AuthRepo(repo string, pk ssh.PublicKey) gm.AccessLevel {
	return cfg.accessForKey(repo, pk)
}

// PasswordHandler returns whether or not password access is allowed.
func (cfg *Config) PasswordHandler(ctx ssh.Context, password string) bool {
	return (cfg.AnonAccess != gm.NoAccess.String()) && cfg.AllowKeyless
}

// KeyboardInteractiveHandler returns whether or not keyboard interactive is allowed.
func (cfg *Config) KeyboardInteractiveHandler(ctx ssh.Context, _ gossh.KeyboardInteractiveChallenge) bool {
	return (cfg.AnonAccess != gm.NoAccess.String()) && cfg.AllowKeyless
}

// PublicKeyHandler returns whether or not the given public key may access the
// repo.
func (cfg *Config) PublicKeyHandler(ctx ssh.Context, pk ssh.PublicKey) bool {
	return cfg.accessForKey("", pk) != gm.NoAccess
}

func (cfg *Config) anonAccessLevel() gm.AccessLevel {
	cfg.mtx.RLock()
	defer cfg.mtx.RUnlock()
	switch cfg.AnonAccess {
	case "no-access":
		return gm.NoAccess
	case "read-only":
		return gm.ReadOnlyAccess
	case "read-write":
		return gm.ReadWriteAccess
	case "admin-access":
		return gm.AdminAccess
	default:
		return gm.NoAccess
	}
}

// accessForKey returns the access level for the given repo.
//
// If repo doesn't exist, then access is based on user's admin privileges, or
// config.AnonAccess.
// If repo exists, and private, then admins and collabs are allowed access.
// If repo exists, and not private, then access is based on config.AnonAccess.
func (cfg *Config) accessForKey(repo string, pk ssh.PublicKey) gm.AccessLevel {
	anon := cfg.anonAccessLevel()
	private := cfg.isPrivate(repo)
	// Find user
	for _, user := range cfg.Users {
		for _, k := range user.PublicKeys {
			apk, _, _, _, err := ssh.ParseAuthorizedKey([]byte(strings.TrimSpace(k)))
			if err != nil {
				log.Printf("error: malformed authorized key: '%s'", k)
				return gm.NoAccess
			}
			if ssh.KeysEqual(pk, apk) {
				if user.Admin {
					return gm.AdminAccess
				}
				u := user
				if cfg.isCollab(repo, &u) {
					if anon > gm.ReadWriteAccess {
						return anon
					}
					return gm.ReadWriteAccess
				}
				if !private {
					if anon > gm.ReadOnlyAccess {
						return anon
					}
					return gm.ReadOnlyAccess
				}
			}
		}
	}
	// Don't restrict access to private repos if no users are configured.
	// Return anon access level.
	if private && len(cfg.Users) > 0 {
		return gm.NoAccess
	}
	return anon
}

func (cfg *Config) findRepo(repo string) *RepoConfig {
	for _, r := range cfg.Repos {
		if r.Repo == repo {
			return &r
		}
	}
	return nil
}

func (cfg *Config) isPrivate(repo string) bool {
	if r := cfg.findRepo(repo); r != nil {
		return r.Private
	}
	return false
}

func (cfg *Config) isCollab(repo string, user *User) bool {
	if user != nil {
		for _, r := range user.CollabRepos {
			if r == repo {
				return true
			}
		}
		if r := cfg.findRepo(repo); r != nil {
			for _, c := range r.Collabs {
				if c == user.Name {
					return true
				}
			}
		}
	}
	return false
}
