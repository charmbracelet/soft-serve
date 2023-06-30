package auth

import "errors"

var (
	// ErrUnsupportedAuthMethod is returned when an unsupported auth method is
	// used.
	ErrUnsupportedAuthMethod = errors.New("unsupported auth method")
)
