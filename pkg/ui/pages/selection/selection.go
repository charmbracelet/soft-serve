package selection

import (
	"fmt"
	"sort"

	"github.com/charmbracelet/bubbles/v2/key"
	"github.com/charmbracelet/bubbles/v2/list"
	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/lipgloss/v2"
	"github.com/charmbracelet/soft-serve/pkg/access"
	"github.com/charmbracelet/soft-serve/pkg/backend"
	"github.com/charmbracelet/soft-serve/pkg/ui/common"
	"github.com/charmbracelet/soft-serve/pkg/ui/components/code"
	"github.com/charmbracelet/soft-serve/pkg/ui/components/selector"
	"github.com/charmbracelet/soft-serve/pkg/ui/components/tabs"
)

const (
	defaultNoContent = "No readme found.\n\nCreate a `.soft-serve` repository and add a `README.md` file to display readme."
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
	common     common.Common
	readme     *code.Code
	selector   *selector.Selector
	activePane pane
	tabs       *tabs.Tabs
}

// New creates a new selection model.
func New(c common.Common) *Selection {
	ts := make([]string, lastPane)
	for i, b := range []pane{selectorPane, readmePane} {
		ts[i] = b.String()
	}
	t := tabs.New(c, ts)
	t.TabSeparator = lipgloss.NewStyle()
	t.TabInactive = c.Styles.TopLevelNormalTab
	t.TabActive = c.Styles.TopLevelActiveTab
	t.TabDot = c.Styles.TopLevelActiveTabDot
	t.UseDot = true
	sel := &Selection{
		common:     c,
		activePane: selectorPane, // start with the selector focused
		tabs:       t,
	}
	readme := code.New(c, "", "")
	readme.UseGlamour = true
	readme.NoContentStyle = c.Styles.NoContent.
		SetString(defaultNoContent)
	selector := selector.New(c,
		[]selector.IdentifiableItem{},
		NewItemDelegate(&c, &sel.activePane))
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
	cfg := s.common.Config()
	if cfg == nil {
		return nil
	}

	ctx := s.common.Context()
	be := s.common.Backend()
	pk := s.common.PublicKey()
	if pk == nil && !be.AllowKeyless(ctx) {
		return nil
	}

	repos, err := be.Repositories(ctx)
	if err != nil {
		return common.ErrorCmd(err)
	}
	sortedItems := make(Items, 0)
	for _, r := range repos {
		if r.Name() == ".soft-serve" {
			readme, path, err := backend.Readme(r, nil)
			if err != nil {
				continue
			}

			readmeCmd = s.readme.SetContent(readme, path)
		}

		if r.IsHidden() {
			continue
		}
		al := be.AccessLevelByPublicKey(ctx, r.Name(), pk)
		if al >= access.ReadOnlyAccess {
			item, err := NewItem(s.common, r)
			if err != nil {
				s.common.Logger.Debugf("ui: failed to create item for %s: %v", r.Name(), err)
				continue
			}
			sortedItems = append(sortedItems, item)
		}
	}
	sort.Sort(sortedItems)
	items := make([]selector.IdentifiableItem, len(sortedItems))
	for i, it := range sortedItems {
		items[i] = it
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
	case tea.KeyPressMsg, tea.MouseMsg:
		switch msg := msg.(type) {
		case tea.KeyPressMsg:
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
