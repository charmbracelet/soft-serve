package ui

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/soft-serve/ui/common"
	"github.com/charmbracelet/soft-serve/ui/pages/selection"
	"github.com/charmbracelet/soft-serve/ui/session"
)

type sessionState int

const (
	startState sessionState = iota
	errorState
	loadedState
)

type UI struct {
	s          session.Session
	common     *common.Common
	pages      []tea.Model
	activePage int
	state      sessionState
}

func New(s session.Session, common *common.Common, initialRepo string) *UI {
	ui := &UI{
		s:          s,
		common:     common,
		pages:      make([]tea.Model, 2), // selection & repo
		activePage: 0,
		state:      startState,
	}
	return ui
}

func (ui *UI) Init() tea.Cmd {
	items := make([]string, 0)
	cfg := ui.s.Config()
	for _, r := range cfg.Repos {
		items = append(items, r.Name)
	}
	for _, r := range cfg.Source.AllRepos() {
		exists := false
		for _, i := range items {
			if i == r.Name() {
				exists = true
				break
			}
		}
		if !exists {
			items = append(items, r.Name())
		}
	}
	ui.pages[0] = selection.New(ui.s, ui.common)
	ui.pages[1] = selection.New(ui.s, ui.common)
	ui.state = loadedState
	return ui.pages[ui.activePage].Init()
}

func (ui *UI) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmds := make([]tea.Cmd, 0)
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		for i, p := range ui.pages {
			m, cmd := p.Update(msg)
			ui.pages[i] = m
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, ui.common.Keymap.Quit):
			return ui, tea.Quit
		default:
			m, cmd := ui.pages[ui.activePage].Update(msg)
			ui.pages[ui.activePage] = m
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
	default:
		m, cmd := ui.pages[ui.activePage].Update(msg)
		ui.pages[ui.activePage] = m
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}
	return ui, tea.Batch(cmds...)
}

func (ui *UI) View() string {
	switch ui.state {
	case startState:
		return "Loading..."
	case errorState:
		return "Error"
	case loadedState:
		return ui.common.Styles.App.Render(ui.pages[ui.activePage].View())
	default:
		return "Unknown state :/ this is a bug!"
	}
}
