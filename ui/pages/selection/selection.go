package selection

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/soft-serve/ui/common"
	"github.com/charmbracelet/soft-serve/ui/components/code"
	"github.com/charmbracelet/soft-serve/ui/components/selector"
	"github.com/charmbracelet/soft-serve/ui/git"
	"github.com/charmbracelet/soft-serve/ui/session"
	wgit "github.com/charmbracelet/wish/git"
)

type box int

const (
	readmeBox box = iota
	selectorBox
)

// Selection is the model for the selection screen/page.
type Selection struct {
	s         session.Session
	common    common.Common
	readme    *code.Code
	selector  *selector.Selector
	activeBox box
}

// New creates a new selection model.
func New(s session.Session, common common.Common) *Selection {
	sel := &Selection{
		s:         s,
		common:    common,
		activeBox: selectorBox, // start with the selector focused
	}
	readme := code.New(common, "", "")
	readme.NoContentStyle = readme.NoContentStyle.SetString("No readme found.")
	selector := selector.New(common,
		[]selector.IdentifiableItem{},
		ItemDelegate{&common, &sel.activeBox})
	selector.SetShowTitle(false)
	selector.SetShowHelp(false)
	selector.SetShowStatusBar(false)
	selector.DisableQuitKeybindings()
	sel.selector = selector
	sel.readme = readme
	return sel
}

// SetSize implements common.Component.
func (s *Selection) SetSize(width, height int) {
	s.common.SetSize(width, height)
	sw := s.common.Styles.SelectorBox.GetWidth()
	wm := sw +
		s.common.Styles.SelectorBox.GetHorizontalFrameSize() +
		s.common.Styles.ReadmeBox.GetHorizontalFrameSize() +
		// +1 to get wrapping to work.
		// This is needed because the readme box width has to be -1 from the
		// readme style in order for wrapping to not break.
		1
	hm := s.common.Styles.ReadmeBox.GetVerticalFrameSize()
	s.readme.SetSize(width-wm, height-hm)
	s.selector.SetSize(sw, height)
}

// ShortHelp implements help.KeyMap.
func (s *Selection) ShortHelp() []key.Binding {
	k := s.selector.KeyMap
	kb := make([]key.Binding, 0)
	kb = append(kb,
		s.common.KeyMap.UpDown,
		s.common.KeyMap.Section,
	)
	if s.activeBox == selectorBox {
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
	switch s.activeBox {
	case readmeBox:
		k := s.readme.KeyMap
		return [][]key.Binding{
			{
				k.PageDown,
				k.PageUp,
			},
			{
				k.HalfPageDown,
				k.HalfPageUp,
			},
			{
				k.Down,
				k.Up,
			},
		}
	case selectorBox:
		copyKey := s.common.KeyMap.Copy
		copyKey.SetHelp("c", "copy command")
		k := s.selector.KeyMap
		return [][]key.Binding{
			{
				s.common.KeyMap.Select,
				copyKey,
			},
			{
				k.CursorUp,
				k.CursorDown,
			},
			{
				k.NextPage,
				k.PrevPage,
			},
			{
				k.GoToStart,
				k.GoToEnd,
			},
			{
				k.Filter,
				k.ClearFilter,
				k.CancelWhileFiltering,
				k.AcceptWhileFiltering,
			},
		}
	}
	return [][]key.Binding{}
}

// Init implements tea.Model.
func (s *Selection) Init() tea.Cmd {
	items := make([]selector.IdentifiableItem, 0)
	cfg := s.s.Config()
	pk := s.s.PublicKey()
	// Put configured repos first
	for _, r := range cfg.Repos {
		if r.Private && cfg.AuthRepo(r.Repo, pk) < wgit.AdminAccess {
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
		if r.IsPrivate() && cfg.AuthRepo(r.Repo(), pk) < wgit.AdminAccess {
			continue
		}
		exists := false
		head, err := r.HEAD()
		if err != nil {
			return common.ErrorCmd(err)
		}
		lc, err := r.CommitsByPage(head, 1, 1)
		if err != nil {
			return common.ErrorCmd(err)
		}
		lastUpdate := lc[0].Committer.When
		if lastUpdate.IsZero() {
			lastUpdate = lc[0].Author.When
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
	case selector.ActiveMsg:
		cmds = append(cmds, s.changeActive(msg))
		// reset readme position when active item change
		s.readme.GotoTop()
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, s.common.KeyMap.Section):
			s.activeBox = (s.activeBox + 1) % 2
		case key.Matches(msg, s.common.KeyMap.Back):
			cmds = append(cmds, s.selector.Init())
		}
	}
	switch s.activeBox {
	case readmeBox:
		r, cmd := s.readme.Update(msg)
		s.readme = r.(*code.Code)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	case selectorBox:
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
	wm := s.common.Styles.SelectorBox.GetWidth() +
		s.common.Styles.SelectorBox.GetHorizontalFrameSize() +
		s.common.Styles.ReadmeBox.GetHorizontalFrameSize()
	hm := s.common.Styles.ReadmeBox.GetVerticalFrameSize()
	rs := s.common.Styles.ReadmeBox.Copy().
		Width(s.common.Width - wm).
		Height(s.common.Height - hm)
	if s.activeBox == readmeBox {
		rs.BorderForeground(s.common.Styles.ActiveBorderColor)
	}
	readme := rs.Render(s.readme.View())
	return lipgloss.JoinHorizontal(
		lipgloss.Top,
		readme,
		s.selector.View(),
	)
}

func (s *Selection) changeActive(msg selector.ActiveMsg) tea.Cmd {
	r := msg.IdentifiableItem.(Item).repo
	rm, rp := r.Readme()
	return s.readme.SetContent(rm, rp)
}
