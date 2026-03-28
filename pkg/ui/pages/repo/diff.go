package repo

import (
	"fmt"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/soft-serve/git"
	"github.com/charmbracelet/soft-serve/pkg/proto"
	"github.com/charmbracelet/soft-serve/pkg/ui/common"
	"github.com/charmbracelet/soft-serve/pkg/ui/components/code"
)

// DiffMsg is a message sent when the diff content between two refs is loaded.
type DiffMsg struct {
	Content string
	From    string
	To      string
}

// Diff is the diff tab component page.
type Diff struct {
	common    common.Common
	code      *code.Code
	ref       *git.Reference
	repo      proto.Repository
	baseRef   string
	targetRef string
	isLoading bool
	spinner   spinner.Model
}

// NewDiff creates a new Diff model.
func NewDiff(c common.Common) *Diff {
	codeComp := code.New(c, "", ".diff")
	codeComp.NoContentStyle = codeComp.NoContentStyle.SetString("No diff available.")
	s := spinner.New(spinner.WithSpinner(spinner.Dot),
		spinner.WithStyle(c.Styles.Spinner))
	return &Diff{
		common:    c,
		code:      codeComp,
		spinner:   s,
		baseRef:   "HEAD~1",
		targetRef: "HEAD",
		isLoading: true,
	}
}

// Path implements common.TabComponent.
func (d *Diff) Path() string {
	return ""
}

// TabName returns the name of the tab.
func (d *Diff) TabName() string {
	return "Diff"
}

// SetSize implements common.Component.
func (d *Diff) SetSize(width, height int) {
	d.common.SetSize(width, height)
	d.code.SetSize(width, height)
}

// ShortHelp implements help.KeyMap.
func (d *Diff) ShortHelp() []key.Binding {
	return []key.Binding{
		d.common.KeyMap.UpDown,
	}
}

// FullHelp implements help.KeyMap.
func (d *Diff) FullHelp() [][]key.Binding {
	k := d.code.KeyMap
	return [][]key.Binding{
		{
			k.PageDown,
			k.PageUp,
			k.HalfPageDown,
			k.HalfPageUp,
		},
		{
			k.Down,
			k.Up,
			d.common.KeyMap.GotoTop,
			d.common.KeyMap.GotoBottom,
		},
	}
}

// Init implements tea.Model.
func (d *Diff) Init() tea.Cmd {
	d.isLoading = true
	return d.spinner.Tick
}

// Update implements tea.Model.
func (d *Diff) Update(msg tea.Msg) (common.Model, tea.Cmd) {
	cmds := make([]tea.Cmd, 0)
	switch msg := msg.(type) {
	case RepoMsg:
		d.repo = msg
	case RefMsg:
		d.ref = msg
		d.baseRef = "HEAD~1"
		d.targetRef = "HEAD"
		cmds = append(cmds, d.Init(), d.updateDiffCmd)
	case EmptyRepoMsg:
		d.ref = nil
		d.isLoading = false
		cmds = append(cmds, d.code.SetContent("", ".diff"))
	case DiffMsg:
		d.isLoading = false
		d.code.GotoTop()
		content := msg.Content
		if content == "" {
			content = fmt.Sprintf("No differences between %s and %s.", msg.From, msg.To)
		}
		cmds = append(cmds, d.code.SetContent(content, ".diff"))
	case tea.WindowSizeMsg:
		d.SetSize(msg.Width, msg.Height)
	case spinner.TickMsg:
		if d.isLoading && d.spinner.ID() == msg.ID {
			s, cmd := d.spinner.Update(msg)
			d.spinner = s
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
	}
	c, cmd := d.code.Update(msg)
	d.code = c.(*code.Code)
	if cmd != nil {
		cmds = append(cmds, cmd)
	}
	return d, tea.Batch(cmds...)
}

// View implements tea.Model.
func (d *Diff) View() string {
	if d.isLoading {
		return renderLoading(d.common, d.spinner)
	}
	return d.code.View()
}

// SpinnerID implements common.TabComponent.
func (d *Diff) SpinnerID() int {
	return d.spinner.ID()
}

// StatusBarValue implements statusbar.StatusBar.
func (d *Diff) StatusBarValue() string {
	return fmt.Sprintf("%s..%s", d.baseRef, d.targetRef)
}

// StatusBarInfo implements statusbar.StatusBar.
func (d *Diff) StatusBarInfo() string {
	return common.ScrollPercent(d.code.ScrollPosition())
}

func (d *Diff) updateDiffCmd() tea.Msg {
	if d.repo == nil {
		return common.ErrorMsg(common.ErrMissingRepo)
	}
	r, err := d.repo.Open()
	if err != nil {
		d.common.Logger.Debugf("ui: diff: error opening repository: %v", err)
		return DiffMsg{From: d.baseRef, To: d.targetRef}
	}
	content, err := r.DiffRefs(d.baseRef, d.targetRef)
	if err != nil {
		d.common.Logger.Debugf("ui: diff: error computing diff: %v", err)
		return DiffMsg{From: d.baseRef, To: d.targetRef}
	}
	return DiffMsg{
		Content: content,
		From:    d.baseRef,
		To:      d.targetRef,
	}
}
