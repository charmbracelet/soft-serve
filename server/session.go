package server

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	appCfg "github.com/charmbracelet/soft-serve/config"
	"github.com/charmbracelet/soft-serve/ui"
	bm "github.com/charmbracelet/wish/bubbletea"
	"github.com/gliderlabs/ssh"
)

type Session struct {
	tea.Model
	*tea.Program
	ssh.Session
	Cfg         *appCfg.Config
	width       int
	height      int
	initialRepo string
}

func (s *Session) Config() *appCfg.Config {
	return s.Cfg
}

func (s *Session) Send(msg tea.Msg) {
	s.Program.Send(msg)
}

func (s *Session) Width() int {
	return s.width
}

func (s *Session) Height() int {
	return s.height
}

func (s *Session) InitialRepo() string {
	return s.initialRepo
}

func SessionHandler(ac *appCfg.Config) bm.ProgramHandler {
	return func(s ssh.Session) *tea.Program {
		pty, _, active := s.Pty()
		if !active {
			fmt.Println("not active")
			return nil
		}
		sess := &Session{
			Session:     s,
			Cfg:         ac,
			width:       pty.Window.Width,
			height:      pty.Window.Height,
			initialRepo: "",
		}
		cmd := s.Command()
		switch len(cmd) {
		case 0:
			sess.initialRepo = ""
		case 1:
			sess.initialRepo = cmd[0]
		}
		if ac.Cfg.Callbacks != nil {
			ac.Cfg.Callbacks.Tui("new session")
		}
		m := ui.New(sess)
		p := tea.NewProgram(m,
			tea.WithInput(s),
			tea.WithOutput(s),
			tea.WithAltScreen(),
			tea.WithoutCatchPanics(),
		)
		sess.Model = m
		sess.Program = p
		return p
	}
}
