package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/soft-serve/internal/config"
	"github.com/gliderlabs/ssh"
)

// SessionHandler handles the bubble tea session.
func SessionHandler(cfg *config.Config) func(ssh.Session) (tea.Model, []tea.ProgramOption) {
	return func(s ssh.Session) (tea.Model, []tea.ProgramOption) {
		pty, _, active := s.Pty()
		if !active {
			fmt.Println("not active")
			return nil, nil
		}
		cmd := s.Command()
		scfg := &SessionConfig{Session: s}
		switch len(cmd) {
		case 0:
			scfg.InitialRepo = ""
		case 1:
			scfg.InitialRepo = cmd[0]
		}
		scfg.Width = pty.Window.Width
		scfg.Height = pty.Window.Height
		if cfg.Cfg.Callbacks != nil {
			cfg.Cfg.Callbacks.Tui("view")
		}
		return NewBubble(cfg, scfg), []tea.ProgramOption{
			tea.WithAltScreen(),
			tea.WithoutCatchPanics(),
		}
	}
}
