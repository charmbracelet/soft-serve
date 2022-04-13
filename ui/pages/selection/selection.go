package selection

import (
	"fmt"
	"time"

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

type Selection struct {
	s        session.Session
	common   common.Common
	readme   *code.Code
	selector *selector.Selector
}

func New(s session.Session, common common.Common) *Selection {
	sel := &Selection{
		s:        s,
		common:   common,
		readme:   code.New(common, "", ""),
		selector: selector.New(common, []list.Item{}),
	}
	return sel
}

func (s *Selection) SetSize(width, height int) {
	s.common.SetSize(width, height)
	s.readme.SetSize(width, height)
	s.selector.SetSize(width, height)
}

func (s *Selection) ShortHelp() []key.Binding {
	k := s.selector.KeyMap()
	return []key.Binding{
		s.common.Keymap.UpDown,
		s.common.Keymap.Select,
		k.Filter,
		k.ClearFilter,
	}
}

func (s *Selection) FullHelp() [][]key.Binding {
	k := s.selector.KeyMap()
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

func (s *Selection) Init() tea.Cmd {
	items := make([]list.Item, 0)
	cfg := s.s.Config()
	yank := func(text string) *yankable.Yankable {
		return yankable.New(
			lipgloss.NewStyle().Foreground(lipgloss.Color("168")),
			lipgloss.NewStyle().Foreground(lipgloss.Color("168")).SetString("Copied!"),
			text,
		)
	}
	// Put configured repos first
	for _, r := range cfg.Repos {
		items = append(items, selector.Item{
			Title:       r.Name,
			Name:        r.Repo,
			Description: r.Note,
			LastUpdate:  time.Now(),
			URL:         yank(repoUrl(cfg, r.Name)),
		})
	}
	for _, r := range cfg.Source.AllRepos() {
		exists := false
		for _, item := range items {
			item := item.(selector.Item)
			if item.Name == r.Name() {
				exists = true
				break
			}
		}
		if !exists {
			items = append(items, selector.Item{
				Title:       r.Name(),
				Name:        r.Name(),
				Description: "",
				LastUpdate:  time.Now(),
				URL:         yank(repoUrl(cfg, r.Name())),
			})
		}
	}
	return tea.Batch(
		s.selector.Init(),
		s.selector.SetItems(items),
	)
}

func (s *Selection) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmds := make([]tea.Cmd, 0)
	switch msg := msg.(type) {
	case selector.ActiveMsg:
		cmds = append(cmds, s.changeActive(msg))
	default:
		m, cmd := s.selector.Update(msg)
		s.selector = m.(*selector.Selector)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}
	return s, tea.Batch(cmds...)
}

func (s *Selection) View() string {
	return lipgloss.JoinHorizontal(
		lipgloss.Top,
		s.readme.View(),
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
