package repo

import (
	"fmt"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/soft-serve/ui/common"
	"github.com/charmbracelet/soft-serve/ui/components/code"
	"github.com/charmbracelet/soft-serve/ui/git"
)

type ReadmeMsg struct{}

// Readme is the readme component page.
type Readme struct {
	common common.Common
	code   *code.Code
	ref    RefMsg
	repo   git.GitRepo
}

// NewReadme creates a new readme model.
func NewReadme(common common.Common) *Readme {
	readme := code.New(common, "", "")
	readme.NoContentStyle = readme.NoContentStyle.SetString("No readme found.")
	return &Readme{
		code:   readme,
		common: common,
	}
}

// SetSize implements common.Component.
func (r *Readme) SetSize(width, height int) {
	r.common.SetSize(width, height)
	r.code.SetSize(width, height)
}

// ShortHelp implements help.KeyMap.
func (r *Readme) ShortHelp() []key.Binding {
	b := []key.Binding{
		r.common.KeyMap.UpDown,
	}
	return b
}

// FullHelp implements help.KeyMap.
func (r *Readme) FullHelp() [][]key.Binding {
	k := r.code.KeyMap
	b := [][]key.Binding{
		{
			k.PageDown,
			k.PageUp,
			k.HalfPageDown,
			k.HalfPageUp,
		},
		{
			k.Down,
			k.Up,
		},
	}
	return b
}

// Init implements tea.Model.
func (r *Readme) Init() tea.Cmd {
	if r.repo == nil {
		return common.ErrorCmd(git.ErrMissingRepo)
	}
	rm, rp := r.repo.Readme()
	r.code.GotoTop()
	return tea.Batch(
		r.code.SetContent(rm, rp),
		r.updateReadmeCmd,
	)
}

// Update implements tea.Model.
func (r *Readme) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmds := make([]tea.Cmd, 0)
	switch msg := msg.(type) {
	case RepoMsg:
		r.repo = git.GitRepo(msg)
		cmds = append(cmds, r.Init())
	case RefMsg:
		r.ref = msg
		cmds = append(cmds, r.Init())
	}
	c, cmd := r.code.Update(msg)
	r.code = c.(*code.Code)
	if cmd != nil {
		cmds = append(cmds, cmd)
	}
	return r, tea.Batch(cmds...)
}

// View implements tea.Model.
func (r *Readme) View() string {
	return r.code.View()
}

// StatusBarValue implements statusbar.StatusBar.
func (r *Readme) StatusBarValue() string {
	return ""
}

// StatusBarInfo implements statusbar.StatusBar.
func (r *Readme) StatusBarInfo() string {
	return fmt.Sprintf("☰ %.f%%", r.code.ScrollPercent()*100)
}

func (r *Readme) updateReadmeCmd() tea.Msg {
	return ReadmeMsg{}
}
