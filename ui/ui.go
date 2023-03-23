package ui

import (
	"errors"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/log"
	"github.com/charmbracelet/soft-serve/server/backend"
	"github.com/charmbracelet/soft-serve/ui/common"
	"github.com/charmbracelet/soft-serve/ui/components/footer"
	"github.com/charmbracelet/soft-serve/ui/components/header"
	"github.com/charmbracelet/soft-serve/ui/components/selector"
	"github.com/charmbracelet/soft-serve/ui/pages/repo"
	"github.com/charmbracelet/soft-serve/ui/pages/selection"
)

var (
	logger = log.WithPrefix("ui")
)

type page int

const (
	selectionPage page = iota
	repoPage
)

type sessionState int

const (
	loadingState sessionState = iota
	errorState
	readyState
)

// UI is the main UI model.
type UI struct {
	serverName  string
	initialRepo string
	common      common.Common
	pages       []common.Component
	activePage  page
	state       sessionState
	header      *header.Header
	footer      *footer.Footer
	showFooter  bool
	error       error
}

// New returns a new UI model.
func New(c common.Common, initialRepo string) *UI {
	var serverName string
	if cfg := c.Config(); cfg != nil {
		serverName = cfg.Backend.ServerName()
	}
	h := header.New(c, serverName)
	ui := &UI{
		serverName:  serverName,
		common:      c,
		pages:       make([]common.Component, 2), // selection & repo
		activePage:  selectionPage,
		state:       loadingState,
		header:      h,
		initialRepo: initialRepo,
		showFooter:  true,
	}
	ui.footer = footer.New(c, ui)
	return ui
}

func (ui *UI) getMargins() (wm, hm int) {
	style := ui.common.Styles.App.Copy()
	switch ui.activePage {
	case selectionPage:
		hm += ui.common.Styles.ServerName.GetHeight() +
			ui.common.Styles.ServerName.GetVerticalFrameSize()
	case repoPage:
	}
	wm += style.GetHorizontalFrameSize()
	hm += style.GetVerticalFrameSize()
	if ui.showFooter {
		// NOTE: we don't use the footer's style to determine the margins
		// because footer.Height() is the height of the footer after applying
		// the styles.
		hm += ui.footer.Height()
	}
	return
}

// ShortHelp implements help.KeyMap.
func (ui *UI) ShortHelp() []key.Binding {
	b := make([]key.Binding, 0)
	switch ui.state {
	case errorState:
		b = append(b, ui.common.KeyMap.Back)
	case readyState:
		b = append(b, ui.pages[ui.activePage].ShortHelp()...)
	}
	if !ui.IsFiltering() {
		b = append(b, ui.common.KeyMap.Quit)
	}
	b = append(b, ui.common.KeyMap.Help)
	return b
}

// FullHelp implements help.KeyMap.
func (ui *UI) FullHelp() [][]key.Binding {
	b := make([][]key.Binding, 0)
	switch ui.state {
	case errorState:
		b = append(b, []key.Binding{ui.common.KeyMap.Back})
	case readyState:
		b = append(b, ui.pages[ui.activePage].FullHelp()...)
	}
	h := []key.Binding{
		ui.common.KeyMap.Help,
	}
	if !ui.IsFiltering() {
		h = append(h, ui.common.KeyMap.Quit)
	}
	b = append(b, h)
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
	ui.pages[selectionPage] = selection.New(ui.common)
	ui.pages[repoPage] = repo.New(ui.common)
	ui.SetSize(ui.common.Width, ui.common.Height)
	cmds := make([]tea.Cmd, 0)
	cmds = append(cmds,
		ui.pages[selectionPage].Init(),
		ui.pages[repoPage].Init(),
	)
	if ui.initialRepo != "" {
		cmds = append(cmds, ui.initialRepoCmd(ui.initialRepo))
	}
	ui.state = readyState
	ui.SetSize(ui.common.Width, ui.common.Height)
	return tea.Batch(cmds...)
}

// IsFiltering returns true if the selection page is filtering.
func (ui *UI) IsFiltering() bool {
	if ui.activePage == selectionPage {
		if s, ok := ui.pages[selectionPage].(*selection.Selection); ok && s.FilterState() == list.Filtering {
			return true
		}
	}
	return false
}

// Update implements tea.Model.
func (ui *UI) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	logger.Debugf("msg received: %T", msg)
	cmds := make([]tea.Cmd, 0)
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		ui.SetSize(msg.Width, msg.Height)
		for i, p := range ui.pages {
			m, cmd := p.Update(msg)
			ui.pages[i] = m.(common.Component)
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
				ui.state = readyState
				// Always show the footer on error.
				ui.showFooter = ui.footer.ShowAll()
			case key.Matches(msg, ui.common.KeyMap.Help):
				cmds = append(cmds, footer.ToggleFooterCmd)
			case key.Matches(msg, ui.common.KeyMap.Quit):
				if !ui.IsFiltering() {
					// Stop bubblezone background workers.
					ui.common.Zone.Close()
					return ui, tea.Quit
				}
			case ui.activePage == repoPage && key.Matches(msg, ui.common.KeyMap.Back):
				ui.activePage = selectionPage
				// Always show the footer on selection page.
				ui.showFooter = true
			}
		case tea.MouseMsg:
			switch msg.Type {
			case tea.MouseLeft:
				switch {
				case ui.common.Zone.Get("footer").InBounds(msg):
					cmds = append(cmds, footer.ToggleFooterCmd)
				}
			}
		}
	case footer.ToggleFooterMsg:
		ui.footer.SetShowAll(!ui.footer.ShowAll())
		// Show the footer when on repo page and shot all help.
		if ui.error == nil && ui.activePage == repoPage {
			ui.showFooter = !ui.showFooter
		}
	case repo.RepoMsg:
		ui.common.SetValue(common.RepoKey, msg)
		ui.activePage = repoPage
		// Show the footer on repo page if show all is set.
		ui.showFooter = ui.footer.ShowAll()
		cmds = append(cmds, repo.UpdateRefCmd(msg))
	case common.ErrorMsg:
		ui.error = msg
		ui.state = errorState
		ui.showFooter = true
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
	if ui.state != loadingState {
		m, cmd := ui.pages[ui.activePage].Update(msg)
		ui.pages[ui.activePage] = m.(common.Component)
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
	switch ui.state {
	case loadingState:
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
	case readyState:
		view = ui.pages[ui.activePage].View()
	default:
		view = "Unknown state :/ this is a bug!"
	}
	if ui.activePage == selectionPage {
		view = lipgloss.JoinVertical(lipgloss.Left, ui.header.View(), view)
	}
	if ui.showFooter {
		view = lipgloss.JoinVertical(lipgloss.Left, view, ui.footer.View())
	}
	return ui.common.Zone.Scan(
		ui.common.Styles.App.Render(view),
	)
}

func (ui *UI) openRepo(rn string) (backend.Repository, error) {
	cfg := ui.common.Config()
	if cfg == nil {
		return nil, errors.New("config is nil")
	}
	repos, err := cfg.Backend.Repositories()
	if err != nil {
		logger.Debugf("ui: failed to list repos: %v", err)
		return nil, err
	}
	for _, r := range repos {
		if r.Name() == rn {
			return r, nil
		}
	}
	return nil, common.ErrMissingRepo
}

func (ui *UI) setRepoCmd(rn string) tea.Cmd {
	return func() tea.Msg {
		r, err := ui.openRepo(rn)
		if err != nil {
			return common.ErrorMsg(err)
		}
		return repo.RepoMsg(r)
	}
}

func (ui *UI) initialRepoCmd(rn string) tea.Cmd {
	return func() tea.Msg {
		r, err := ui.openRepo(rn)
		if err != nil {
			return nil
		}
		return repo.RepoMsg(r)
	}
}
