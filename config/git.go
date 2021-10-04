package config

import (
	"log"

	gm "github.com/charmbracelet/wish/git"
	"github.com/gliderlabs/ssh"
)

func (cfg *Config) Push(repo string, pk ssh.PublicKey) {
	log.Printf("git push: %s", repo)
	err := cfg.reload()
	if err != nil {
		log.Printf("error reloading after push: %s", err)
	}
}

func (cfg *Config) Fetch(repo string, pk ssh.PublicKey) {
	log.Printf("git fetch: %s", repo)
}

func (cfg *Config) AuthRepo(repo string, pk ssh.PublicKey) gm.AccessLevel {
	return cfg.accessForKey(repo, pk)
}

func (cfg *Config) PasswordHandler(ctx ssh.Context, password string) bool {
	return (cfg.AnonAccess != "no-access") && cfg.AllowNoKeys
}

func (cfg *Config) PublicKeyHandler(ctx ssh.Context, pk ssh.PublicKey) bool {
	if cfg.accessForKey("", pk) == gm.NoAccess {
		return false
	}
	return true
}

func (cfg *Config) accessForKey(repo string, pk ssh.PublicKey) gm.AccessLevel {
	for _, u := range cfg.Users {
		apk, _, _, _, err := ssh.ParseAuthorizedKey([]byte(u.PublicKey))
		if err != nil {
			log.Printf("error: malformed authorized key: '%s'", u.PublicKey)
			return gm.NoAccess
		}
		if ssh.KeysEqual(pk, apk) {
			if u.Admin {
				return gm.AdminAccess
			}
			for _, r := range u.CollabRepos {
				if repo == r {
					return gm.ReadWriteAccess
				}
			}
			if repo != "config" {
				return gm.ReadOnlyAccess
			}
		}
	}
	if repo == "config" && (cfg.AnonAccess != "read-write") {
		return gm.NoAccess
	}
	switch cfg.AnonAccess {
	case "no-access":
		return gm.NoAccess
	case "read-only":
		return gm.ReadOnlyAccess
	case "read-write":
		return gm.ReadWriteAccess
	default:
		return gm.NoAccess
	}
}
