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
