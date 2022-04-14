package server

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	appCfg "github.com/charmbracelet/soft-serve/config"
	"github.com/charmbracelet/soft-serve/ui"
	"github.com/charmbracelet/soft-serve/ui/common"
	"github.com/charmbracelet/soft-serve/ui/keymap"
	"github.com/charmbracelet/soft-serve/ui/styles"
	bm "github.com/charmbracelet/wish/bubbletea"
	"github.com/gliderlabs/ssh"
)

type Session struct {
	tea.Model
	*tea.Program
	session ssh.Session
	Cfg     *appCfg.Config
}

func (s *Session) Config() *appCfg.Config {
	return s.Cfg
}

func (s *Session) Send(msg tea.Msg) {
	s.Program.Send(msg)
}

func (s *Session) PublicKey() ssh.PublicKey {
	return s.session.PublicKey()
}

func (s *Session) Session() ssh.Session {
	return s.session
}

func SessionHandler(ac *appCfg.Config) bm.ProgramHandler {
	return func(s ssh.Session) *tea.Program {
		pty, _, active := s.Pty()
		if !active {
			fmt.Println("not active")
			return nil
		}
		sess := &Session{
			session: s,
			Cfg:     ac,
		}
		cmd := s.Command()
		initialRepo := ""
		if len(cmd) == 1 {
			initialRepo = cmd[0]
		}
		if ac.Cfg.Callbacks != nil {
			ac.Cfg.Callbacks.Tui("new session")
		}
		c := common.Common{
			Styles: styles.DefaultStyles(),
			Keymap: keymap.DefaultKeyMap(),
			Width:  pty.Window.Width,
			Height: pty.Window.Height,
		}
		m := ui.New(
			sess,
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
		sess.Model = m
		sess.Program = p
		return p
	}
}
