package utils

import (
	"errors"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/soft-serve/internal/tui/style"
)

var (
	ErrDiffTooLong      = errors.New("diff is too long")
	ErrDiffFilesTooLong = errors.New("diff files are too long")
	ErrBinaryFile       = errors.New("binary file")
	ErrFileTooLarge     = errors.New("file is too large")
	ErrInvalidFile      = errors.New("invalid file")
)

type ErrMsg struct {
	Err error
}

func (e ErrMsg) Error() string {
	return e.Err.Error()
}

func (e ErrMsg) View(s *style.Styles) string {
	return e.ViewWithPrefix(s, "")
}

func (e ErrMsg) ViewWithPrefix(s *style.Styles, prefix string) string {
	return lipgloss.JoinHorizontal(
		lipgloss.Top,
		s.ErrorTitle.Render(prefix),
		s.ErrorBody.Render(e.Error()),
	)
}
