package tui

import (
	"fmt"
	"log"
	"soft-serve/tui/bubbles/repo"
	"soft-serve/tui/bubbles/selection"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

type windowMsg struct{}
type errMsg struct{ err error }

func (e errMsg) Error() string {
	return e.err.Error()
}

func (b *Bubble) setupCmd() tea.Msg {
	ct := time.Now()
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
	for _, me := range mes {
		if me.Repo == "config" {
			tmplConfig = b.config
		}
		width := b.width
		boxLeftWidth := b.styles.Menu.GetWidth() + b.styles.Menu.GetHorizontalFrameSize()
		// TODO: also send this along with a tea.WindowSizeMsg
		var heightMargin = lipgloss.Height(b.headerView()) +
			lipgloss.Height(b.footerView()) +
			b.styles.RepoBody.GetVerticalFrameSize() +
			b.styles.App.GetVerticalMargins()
		rb := repo.NewBubble(b.repoSource, me.Repo, b.styles, width, boxLeftWidth, b.height, heightMargin, tmplConfig)
		rb.Host = b.config.Host
		rb.Port = b.config.Port
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
	b.repoSelect = selection.NewBubble(rs, b.styles)
	b.boxes[0] = b.repoSelect
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
	log.Printf("App bubble loaded in %s", time.Since(ct))
	return nil
}
