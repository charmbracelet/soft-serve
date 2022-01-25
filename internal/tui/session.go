package tui

import (
	"fmt"
	"net/url"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/soft-serve/internal/config"
	"github.com/gliderlabs/ssh"
)

func SessionHandler(cfg *config.Config) func(ssh.Session) (tea.Model, []tea.ProgramOption) {
	return func(s ssh.Session) (tea.Model, []tea.ProgramOption) {
		cmd := s.Command()
		scfg := &SessionConfig{Session: s}
		switch len(cmd) {
		case 0:
			scfg.InitialRepo = ""
		case 1:
			p, err := url.Parse(cmd[0])
			if err != nil || strings.Contains(p.Path, "/") {
				return nil, nil
			}
			scfg.InitialRepo = cmd[0]
		default:
			return nil, nil
		}
		pty, _, active := s.Pty()
		if !active {
			fmt.Println("not active")
			return nil, nil
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
