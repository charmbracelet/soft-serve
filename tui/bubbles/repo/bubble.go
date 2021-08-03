package repo

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/go-git/go-git/v5"
)

type Bubble struct {
	repo *git.Repository
}

func (b *Bubble) Init() tea.Cmd {
	return nil
}

func (b *Bubble) Update(tea.Msg) (tea.Model, tea.Cmd) {
	return nil, nil
}

func (b *Bubble) View() string {
	return "repo"
}
