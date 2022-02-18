package tui

import (
	"bytes"
	"fmt"
	"text/template"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	gitypes "github.com/charmbracelet/soft-serve/internal/tui/bubbles/git/types"
	"github.com/charmbracelet/soft-serve/internal/tui/bubbles/repo"
	"github.com/charmbracelet/soft-serve/internal/tui/bubbles/selection"
	gm "github.com/charmbracelet/wish/git"
)

type errMsg struct{ err error }

func (e errMsg) Error() string {
	return e.err.Error()
}

func (b *Bubble) setupCmd() tea.Msg {
	if b.config == nil || b.config.Source == nil {
		return errMsg{err: fmt.Errorf("config not set")}
	}
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
	for _, cr := range b.config.Repos {
		acc := b.config.AuthRepo(cr.Repo, b.session.PublicKey())
		if acc == gm.NoAccess && cr.Repo != "config" {
			continue
		}
		me, err := b.newMenuEntry(cr.Name, cr.Repo)
		if err != nil {
			return nil, err
		}
		mes = append(mes, me)
	}
	for _, r := range b.config.Source.AllRepos() {
		var found bool
		rn := r.Name()
		for _, me := range mes {
			if me.Repo == rn {
				found = true
			}
		}
		if !found {
			acc := b.config.AuthRepo(rn, b.session.PublicKey())
			if acc == gm.NoAccess {
				continue
			}
			me, err := b.newMenuEntry(rn, rn)
			if err != nil {
				return nil, err
			}
			mes = append(mes, me)
		}
	}
	return mes, nil
}

func (b *Bubble) newMenuEntry(name string, rn string) (MenuEntry, error) {
	me := MenuEntry{Name: name, Repo: rn}
	r, err := b.config.Source.GetRepo(rn)
	if err != nil {
		return me, err
	}
	if rn == "config" {
		md, err := templatize(r.Readme, b.config)
		if err != nil {
			return me, err
		}
		r.Readme = md
	}
	boxLeftWidth := b.styles.Menu.GetWidth() + b.styles.Menu.GetHorizontalFrameSize()
	// TODO: also send this along with a tea.WindowSizeMsg
	var heightMargin = lipgloss.Height(b.headerView()) +
		lipgloss.Height(b.footerView()) +
		b.styles.RepoBody.GetVerticalFrameSize() +
		b.styles.App.GetVerticalMargins()
	rb := repo.NewBubble(r, b.config.Host, b.config.Port, b.styles, b.width, boxLeftWidth, b.height, heightMargin)
	initCmd := rb.Init()
	msg := initCmd()
	switch msg := msg.(type) {
	case gitypes.ErrMsg:
		return me, fmt.Errorf("missing %s: %s", me.Repo, msg.Err.Error())
	}
	me.bubble = rb
	return me, nil
}

func templatize(mdt string, tmpl interface{}) (string, error) {
	t, err := template.New("readme").Parse(mdt)
	if err != nil {
		return "", err
	}
	buf := &bytes.Buffer{}
	err = t.Execute(buf, tmpl)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}
