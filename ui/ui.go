package ui

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	appCfg "github.com/charmbracelet/soft-serve/config"
	"github.com/charmbracelet/soft-serve/ui/keymap"
)

type Session interface {
	Send(tea.Msg)
	Config() *appCfg.Config
	Width() int
	Height() int
	InitialRepo() string
}

type UI struct {
	s    Session
	keys *keymap.KeyMap
}

func New(s Session) *UI {
	ui := &UI{
		s:    s,
		keys: keymap.DefaultKeyMap(),
	}
	return ui
}

func (ui *UI) Init() tea.Cmd {
	return nil
}

func (ui *UI) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmds := make([]tea.Cmd, 0)
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, ui.keys.Quit):
			return ui, tea.Quit
		}
	}
	return ui, tea.Batch(cmds...)
}

func (ui *UI) View() string {
	return ""
}
