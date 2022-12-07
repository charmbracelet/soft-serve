package config

import (
	"log"
	"net/mail"

	"github.com/charmbracelet/soft-serve/proto"
	"github.com/charmbracelet/soft-serve/server/db/types"
	"golang.org/x/crypto/ssh"
)

var _ proto.User = &user{}

type user struct {
	cfg  *Config
	user *types.User
	keys []*types.PublicKey
}

func (u *user) Name() string {
	return u.user.Name
}

func (u *user) Email() *mail.Address {
	return u.user.Address()
}

func (u *user) Login() *string {
	return u.user.Login
}

func (u *user) Password() *string {
	return u.user.Password
}

func (u *user) IsAdmin() bool {
	return u.user.Admin
}

func (u *user) PublicKeys() []ssh.PublicKey {
	keys := u.keys
	if keys == nil || len(keys) == 0 {
		ks, err := u.cfg.db.GetUserPublicKeys(u.user)
		if err != nil {
			log.Printf("error getting public keys for %q: %v", u.Name(), err)
			return nil
		}
		u.keys = ks
		keys = ks
	}
	ks := make([]ssh.PublicKey, len(keys))
	for i, k := range keys {
		ks[i] = k
	}
	return ks
}
