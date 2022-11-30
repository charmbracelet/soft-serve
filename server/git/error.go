package git

import "errors"

// ErrNotAuthed represents unauthorized access.
var ErrNotAuthed = errors.New("you are not authorized to do this")

// ErrSystemMalfunction represents a general system error returned to clients.
var ErrSystemMalfunction = errors.New("something went wrong")

// ErrInvalidRepo represents an attempt to access a non-existent repo.
var ErrInvalidRepo = errors.New("invalid repo")

// ErrInvalidRequest represents an invalid request.
var ErrInvalidRequest = errors.New("invalid request")

// ErrMaxConnections represents a maximum connection limit being reached.
var ErrMaxConnections = errors.New("too many connections, try again later")

// ErrTimeout is returned when the maximum read timeout is exceeded.
var ErrTimeout = errors.New("I/O timeout reached")
