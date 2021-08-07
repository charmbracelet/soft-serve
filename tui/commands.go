package tui

import (
	"os"
	"path/filepath"
	"smoothie/git"
	"smoothie/tui/bubbles/commits"
	"smoothie/tui/bubbles/selection"

	tea "github.com/charmbracelet/bubbletea"
	gg "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
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

func (b *Bubble) loadGitCmd() tea.Msg {
	cn := "config"
	err := b.repoSource.LoadRepos()
	cr, err := b.repoSource.GetRepo(cn)
	if err == git.ErrMissingRepo {
		cr, err = b.repoSource.InitRepo(cn, false)
		if err != nil {
			return errMsg{err}
		}

		// Add default README and config
		rp := filepath.Join(b.repoSource.Path, cn, "README.md")
		rf, err := os.Create(rp)
		if err != nil {
			return errMsg{err}
		}
		defer rf.Close()
		_, err = rf.WriteString(defaultReadme)
		if err != nil {
			return errMsg{err}
		}
		err = rf.Sync()
		if err != nil {
			return errMsg{err}
		}
		cp := filepath.Join(b.repoSource.Path, cn, "config.json")
		cf, err := os.Create(cp)
		if err != nil {
			return errMsg{err}
		}
		defer cf.Close()
		_, err = cf.WriteString(defaultConfig)
		if err != nil {
			return errMsg{err}
		}
		err = cf.Sync()
		if err != nil {
			return errMsg{err}
		}
		wt, err := cr.Repository.Worktree()
		if err != nil {
			return errMsg{err}
		}
		_, err = wt.Add("README.md")
		if err != nil {
			return errMsg{err}
		}
		_, err = wt.Add("config.json")
		if err != nil {
			return errMsg{err}
		}
		_, err = wt.Commit("Default init", &gg.CommitOptions{
			All: true,
			Author: &object.Signature{
				Name:  "Smoothie Server",
				Email: "vt100@charm.sh",
			},
		})
		if err != nil {
			return errMsg{err}
		}
		err = b.repoSource.LoadRepos()
		if err != nil {
			return errMsg{err}
		}
	} else if err != nil {
		return errMsg{err}
	}
	b.repos = b.repoSource.AllRepos()

	rs := make([]string, 0)
	for _, r := range b.repos {
		rs = append(rs, r.Name)
	}
	b.repoSelect = selection.NewBubble(rs)
	b.boxes[0] = b.repoSelect
	b.commitsLog = commits.NewBubble(
		b.height-verticalPadding-2,
		boxRightWidth-horizontalPadding-2,
		b.repoSource.GetCommits(200),
	)
	b.boxes[1] = b.commitsLog
	b.activeBox = 0
	b.state = loadedState
	return nil
}

func (b *Bubble) getRepoCmd(name string) tea.Cmd {
	return func() tea.Msg {
		r, err := b.repoSource.GetRepo(name)
		if err != nil {
			return errMsg{err}
		}
		b.readmeViewport.Viewport.GotoTop()
		b.readmeViewport.Viewport.Height = b.height - verticalPadding - viewportHeightConstant
		b.readmeViewport.Viewport.Width = boxLeftWidth - 2
		b.readmeViewport.Viewport.SetContent(r.Readme)
		b.boxes[1] = b.readmeViewport
		b.activeBox = 1
		return nil
	}
}
