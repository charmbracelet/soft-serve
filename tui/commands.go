package tui

import (
	"smoothie/tui/bubbles/commits"
	"smoothie/tui/bubbles/selection"

	tea "github.com/charmbracelet/bubbletea"
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

func (b *Bubble) loadGitCmd() tea.Msg {
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
