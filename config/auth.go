package config

import (
	gm "github.com/charmbracelet/wish/git"
	"github.com/gliderlabs/ssh"
)

func (cfg *Config) AuthRepo(repo string, pk ssh.PublicKey) gm.AccessLevel {
	// TODO: check yaml for access rules
	return gm.ReadWriteAccess
}

func (cfg *Config) PasswordHandler(ctx ssh.Context, password string) bool {
	return cfg.AnonReadOnly && cfg.AllowNoKeys
}

func (cfg *Config) PublicKeyHandler(ctx ssh.Context, pk ssh.PublicKey) bool {
	// TODO: check yaml for access rules
	return true
}
