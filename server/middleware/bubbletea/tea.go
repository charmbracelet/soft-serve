package bubbletea

import (
	"smoothie/server/middleware"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gliderlabs/ssh"
)

func Middleware(bth func(ssh.Session) tea.Model, opts ...tea.ProgramOption) middleware.Middleware {
	return func(sh ssh.Handler) ssh.Handler {
		return func(s ssh.Session) {
			m := bth(s)
			if m != nil {
				opts = append(opts, tea.WithInput(s), tea.WithOutput(s))
				p := tea.NewProgram(m, opts...)
				_ = p.Start()
			}
			sh(s)
		}
	}
}
