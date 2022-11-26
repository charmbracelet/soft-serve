package git

import (
	"github.com/charmbracelet/soft-serve/proto"
	"github.com/gliderlabs/ssh"
)

// Hooks is an interface that allows for custom authorization
// implementations and post push/fetch notifications. Prior to git access,
// AuthRepo will be called with the ssh.Session public key and the repo name.
// Implementers return the appropriate AccessLevel.
type Hooks interface {
	AuthRepo(string, ssh.PublicKey) proto.AccessLevel
	Push(string, ssh.PublicKey)
	Fetch(string, ssh.PublicKey)
}
