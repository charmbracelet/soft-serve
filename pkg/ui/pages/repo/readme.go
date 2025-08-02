package repo

import (
	"path/filepath"

	"github.com/charmbracelet/bubbles/v2/key"
	"github.com/charmbracelet/bubbles/v2/spinner"
	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/soft-serve/pkg/backend"
	"github.com/charmbracelet/soft-serve/pkg/proto"
	"github.com/charmbracelet/soft-serve/pkg/ui/common"
	"github.com/charmbracelet/soft-serve/pkg/ui/components/code"
)

// ReadmeMsg is a message sent when the readme is loaded.
type ReadmeMsg struct {
	Content string
	Path    string
}

// Readme is the readme component page.
type Readme struct {
	common     common.Common
	code       *code.Code
	ref        RefMsg
	repo       proto.Repository
	readmePath string
	spinner    spinner.Model
	isLoading  bool
}

// NewReadme creates a new readme model.
func NewReadme(common common.Common) *Readme {
	readme := code.New(common, "", "")
	readme.NoContentStyle = readme.NoContentStyle.SetString("No readme found.")
	readme.UseGlamour = true
	s := spinner.New(spinner.WithSpinner(spinner.Dot),
		spinner.WithStyle(common.Styles.Spinner))
	return &Readme{
		code:      readme,
		common:    common,
		spinner:   s,
		isLoading: true,
	}
}

// Path implements common.TabComponent.
func (r *Readme) Path() string {
	return ""
}

// TabName returns the name of the tab.
func (r *Readme) TabName() string {
	return "Readme"
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
	r.isLoading = true
	return tea.Batch(r.spinner.Tick, r.updateReadmeCmd)
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
	case tea.WindowSizeMsg:
		r.SetSize(msg.Width, msg.Height)
	case EmptyRepoMsg:
		cmds = append(cmds,
			r.code.SetContent(defaultEmptyRepoMsg(r.common.Config(),
				r.repo.Name()), ".md"),
		)
	case ReadmeMsg:
		r.isLoading = false
		r.readmePath = msg.Path
		r.code.GotoTop()
		cmds = append(cmds, r.code.SetContent(msg.Content, msg.Path))
	case spinner.TickMsg:
		if r.isLoading && r.spinner.ID() == msg.ID {
			s, cmd := r.spinner.Update(msg)
			r.spinner = s
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
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
	if r.isLoading {
		return renderLoading(r.common, r.spinner)
	}
	return r.code.View()
}

// SpinnerID implements common.TabComponent.
func (r *Readme) SpinnerID() int {
	return r.spinner.ID()
}

// StatusBarValue implements statusbar.StatusBar.
func (r *Readme) StatusBarValue() string {
	dir := filepath.Dir(r.readmePath)
	if dir == "." || dir == "" {
		return " "
	}
	return dir
}

// StatusBarInfo implements statusbar.StatusBar.
func (r *Readme) StatusBarInfo() string {
	return common.ScrollPercent(r.code.ScrollPosition())
}

func (r *Readme) updateReadmeCmd() tea.Msg {
	m := ReadmeMsg{}
	if r.repo == nil {
		return common.ErrorMsg(common.ErrMissingRepo)
	}
	rm, rp, _ := backend.Readme(r.repo, r.ref)
	m.Content = rm
	m.Path = rp
	return m
}
