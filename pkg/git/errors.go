// Package git provides Git service operations and utilities.
package git

import "errors"

var (
	// ErrNotAuthed represents unauthorized access.
	ErrNotAuthed = errors.New("you are not authorized to do this")

	// ErrSystemMalfunction represents a general system error returned to clients.
	ErrSystemMalfunction = errors.New("something went wrong")

	// ErrInvalidRepo represents an attempt to access a non-existent repo.
	ErrInvalidRepo = errors.New("invalid repo")

	// ErrInvalidRequest represents an invalid request.
	ErrInvalidRequest = errors.New("invalid request")

	// ErrMaxConnections represents a maximum connection limit being reached.
	ErrMaxConnections = errors.New("too many connections, try again later")

	// ErrTimeout is returned when the maximum read timeout is exceeded.
	ErrTimeout = errors.New("I/O timeout reached")
)
