package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/soft/config"
	br "github.com/charmbracelet/soft/tui/bubbles/repo"
	"github.com/charmbracelet/soft/tui/bubbles/selection"
	"github.com/muesli/termenv"
)

type errMsg struct{ err error }

func (e errMsg) Error() string {
	return e.err.Error()
}

func (b *Bubble) setupCmd() tea.Msg {
	if b.config == nil || b.config.Source == nil {
		return errMsg{err: fmt.Errorf("config not set")}
	}
	lipgloss.SetColorProfile(termenv.ANSI256)
	mes, err := b.menuEntriesFromSource()
	if err != nil {
		return errMsg{err}
	}
	if len(mes) == 0 {
		return errMsg{fmt.Errorf("no repos found")}
	}
	b.repoMenu = mes
	rs := make([]string, 0)
	for _, m := range mes {
		rs = append(rs, m.Name)
	}
	b.repoSelect = selection.NewBubble(rs, b.styles)
	b.boxes[0] = b.repoSelect

	// Jump to an initial repo
	ir := -1
	if b.initialRepo != "" {
		for i, me := range b.repoMenu {
			if me.Repo == b.initialRepo {
				ir = i
			}
		}
	}
	if ir == -1 {
		b.boxes[1] = b.repoMenu[0].bubble
		b.activeBox = 0
	} else {
		b.boxes[1] = b.repoMenu[ir].bubble
		b.repoSelect.SelectedItem = ir
		b.activeBox = 1
	}

	b.state = loadedState
	return nil
}

func (b *Bubble) menuEntriesFromSource() ([]MenuEntry, error) {
	mes := make([]MenuEntry, 0)
	for _, r := range b.config.Repos {
		me, err := b.newMenuEntry(r.Name, r.Repo)
		if err != nil {
			return nil, err
		}
		mes = append(mes, me)
	}
	rs := b.config.Source.AllRepos()
OUTER:
	for _, r := range rs {
		for _, me := range mes {
			if r.Name == me.Repo {
				continue OUTER
			}
		}
		me, err := b.newMenuEntry(r.Name, r.Name)
		if err != nil {
			return nil, err
		}
		mes = append(mes, me)
	}
	return mes, nil
}

func (b *Bubble) newMenuEntry(name string, repo string) (MenuEntry, error) {
	var tmplConfig *config.Config
	me := MenuEntry{Name: name, Repo: repo}
	width := b.width
	boxLeftWidth := b.styles.Menu.GetWidth() + b.styles.Menu.GetHorizontalFrameSize()
	// TODO: also send this along with a tea.WindowSizeMsg
	var heightMargin = lipgloss.Height(b.headerView()) +
		lipgloss.Height(b.footerView()) +
		b.styles.RepoBody.GetVerticalFrameSize() +
		b.styles.App.GetVerticalMargins()
	if repo == "config" {
		tmplConfig = b.config
	}
	rb := br.NewBubble(
		b.config.Source,
		me.Repo,
		b.styles,
		width,
		boxLeftWidth,
		b.height,
		heightMargin,
		tmplConfig,
	)
	rb.Host = b.config.Host
	rb.Port = b.config.Port
	initCmd := rb.Init()
	msg := initCmd()
	switch msg := msg.(type) {
	case br.ErrMsg:
		return me, fmt.Errorf("missing %s: %s", me.Repo, msg.Error)
	}
	me.bubble = rb
	return me, nil
}
