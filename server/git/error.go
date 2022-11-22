package git

import "errors"

// ErrNotAuthed represents unauthorized access.
var ErrNotAuthed = errors.New("you are not authorized to do this")

// ErrSystemMalfunction represents a general system error returned to clients.
var ErrSystemMalfunction = errors.New("something went wrong")

// ErrInvalidRepo represents an attempt to access a non-existent repo.
var ErrInvalidRepo = errors.New("invalid repo")

// ErrMaxConns represents a maximum connection limit being reached.
var ErrMaxConns = errors.New("too many connections, try again later")

// ErrMaxTimeout is returned when the maximum read timeout is exceeded.
var ErrMaxTimeout = errors.New("git: max timeout reached")
