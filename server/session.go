package server

import (
	"fmt"

	"github.com/aymanbagabas/go-osc52"
	tea "github.com/charmbracelet/bubbletea"
	appCfg "github.com/charmbracelet/soft-serve/config"
	cm "github.com/charmbracelet/soft-serve/server/cmd"
	"github.com/charmbracelet/soft-serve/ui"
	"github.com/charmbracelet/soft-serve/ui/common"
	"github.com/charmbracelet/soft-serve/ui/keymap"
	"github.com/charmbracelet/soft-serve/ui/styles"
	"github.com/charmbracelet/wish"
	bm "github.com/charmbracelet/wish/bubbletea"
	gm "github.com/charmbracelet/wish/git"
	"github.com/gliderlabs/ssh"
	zone "github.com/lrstanley/bubblezone"
)

// SessionHandler is the soft-serve bubbletea ssh session handler.
func SessionHandler(ac *appCfg.Config) bm.ProgramHandler {
	return func(s ssh.Session) *tea.Program {
		pty, _, active := s.Pty()
		if !active {
			return nil
		}
		cmd := s.Command()
		initialRepo := ""
		if len(cmd) == 1 {
			initialRepo = cmd[0]
			auth := ac.AuthRepo(initialRepo, s.PublicKey())
			if auth < gm.ReadOnlyAccess {
				wish.Fatalln(s, cm.ErrUnauthorized)
				return nil
			}
		}
		if ac.Cfg.Callbacks != nil {
			ac.Cfg.Callbacks.Tui("new session")
		}
		envs := s.Environ()
		envs = append(envs, fmt.Sprintf("TERM=%s", pty.Term))
		output := osc52.NewOutput(s, envs)
		c := common.Common{
			Copy:   output,
			Styles: styles.DefaultStyles(),
			KeyMap: keymap.DefaultKeyMap(),
			Width:  pty.Window.Width,
			Height: pty.Window.Height,
			Zone:   zone.New(),
		}
		m := ui.New(
			ac,
			s,
			c,
			initialRepo,
		)
		p := tea.NewProgram(m,
			tea.WithInput(s),
			tea.WithOutput(s),
			tea.WithAltScreen(),
			tea.WithoutCatchPanics(),
			tea.WithMouseCellMotion(),
		)
		return p
	}
}
