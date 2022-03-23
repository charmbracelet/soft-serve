package about

import (
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/soft-serve/internal/tui/style"
	"github.com/charmbracelet/soft-serve/pkg/git"
	"github.com/charmbracelet/soft-serve/pkg/tui/common"
	"github.com/charmbracelet/soft-serve/pkg/tui/refs"
	vp "github.com/charmbracelet/soft-serve/pkg/tui/viewport"
	"github.com/muesli/reflow/wrap"
)

type Bubble struct {
	readmeViewport *vp.ViewportBubble
	repo           common.GitRepo
	styles         *style.Styles
	height         int
	heightMargin   int
	width          int
	widthMargin    int
	ref            *git.Reference
}

func NewBubble(repo common.GitRepo, styles *style.Styles, width, wm, height, hm int) *Bubble {
	b := &Bubble{
		readmeViewport: &vp.ViewportBubble{
			Viewport: &viewport.Model{},
		},
		repo:         repo,
		styles:       styles,
		widthMargin:  wm,
		heightMargin: hm,
	}
	b.SetSize(width, height)
	return b
}
func (b *Bubble) Init() tea.Cmd {
	return b.reset
}

func (b *Bubble) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		b.SetSize(msg.Width, msg.Height)
		// XXX: if we find that longer readmes take more than a few
		// milliseconds to render we may need to move Glamour rendering into a
		// command.
		md, err := b.glamourize()
		if err != nil {
			return b, nil
		}
		b.readmeViewport.Viewport.SetContent(md)
	case tea.KeyMsg:
		switch msg.String() {
		case "R":
			return b, b.reset
		}
	case refs.RefMsg:
		b.ref = msg
		return b, b.reset
	}
	rv, cmd := b.readmeViewport.Update(msg)
	b.readmeViewport = rv.(*vp.ViewportBubble)
	cmds = append(cmds, cmd)
	return b, tea.Batch(cmds...)
}

func (b *Bubble) SetSize(w, h int) {
	b.width = w
	b.height = h
	b.readmeViewport.Viewport.Width = w - b.widthMargin
	b.readmeViewport.Viewport.Height = h - b.heightMargin
}

func (b *Bubble) GotoTop() {
	b.readmeViewport.Viewport.GotoTop()
}

func (b *Bubble) View() string {
	return b.readmeViewport.View()
}

func (b *Bubble) Help() []common.HelpEntry {
	return nil
}

func (b *Bubble) glamourize() (string, error) {
	w := b.width - b.widthMargin - b.styles.RepoBody.GetHorizontalFrameSize()
	rm, rp := b.repo.Readme()
	if rm == "" {
		return b.styles.AboutNoReadme.Render("No readme found."), nil
	}
	f, err := common.RenderFile(rp, rm, w)
	if err != nil {
		return "", err
	}
	// For now, hard-wrap long lines in Glamour that would otherwise break the
	// layout when wrapping. This may be due to #43 in Reflow, which has to do
	// with a bug in the way lines longer than the given width are wrapped.
	//
	//     https://github.com/muesli/reflow/issues/43
	//
	// TODO: solve this upstream in Glamour/Reflow.
	return wrap.String(f, w), nil
}

func (b *Bubble) reset() tea.Msg {
	md, err := b.glamourize()
	if err != nil {
		return common.ErrMsg{Err: err}
	}
	head, err := b.repo.HEAD()
	if err != nil {
		return common.ErrMsg{Err: err}
	}
	b.ref = head
	b.readmeViewport.Viewport.SetContent(md)
	b.GotoTop()
	return nil
}
