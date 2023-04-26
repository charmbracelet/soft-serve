package errors

import "fmt"

var (
	// ErrUnauthorized is returned when the user is not authorized to perform action.
	ErrUnauthorized = fmt.Errorf("Unauthorized")
	// ErrRepoNotFound is returned when the repo is not found.
	ErrRepoNotFound = fmt.Errorf("Repository not found")
	// ErrFileNotFound is returned when the file is not found.
	ErrFileNotFound = fmt.Errorf("File not found")
)
