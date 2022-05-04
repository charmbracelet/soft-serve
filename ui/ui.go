package ui

import (
	"log"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/soft-serve/ui/common"
	"github.com/charmbracelet/soft-serve/ui/components/footer"
	"github.com/charmbracelet/soft-serve/ui/components/header"
	"github.com/charmbracelet/soft-serve/ui/components/selector"
	"github.com/charmbracelet/soft-serve/ui/git"
	"github.com/charmbracelet/soft-serve/ui/pages/repo"
	"github.com/charmbracelet/soft-serve/ui/pages/selection"
	"github.com/charmbracelet/soft-serve/ui/session"
)

type page int

const (
	selectionPage page = iota
	repoPage
)

type sessionState int

const (
	startState sessionState = iota
	errorState
	loadedState
)

// UI is the main UI model.
type UI struct {
	s           session.Session
	initialRepo string
	common      common.Common
	pages       []common.Page
	activePage  page
	state       sessionState
	header      *header.Header
	footer      *footer.Footer
	error       error
}

// New returns a new UI model.
func New(s session.Session, c common.Common, initialRepo string) *UI {
	h := header.New(c, s.Config().Name)
	ui := &UI{
		s:           s,
		common:      c,
		pages:       make([]common.Page, 2), // selection & repo
		activePage:  selectionPage,
		state:       startState,
		header:      h,
		initialRepo: initialRepo,
	}
	ui.footer = footer.New(c, ui)
	ui.SetSize(c.Width, c.Height)
	return ui
}

func (ui *UI) getMargins() (wm, hm int) {
	wm = ui.common.Styles.App.GetHorizontalFrameSize()
	hm = ui.common.Styles.App.GetVerticalFrameSize() +
		ui.common.Styles.Header.GetHeight() +
		ui.common.Styles.Header.GetVerticalFrameSize() +
		ui.common.Styles.Footer.GetVerticalFrameSize() +
		ui.footer.Height()
	return
}

// ShortHelp implements help.KeyMap.
func (ui *UI) ShortHelp() []key.Binding {
	b := make([]key.Binding, 0)
	switch ui.state {
	case errorState:
		b = append(b, ui.common.KeyMap.Back)
	case loadedState:
		b = append(b, ui.pages[ui.activePage].ShortHelp()...)
	}
	b = append(b,
		ui.common.KeyMap.Quit,
		ui.common.KeyMap.Help,
	)
	return b
}

// FullHelp implements help.KeyMap.
func (ui *UI) FullHelp() [][]key.Binding {
	b := make([][]key.Binding, 0)
	switch ui.state {
	case errorState:
		b = append(b, []key.Binding{ui.common.KeyMap.Back})
	case loadedState:
		b = append(b, ui.pages[ui.activePage].FullHelp()...)
	}
	b = append(b, []key.Binding{
		ui.common.KeyMap.Quit,
		ui.common.KeyMap.Help,
	})
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
	ui.pages[selectionPage] = selection.New(ui.s, ui.common)
	ui.pages[repoPage] = repo.New(ui.s, ui.common)
	ui.SetSize(ui.common.Width, ui.common.Height)
	cmds := make([]tea.Cmd, 0)
	cmds = append(cmds,
		ui.pages[selectionPage].Init(),
		ui.pages[repoPage].Init(),
	)
	if ui.initialRepo != "" {
		cmds = append(cmds, ui.initialRepoCmd(ui.initialRepo))
	}
	ui.state = loadedState
	return tea.Batch(cmds...)
}

// Update implements tea.Model.
func (ui *UI) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	log.Printf("msg: %T", msg)
	cmds := make([]tea.Cmd, 0)
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		ui.SetSize(msg.Width, msg.Height)
		for i, p := range ui.pages {
			m, cmd := p.Update(msg)
			ui.pages[i] = m.(common.Page)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
	case tea.KeyMsg, tea.MouseMsg:
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch {
			case key.Matches(msg, ui.common.KeyMap.Back) && ui.error != nil:
				ui.error = nil
				ui.state = loadedState
			case key.Matches(msg, ui.common.KeyMap.Help):
				ui.footer.SetShowAll(!ui.footer.ShowAll())
			case key.Matches(msg, ui.common.KeyMap.Quit):
				return ui, tea.Quit
			case ui.activePage == repoPage && key.Matches(msg, ui.common.KeyMap.Back):
				ui.activePage = selectionPage
			}
		}
	case common.ErrorMsg:
		ui.error = msg
		ui.state = errorState
		return ui, nil
	case selector.SelectMsg:
		switch msg.IdentifiableItem.(type) {
		case selection.Item:
			if ui.activePage == selectionPage {
				cmds = append(cmds, ui.setRepoCmd(msg.ID()))
			}
		}
	}
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
	if ui.state == loadedState {
		m, cmd := ui.pages[ui.activePage].Update(msg)
		ui.pages[ui.activePage] = m.(common.Page)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}
	// This fixes determining the height margin of the footer.
	ui.SetSize(ui.common.Width, ui.common.Height)
	return ui, tea.Batch(cmds...)
}

// View implements tea.Model.
func (ui *UI) View() string {
	var view string
	wm, hm := ui.getMargins()
	footer := ui.footer.View()
	style := ui.common.Styles.App.Copy()
	switch ui.state {
	case startState:
		view = "Loading..."
	case errorState:
		err := ui.common.Styles.ErrorTitle.Render("Bummer")
		err += ui.common.Styles.ErrorBody.Render(ui.error.Error())
		view = ui.common.Styles.Error.Copy().
			Width(ui.common.Width -
				wm -
				ui.common.Styles.ErrorBody.GetHorizontalFrameSize()).
			Height(ui.common.Height -
				hm -
				ui.common.Styles.Error.GetVerticalFrameSize()).
			Render(err)
	case loadedState:
		view = ui.pages[ui.activePage].View()
	default:
		view = "Unknown state :/ this is a bug!"
	}
	return style.Render(
		lipgloss.JoinVertical(lipgloss.Bottom,
			ui.header.View(),
			view,
			footer,
		),
	)
}

func (ui *UI) setRepoCmd(rn string) tea.Cmd {
	rs := ui.s.Source()
	return func() tea.Msg {
		for _, r := range rs.AllRepos() {
			if r.Repo() == rn {
				ui.activePage = repoPage
				return repo.RepoMsg(r)
			}
		}
		return common.ErrorMsg(git.ErrMissingRepo)
	}
}

func (ui *UI) initialRepoCmd(rn string) tea.Cmd {
	rs := ui.s.Source()
	return func() tea.Msg {
		for _, r := range rs.AllRepos() {
			if r.Repo() == rn {
				ui.activePage = repoPage
				return repo.RepoMsg(r)
			}
		}
		return nil
	}
}
