package config

import (
	"net/mail"

	"github.com/charmbracelet/soft-serve/proto"
	"github.com/charmbracelet/soft-serve/server/db/types"
	"golang.org/x/crypto/ssh"
)

var _ proto.User = &user{}

type user struct {
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
	ks := make([]ssh.PublicKey, len(u.keys))
	for i, k := range u.keys {
		ks[i] = k
	}
	return ks
}
