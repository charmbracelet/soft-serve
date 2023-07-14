package repo

import (
	"fmt"
	"path/filepath"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/soft-serve/server/backend"
	"github.com/charmbracelet/soft-serve/server/proto"
	"github.com/charmbracelet/soft-serve/server/ui/common"
	"github.com/charmbracelet/soft-serve/server/ui/components/code"
)

// ReadmeMsg is a message sent when the readme is loaded.
type ReadmeMsg struct {
	Msg tea.Msg
}

// Readme is the readme component page.
type Readme struct {
	common     common.Common
	code       *code.Code
	ref        RefMsg
	repo       proto.Repository
	readmePath string
}

// NewReadme creates a new readme model.
func NewReadme(common common.Common) *Readme {
	readme := code.New(common, "", "")
	readme.NoContentStyle = readme.NoContentStyle.Copy().SetString("No readme found.")
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
			r.common.KeyMap.GotoTop,
			r.common.KeyMap.GotoBottom,
		},
	}
	return b
}

// Init implements tea.Model.
func (r *Readme) Init() tea.Cmd {
	return r.updateReadmeCmd
}

// Update implements tea.Model.
func (r *Readme) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmds := make([]tea.Cmd, 0)
	switch msg := msg.(type) {
	case RepoMsg:
		r.repo = msg
	case RefMsg:
		r.ref = msg
		cmds = append(cmds, r.Init())
	case EmptyRepoMsg:
		r.code.SetContent(defaultEmptyRepoMsg(r.common.Config(),
			r.repo.Name()), ".md")
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
	dir := filepath.Dir(r.readmePath)
	if dir == "." {
		return ""
	}
	return dir
}

// StatusBarInfo implements statusbar.StatusBar.
func (r *Readme) StatusBarInfo() string {
	return fmt.Sprintf("☰ %.f%%", r.code.ScrollPercent()*100)
}

func (r *Readme) updateReadmeCmd() tea.Msg {
	m := ReadmeMsg{}
	if r.repo == nil {
		return common.ErrorCmd(common.ErrMissingRepo)
	}
	rm, rp, _ := backend.Readme(r.repo)
	r.readmePath = rp
	r.code.GotoTop()
	cmd := r.code.SetContent(rm, rp)
	if cmd != nil {
		m.Msg = cmd()
	}
	return m
}
