package sqlite

import (
	"net/mail"
	"strconv"

	"github.com/charmbracelet/soft-serve/proto"
	"github.com/charmbracelet/soft-serve/server/db/types"
	"golang.org/x/crypto/ssh"
)

var _ proto.CollaboratorService = &Sqlite{}

// AddCollaborator adds a collaborator to a repository.
func (d *Sqlite) AddCollaborator(repo string, collab proto.Collaborator) error {
	r, err := d.GetRepo(repo)
	if err != nil {
		return err
	}
	switch c := collab.(type) {
	}
}

// RemoveCollaborator removes a collaborator from a repository.
func (d *Sqlite) RemoveCollaborator(repo string, collab proto.Collaborator) error {
	return nil
}

// ListCollaborators lists the collaborators of a repository.
func (d *Sqlite) ListCollaborators(repo string) ([]proto.Collaborator, error) {
	return nil, nil
}

type publicKey struct {
	key *types.PublicKey
}

// PublicKey returns the collaborator's public key.
func (k publicKey) PublicKey() ssh.PublicKey {
	pk, err := ssh.ParsePublicKey([]byte(k.key.PublicKey))
	if err != nil {
		return nil
	}
	return pk
}

var _ proto.PublicKeyCollaborator = &collaborator{}
var _ proto.UserLoginCollaborator = &collaborator{}
var _ proto.EmailCollaborator = &collaborator{}

type collaborator struct {
	user *types.User
	keys []*types.PublicKey
	db   *Sqlite
}

func (c *collaborator) init() {
	if c.keys != nil || len(c.keys) > 0 {
		return
	}
	ks, err := c.db.GetUserPublicKeys(c.user)
	if err != nil {
		return
	}
	c.keys = ks
}

// Identifier returns the collaborator's identifier.
func (c *collaborator) Identifier() string {
	return strconv.Itoa(c.user.ID)
}

// Name returns the collaborator's name.
func (c *collaborator) Name() string {
	return c.user.Name
}

// String returns the collaborator's username.
func (c *collaborator) String() string {
	return c.user.Name
}

// Marshal implements proto.PublicKeyCollaborator
func (c *collaborator) Marshal() []byte {
	c.init()
	pk := c.keys[0]
	return pk.Marshal()
}

// Type implements proto.PublicKeyCollaborator
func (c *collaborator) Type() string {
	c.init()
	pk := c.keys[0]
	return pk.Type()
}

// Verify implements proto.PublicKeyCollaborator
func (c *collaborator) Verify(data []byte, sig *ssh.Signature) error {
	c.init()
	pk := c.keys[0]
	return pk.Verify(data, sig)
}

// Login implements proto.UserLoginCollaborator
func (c *collaborator) Login() string {
	var login string
	if c.user.Login != nil {
		login = *c.user.Login
	}
	return login
}

// Address implements proto.EmailCollaborator
func (c *collaborator) Address() mail.Address {
	var addr string
	if c.user.Email != nil {
		addr = *c.user.Email
	}
	return mail.Address{
		Name:    c.user.Name,
		Address: addr,
	}
}

// Password implements proto.UserLoginCollaborator
func (c *collaborator) Password() string {
	var pwd string
	if c.user.Password != nil {
		pwd = *c.user.Password
	}
	return pwd
}
