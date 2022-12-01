package server

import (
	"fmt"

	"github.com/aymanbagabas/go-osc52"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/soft-serve/proto"
	cm "github.com/charmbracelet/soft-serve/server/cmd"
	"github.com/charmbracelet/soft-serve/server/config"
	"github.com/charmbracelet/soft-serve/ui"
	"github.com/charmbracelet/soft-serve/ui/common"
	"github.com/charmbracelet/soft-serve/ui/keymap"
	"github.com/charmbracelet/soft-serve/ui/styles"
	"github.com/charmbracelet/wish"
	bm "github.com/charmbracelet/wish/bubbletea"
	"github.com/gliderlabs/ssh"
	zone "github.com/lrstanley/bubblezone"
)

// SessionHandler is the soft-serve bubbletea ssh session handler.
func SessionHandler(cfg *config.Config) bm.ProgramHandler {
	return func(s ssh.Session) *tea.Program {
		pty, _, active := s.Pty()
		if !active {
			return nil
		}
		cmd := s.Command()
		initialRepo := ""
		if len(cmd) == 1 {
			initialRepo = cmd[0]
			auth := cfg.AuthRepo(initialRepo, s.PublicKey())
			if auth < proto.ReadOnlyAccess {
				wish.Fatalln(s, cm.ErrUnauthorized)
				return nil
			}
		}
		if cfg.Callbacks != nil {
			cfg.Callbacks.Tui("new session")
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
			cfg,
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
