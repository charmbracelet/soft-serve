package commits

import (
	"soft-serve/git"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/dustin/go-humanize"
)

type Bubble struct {
	Commits  []git.RepoCommit
	Height   int
	Width    int
	viewport viewport.Model
}

func NewBubble(height int, width int, rcs []git.RepoCommit) *Bubble {
	b := &Bubble{
		Commits:  rcs,
		viewport: viewport.Model{Height: height, Width: width},
	}
	s := ""
	for _, rc := range rcs {
		s += b.renderCommit(rc) + "\n"
	}
	b.viewport.SetContent(s)
	return b
}

func (b *Bubble) Init() tea.Cmd {
	return nil
}

func (b *Bubble) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmds := make([]tea.Cmd, 0)
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			b.viewport.LineUp(1)
		case "down", "j":
			b.viewport.LineDown(1)
		}
	}
	return b, tea.Batch(cmds...)
}

func (b *Bubble) renderCommit(rc git.RepoCommit) string {
	s := ""
	s += commitRepoNameStyle.Render(rc.Name)
	s += " "
	s += commitDateStyle.Render(humanize.Time(rc.Commit.Author.When))
	s += "\n"
	s += commitCommentStyle.Render(strings.TrimSpace(rc.Commit.Message))
	s += "\n"
	s += commitAuthorStyle.Render(rc.Commit.Author.Name)
	s += " "
	s += commitAuthorEmailStyle.Render(rc.Commit.Author.Email)
	s += " "
	return commitBoxStyle.Width(b.viewport.Width).Render(s)
}

func (b *Bubble) View() string {
	return b.viewport.View()
}
