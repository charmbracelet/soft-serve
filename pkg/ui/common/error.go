package common

import (
	"errors"

	tea "charm.land/bubbletea/v2"
)

// ErrMissingRepo indicates that the requested repository could not be found.
var ErrMissingRepo = errors.New("missing repo")

// ErrorMsg is a Bubble Tea message that represents an error.
type ErrorMsg error

// ErrorCmd returns an ErrorMsg from error.
func ErrorCmd(err error) tea.Cmd {
	return func() tea.Msg {
		return ErrorMsg(err)
	}
}
