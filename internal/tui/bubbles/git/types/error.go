package types

import "errors"

var (
	ErrDiffTooLong      = errors.New("diff is too long")
	ErrDiffFilesTooLong = errors.New("diff files are too long")
	ErrBinaryFile       = errors.New("binary file")
	ErrFileTooLarge     = errors.New("file is too large")
	ErrInvalidFile      = errors.New("invalid file")
)

type ErrMsg struct {
	Error error
}
