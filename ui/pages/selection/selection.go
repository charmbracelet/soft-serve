package selection

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	appCfg "github.com/charmbracelet/soft-serve/config"
	"github.com/charmbracelet/soft-serve/ui/common"
	"github.com/charmbracelet/soft-serve/ui/components/code"
	"github.com/charmbracelet/soft-serve/ui/components/selector"
	"github.com/charmbracelet/soft-serve/ui/components/yankable"
	"github.com/charmbracelet/soft-serve/ui/session"
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
		ItemDelegate{common.Styles, &sel.activeBox})
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
		s.common.Keymap.UpDown,
		s.common.Keymap.Section,
	)
	if s.activeBox == selectorBox {
		kb = append(kb,
			s.common.Keymap.Select,
			k.Filter,
			k.ClearFilter,
		)
	}
	return kb
}

// FullHelp implements help.KeyMap.
// TODO implement full help on ?
func (s *Selection) FullHelp() [][]key.Binding {
	k := s.selector.KeyMap
	return [][]key.Binding{
		{
			k.CursorUp,
			k.CursorDown,
			k.NextPage,
			k.PrevPage,
			k.GoToStart,
			k.GoToEnd,
		},
		{
			k.Filter,
			k.ClearFilter,
			k.CancelWhileFiltering,
			k.AcceptWhileFiltering,
			k.ShowFullHelp,
			k.CloseFullHelp,
		},
		// Ignore the following keys:
		// k.Quit,
		// k.ForceQuit,
	}
}

// Init implements tea.Model.
func (s *Selection) Init() tea.Cmd {
	session := s.s.Session()
	environ := session.Environ()
	termExists := false
	// Add TERM using pty.Term if it's not already set.
	for _, env := range environ {
		if strings.HasPrefix(env, "TERM=") {
			termExists = true
			break
		}
	}
	if !termExists {
		pty, _, _ := session.Pty()
		environ = append(environ, fmt.Sprintf("TERM=%s", pty.Term))
	}
	items := make([]list.Item, 0)
	cfg := s.s.Config()
	// TODO clean up this
	yank := func(text string) *yankable.Yankable {
		return yankable.New(
			session,
			environ,
			lipgloss.NewStyle().Foreground(lipgloss.Color("168")),
			lipgloss.NewStyle().Foreground(lipgloss.Color("168")).SetString("Copied!"),
			text,
		)
	}
	// Put configured repos first
	for _, r := range cfg.Repos {
		items = append(items, Item{
			name: r.Name,
			repo: r.Repo,
			desc: r.Note,
			url:  yank(repoUrl(cfg, r.Repo)),
		})
	}
	for _, r := range cfg.Source.AllRepos() {
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
		for _, item := range items {
			item := item.(Item)
			if item.repo == r.Name() {
				exists = true
				item.lastUpdate = lastUpdate
				break
			}
		}
		if !exists {
			items = append(items, Item{
				name:       r.Name(),
				repo:       r.Name(),
				desc:       "",
				lastUpdate: lastUpdate,
				url:        yank(repoUrl(cfg, r.Name())),
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
		case key.Matches(msg, s.common.Keymap.Section):
			s.activeBox = (s.activeBox + 1) % 2
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
	cfg := s.s.Config()
	r, err := cfg.Source.GetRepo(string(msg))
	if err != nil {
		return common.ErrorCmd(err)
	}
	rm, rp := r.Readme()
	return s.readme.SetContent(rm, rp)
}

func repoUrl(cfg *appCfg.Config, name string) string {
	port := ""
	if cfg.Port != 22 {
		port += fmt.Sprintf(":%d", cfg.Port)
	}
	return fmt.Sprintf("git clone ssh://%s/%s", cfg.Host+port, name)
}
