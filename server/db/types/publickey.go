package types

import (
	"fmt"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

var _ ssh.PublicKey = &PublicKey{}
var _ fmt.Stringer = &PublicKey{}

// PublicKey is a public key database model.
type PublicKey struct {
	ID        int
	UserID    int
	PublicKey string
	CreatedAt *time.Time
	UpdatedAt *time.Time
}

func (k *PublicKey) publicKey() ssh.PublicKey {
	pk, err := ssh.ParsePublicKey([]byte(k.PublicKey))
	if err != nil {
		return nil
	}
	return pk
}

func (k *PublicKey) String() string {
	pk := k.publicKey()
	if pk == nil {
		return ""
	}
	return strings.TrimSpace(string(ssh.MarshalAuthorizedKey(pk)))
}

// Type returns the type of the public key.
func (k *PublicKey) Type() string {
	pk := k.publicKey()
	if pk == nil {
		return ""
	}
	return pk.Type()
}

// Marshal returns the serialized form of the public key.
func (k *PublicKey) Marshal() []byte {
	pk := k.publicKey()
	if pk == nil {
		return nil
	}
	return pk.Marshal()
}

// Verify verifies the signature of the given data.
func (k *PublicKey) Verify(data []byte, sig *ssh.Signature) error {
	pk := k.publicKey()
	if pk == nil {
		return fmt.Errorf("invalid public key")
	}
	return pk.Verify(data, sig)
}
