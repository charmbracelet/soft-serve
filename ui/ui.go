package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/soft-serve/ui/common"
	"github.com/charmbracelet/soft-serve/ui/components/footer"
	"github.com/charmbracelet/soft-serve/ui/components/header"
	"github.com/charmbracelet/soft-serve/ui/pages/selection"
	"github.com/charmbracelet/soft-serve/ui/session"
)

type sessionState int

const (
	startState sessionState = iota
	errorState
	loadedState
)

// UI is the main UI model.
type UI struct {
	s          session.Session
	common     common.Common
	pages      []common.Page
	activePage int
	state      sessionState
	header     *header.Header
	footer     *footer.Footer
	error      error
}

// New returns a new UI model.
func New(s session.Session, c common.Common, initialRepo string) *UI {
	h := header.New(c, s.Config().Name)
	ui := &UI{
		s:          s,
		common:     c,
		pages:      make([]common.Page, 2), // selection & repo
		activePage: 0,
		state:      startState,
		header:     h,
	}
	ui.footer = footer.New(c, ui)
	ui.SetSize(c.Width, c.Height)
	return ui
}

func (ui *UI) getMargins() (wm, hm int) {
	wm = ui.common.Styles.App.GetHorizontalFrameSize()
	hm = ui.common.Styles.App.GetVerticalFrameSize() +
		ui.common.Styles.Header.GetHeight() +
		ui.common.Styles.Footer.GetHeight()
	return
}

// ShortHelp implements help.KeyMap.
func (ui *UI) ShortHelp() []key.Binding {
	b := make([]key.Binding, 0)
	b = append(b, ui.pages[ui.activePage].ShortHelp()...)
	b = append(b, ui.common.Keymap.Quit)
	return b
}

// FullHelp implements help.KeyMap.
func (ui *UI) FullHelp() [][]key.Binding {
	b := make([][]key.Binding, 0)
	b = append(b, ui.pages[ui.activePage].FullHelp()...)
	b = append(b, []key.Binding{ui.common.Keymap.Quit})
	return b
}

// SetSize implements common.Component.
func (ui *UI) SetSize(width, height int) {
	ui.common.SetSize(width, height)
	wm, hm := ui.getMargins()
	ui.header.SetSize(width-wm, height-hm)
	ui.footer.SetSize(width-wm, height-hm)
	for _, p := range ui.pages {
		if p != nil {
			p.SetSize(width-wm, height-hm)
		}
	}
}

// Init implements tea.Model.
func (ui *UI) Init() tea.Cmd {
	ui.pages[0] = selection.New(ui.s, ui.common)
	ui.pages[1] = selection.New(ui.s, ui.common)
	ui.SetSize(ui.common.Width, ui.common.Height)
	ui.state = loadedState
	return ui.pages[ui.activePage].Init()
}

// Update implements tea.Model.
// TODO update help when page change.
func (ui *UI) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmds := make([]tea.Cmd, 0)
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		h, cmd := ui.header.Update(msg)
		ui.header = h.(*header.Header)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
		f, cmd := ui.footer.Update(msg)
		ui.footer = f.(*footer.Footer)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
		for i, p := range ui.pages {
			m, cmd := p.Update(msg)
			ui.pages[i] = m.(common.Page)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
		ui.SetSize(msg.Width, msg.Height)
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, ui.common.Keymap.Quit):
			return ui, tea.Quit
		}
	case common.ErrorMsg:
		ui.error = msg
		ui.state = errorState
		return ui, nil
	}
	m, cmd := ui.pages[ui.activePage].Update(msg)
	ui.pages[ui.activePage] = m.(common.Page)
	if cmd != nil {
		cmds = append(cmds, cmd)
	}
	return ui, tea.Batch(cmds...)
}

// View implements tea.Model.
func (ui *UI) View() string {
	s := strings.Builder{}
	switch ui.state {
	case startState:
		return "\n Loading..."
	case errorState:
		err := ui.common.Styles.ErrorTitle.Render("Bummer")
		err += ui.common.Styles.ErrorBody.Render(ui.error.Error())
		return err
	case loadedState:
		s.WriteString(lipgloss.JoinVertical(
			lipgloss.Bottom,
			ui.header.View(),
			ui.pages[ui.activePage].View(),
			ui.footer.View(),
		))
	default:
		return "\n Unknown state :/ this is a bug!"
	}
	return ui.common.Styles.App.Render(s.String())
}
