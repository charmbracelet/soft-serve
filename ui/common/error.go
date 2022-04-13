package common

import tea "github.com/charmbracelet/bubbletea"

type ErrorMsg error

func ErrorCmd(err error) tea.Cmd {
	return func() tea.Msg {
		return ErrorMsg(err)
	}
}
