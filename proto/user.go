package proto

import (
	"net/mail"

	"golang.org/x/crypto/ssh"
)

// User is a user.
type User interface {
	Name() string
	PublicKeys() []ssh.PublicKey
	Login() *string
	Email() *mail.Address
	Password() *string
	IsAdmin() bool
}
