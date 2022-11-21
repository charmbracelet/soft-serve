package selection

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/soft-serve/config"
	gm "github.com/charmbracelet/soft-serve/server/git"
	"github.com/charmbracelet/soft-serve/ui/common"
	"github.com/charmbracelet/soft-serve/ui/components/code"
	"github.com/charmbracelet/soft-serve/ui/components/selector"
	"github.com/charmbracelet/soft-serve/ui/components/tabs"
	"github.com/charmbracelet/soft-serve/ui/git"
	"github.com/gliderlabs/ssh"
)

type pane int

const (
	selectorPane pane = iota
	readmePane
	lastPane
)

func (p pane) String() string {
	return []string{
		"Repositories",
		"About",
	}[p]
}

// Selection is the model for the selection screen/page.
type Selection struct {
	cfg          *config.Config
	pk           ssh.PublicKey
	common       common.Common
	readme       *code.Code
	readmeHeight int
	selector     *selector.Selector
	activePane   pane
	tabs         *tabs.Tabs
}

// New creates a new selection model.
func New(cfg *config.Config, pk ssh.PublicKey, common common.Common) *Selection {
	ts := make([]string, lastPane)
	for i, b := range []pane{selectorPane, readmePane} {
		ts[i] = b.String()
	}
	t := tabs.New(common, ts)
	t.TabSeparator = lipgloss.NewStyle()
	t.TabInactive = common.Styles.TopLevelNormalTab.Copy()
	t.TabActive = common.Styles.TopLevelActiveTab.Copy()
	t.TabDot = common.Styles.TopLevelActiveTabDot.Copy()
	t.UseDot = true
	sel := &Selection{
		cfg:        cfg,
		pk:         pk,
		common:     common,
		activePane: selectorPane, // start with the selector focused
		tabs:       t,
	}
	readme := code.New(common, "", "")
	readme.NoContentStyle = readme.NoContentStyle.SetString("No readme found.")
	selector := selector.New(common,
		[]selector.IdentifiableItem{},
		ItemDelegate{&common, &sel.activePane})
	selector.SetShowTitle(false)
	selector.SetShowHelp(false)
	selector.SetShowStatusBar(false)
	selector.DisableQuitKeybindings()
	sel.selector = selector
	sel.readme = readme
	return sel
}

func (s *Selection) getMargins() (wm, hm int) {
	wm = 0
	hm = s.common.Styles.Tabs.GetVerticalFrameSize() +
		s.common.Styles.Tabs.GetHeight()
	if s.activePane == selectorPane && s.IsFiltering() {
		// hide tabs when filtering
		hm = 0
	}
	return
}

// FilterState returns the current filter state.
func (s *Selection) FilterState() list.FilterState {
	return s.selector.FilterState()
}

// SetSize implements common.Component.
func (s *Selection) SetSize(width, height int) {
	s.common.SetSize(width, height)
	wm, hm := s.getMargins()
	s.tabs.SetSize(width, height-hm)
	s.selector.SetSize(width-wm, height-hm)
	s.readme.SetSize(width-wm, height-hm-1) // -1 for readme status line
}

// IsFiltering returns true if the selector is currently filtering.
func (s *Selection) IsFiltering() bool {
	return s.FilterState() == list.Filtering
}

// ShortHelp implements help.KeyMap.
func (s *Selection) ShortHelp() []key.Binding {
	k := s.selector.KeyMap
	kb := make([]key.Binding, 0)
	kb = append(kb,
		s.common.KeyMap.UpDown,
		s.common.KeyMap.Section,
	)
	if s.activePane == selectorPane {
		copyKey := s.common.KeyMap.Copy
		copyKey.SetHelp("c", "copy command")
		kb = append(kb,
			s.common.KeyMap.Select,
			k.Filter,
			k.ClearFilter,
			copyKey,
		)
	}
	return kb
}

// FullHelp implements help.KeyMap.
func (s *Selection) FullHelp() [][]key.Binding {
	b := [][]key.Binding{
		{
			s.common.KeyMap.Section,
		},
	}
	switch s.activePane {
	case readmePane:
		k := s.readme.KeyMap
		b = append(b, []key.Binding{
			k.PageDown,
			k.PageUp,
		})
		b = append(b, []key.Binding{
			k.HalfPageDown,
			k.HalfPageUp,
		})
		b = append(b, []key.Binding{
			k.Down,
			k.Up,
		})
	case selectorPane:
		copyKey := s.common.KeyMap.Copy
		copyKey.SetHelp("c", "copy command")
		k := s.selector.KeyMap
		if !s.IsFiltering() {
			b[0] = append(b[0],
				s.common.KeyMap.Select,
				copyKey,
			)
		}
		b = append(b, []key.Binding{
			k.CursorUp,
			k.CursorDown,
		})
		b = append(b, []key.Binding{
			k.NextPage,
			k.PrevPage,
			k.GoToStart,
			k.GoToEnd,
		})
		b = append(b, []key.Binding{
			k.Filter,
			k.ClearFilter,
			k.CancelWhileFiltering,
			k.AcceptWhileFiltering,
		})
	}
	return b
}

// Init implements tea.Model.
func (s *Selection) Init() tea.Cmd {
	var readmeCmd tea.Cmd
	items := make([]selector.IdentifiableItem, 0)
	cfg := s.cfg
	pk := s.pk
	// Put configured repos first
	for _, r := range cfg.Repos {
		acc := cfg.AuthRepo(r.Repo, pk)
		if r.Private && acc < gm.ReadOnlyAccess {
			continue
		}
		repo, err := cfg.Source.GetRepo(r.Repo)
		if err != nil {
			continue
		}
		items = append(items, Item{
			repo: repo,
			cmd:  git.RepoURL(cfg.Host, cfg.Port, r.Repo),
		})
	}
	for _, r := range cfg.Source.AllRepos() {
		if r.Repo() == "config" {
			rm, rp := r.Readme()
			s.readmeHeight = strings.Count(rm, "\n")
			readmeCmd = s.readme.SetContent(rm, rp)
		}
		acc := cfg.AuthRepo(r.Repo(), pk)
		if r.IsPrivate() && acc < gm.ReadOnlyAccess {
			continue
		}
		exists := false
		lc, err := r.Commit("HEAD")
		if err != nil {
			return common.ErrorCmd(err)
		}
		lastUpdate := lc.Committer.When
		if lastUpdate.IsZero() {
			lastUpdate = lc.Author.When
		}
		for i, item := range items {
			item := item.(Item)
			if item.repo.Repo() == r.Repo() {
				exists = true
				item.lastUpdate = lastUpdate
				items[i] = item
				break
			}
		}
		if !exists {
			items = append(items, Item{
				repo:       r,
				lastUpdate: lastUpdate,
				cmd:        git.RepoURL(cfg.Host, cfg.Port, r.Name()),
			})
		}
	}
	return tea.Batch(
		s.selector.Init(),
		s.selector.SetItems(items),
		readmeCmd,
	)
}

// Update implements tea.Model.
func (s *Selection) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmds := make([]tea.Cmd, 0)
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		r, cmd := s.readme.Update(msg)
		s.readme = r.(*code.Code)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
		m, cmd := s.selector.Update(msg)
		s.selector = m.(*selector.Selector)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	case tea.KeyMsg, tea.MouseMsg:
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch {
			case key.Matches(msg, s.common.KeyMap.Back):
				cmds = append(cmds, s.selector.Init())
			}
		}
		t, cmd := s.tabs.Update(msg)
		s.tabs = t.(*tabs.Tabs)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	case tabs.ActiveTabMsg:
		s.activePane = pane(msg)
	}
	switch s.activePane {
	case readmePane:
		r, cmd := s.readme.Update(msg)
		s.readme = r.(*code.Code)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	case selectorPane:
		m, cmd := s.selector.Update(msg)
		s.selector = m.(*selector.Selector)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}
	return s, tea.Batch(cmds...)
}

// View implements tea.Model.
func (s *Selection) View() string {
	var view string
	wm, hm := s.getMargins()
	switch s.activePane {
	case selectorPane:
		ss := lipgloss.NewStyle().
			Width(s.common.Width - wm).
			Height(s.common.Height - hm)
		view = ss.Render(s.selector.View())
	case readmePane:
		rs := lipgloss.NewStyle().
			Height(s.common.Height - hm)
		status := fmt.Sprintf("☰ %.f%%", s.readme.ScrollPercent()*100)
		readmeStatus := lipgloss.NewStyle().
			Align(lipgloss.Right).
			Width(s.common.Width - wm).
			Foreground(s.common.Styles.InactiveBorderColor).
			Render(status)
		view = rs.Render(lipgloss.JoinVertical(lipgloss.Left,
			s.readme.View(),
			readmeStatus,
		))
	}
	if s.activePane != selectorPane || s.FilterState() != list.Filtering {
		tabs := s.common.Styles.Tabs.Render(s.tabs.View())
		view = lipgloss.JoinVertical(lipgloss.Left,
			tabs,
			view,
		)
	}
	return lipgloss.JoinVertical(
		lipgloss.Left,
		view,
	)
}
