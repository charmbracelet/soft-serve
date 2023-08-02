package sshutils

import (
	"bytes"
	"context"

	"github.com/charmbracelet/ssh"
	gossh "golang.org/x/crypto/ssh"
)

// ParseAuthorizedKey parses an authorized key string into a public key.
func ParseAuthorizedKey(ak string) (gossh.PublicKey, string, error) {
	pk, c, _, _, err := gossh.ParseAuthorizedKey([]byte(ak))
	return pk, c, err
}

// MarshalAuthorizedKey marshals a public key into an authorized key string.
//
// This is the inverse of ParseAuthorizedKey.
// This function is a copy of ssh.MarshalAuthorizedKey, but without the trailing newline.
// It returns an empty string if pk is nil.
func MarshalAuthorizedKey(pk gossh.PublicKey) string {
	if pk == nil {
		return ""
	}
	return string(bytes.TrimSuffix(gossh.MarshalAuthorizedKey(pk), []byte("\n")))
}

// KeysEqual returns whether the two public keys are equal.
func KeysEqual(a, b gossh.PublicKey) bool {
	return ssh.KeysEqual(a, b)
}

// PublicKeyFromContext returns the public key from the context.
func PublicKeyFromContext(ctx context.Context) gossh.PublicKey {
	if pk, ok := ctx.Value(ssh.ContextKeyPublicKey).(gossh.PublicKey); ok {
		return pk
	}
	return nil
}

// ContextKeySession is the context key for the SSH session.
var ContextKeySession = &struct{ string }{"session"}

// SessionFromContext returns the SSH session from the context.
func SessionFromContext(ctx context.Context) ssh.Session {
	if s, ok := ctx.Value(ContextKeySession).(ssh.Session); ok {
		return s
	}
	return nil
}
