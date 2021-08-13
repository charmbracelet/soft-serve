package tui

import (
	"fmt"
	"smoothie/tui/bubbles/commits"
	"smoothie/tui/bubbles/repo"
	"smoothie/tui/bubbles/selection"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

type windowMsg struct{}
type errMsg struct{ err error }

func (e errMsg) Error() string {
	return e.err.Error()
}

func (b *Bubble) windowChangesCmd() tea.Msg {
	w := <-b.windowChanges
	b.width = w.Width
	b.height = w.Height
	return windowMsg{}
}

func (b *Bubble) setupCmd() tea.Msg {
	lipgloss.SetColorProfile(termenv.ANSI256)
	b.repos = b.repoSource.AllRepos()
	mes := make([]MenuEntry, 0)
	rs := make([]string, 0)
	for _, me := range b.config.Menu {
		mes = append(mes, me)
	}
	if b.config.ShowAllRepos {
	OUTER:
		for _, r := range b.repos {
			for _, me := range mes {
				if r.Name == me.Repo {
					continue OUTER
				}
			}
			mes = append(mes, MenuEntry{Name: r.Name, Repo: r.Name})
		}
	}
	var tmplConfig *Config
	h := b.height - verticalPadding - viewportHeightConstant
	w := boxRightWidth - 2
	for _, me := range mes {
		if me.Repo == "config" {
			tmplConfig = b.config
		}
		rb := repo.NewBubble(b.repoSource, me.Repo, w, h, tmplConfig)
		initCmd := rb.Init()
		msg := initCmd()
		switch msg := msg.(type) {
		case repo.ErrMsg:
			return errMsg{fmt.Errorf("missing %s: %s", me.Repo, msg.Error)}
		}
		me.bubble = rb
		b.repoMenu = append(b.repoMenu, me)
		rs = append(rs, me.Name)
	}
	b.repoSelect = selection.NewBubble(rs)
	b.boxes[0] = b.repoSelect
	b.commitsLog = commits.NewBubble(
		b.height-verticalPadding-2,
		boxRightWidth-horizontalPadding-2,
		b.repoSource.GetCommits(200),
	)
	b.boxes[1] = b.repoMenu[0].bubble
	b.activeBox = 0
	b.state = loadedState
	return nil
}
