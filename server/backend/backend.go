package backend

import (
	"bytes"

	"golang.org/x/crypto/ssh"
)

// Backend is an interface that handles repositories management and any
// non-Git related operations.
type Backend interface {
	ServerBackend
	RepositoryStore
	RepositoryMetadata
	RepositoryAccess
}

// ParseAuthorizedKey parses an authorized key string into a public key.
func ParseAuthorizedKey(ak string) (ssh.PublicKey, string, error) {
	pk, c, _, _, err := ssh.ParseAuthorizedKey([]byte(ak))
	return pk, c, err
}

// MarshalAuthorizedKey marshals a public key into an authorized key string.
//
// This is the inverse of ParseAuthorizedKey.
// This function is a copy of ssh.MarshalAuthorizedKey, but without the trailing newline.
// It returns an empty string if pk is nil.
func MarshalAuthorizedKey(pk ssh.PublicKey) string {
	if pk == nil {
		return ""
	}
	return string(bytes.TrimSuffix(ssh.MarshalAuthorizedKey(pk), []byte("\n")))
}
