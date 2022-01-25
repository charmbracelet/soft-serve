package refs

import (
	"fmt"
	"io"
	"sort"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/soft-serve/internal/tui/bubbles/git/types"
	"github.com/charmbracelet/soft-serve/internal/tui/style"
	"github.com/go-git/go-git/v5/plumbing"
)

type item struct {
	*plumbing.Reference
}

func (i item) Short() string {
	return i.Name().Short()
}

func (i item) FilterValue() string { return i.Short() }

type items []item

func (cl items) Len() int      { return len(cl) }
func (cl items) Swap(i, j int) { cl[i], cl[j] = cl[j], cl[i] }
func (cl items) Less(i, j int) bool {
	return cl[i].Name().Short() < cl[j].Name().Short()
}

type itemDelegate struct {
	style *style.Styles
}

func (d itemDelegate) Height() int                               { return 1 }
func (d itemDelegate) Spacing() int                              { return 0 }
func (d itemDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd { return nil }
func (d itemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(item)
	if !ok {
		return
	}

	branch := i.Short()
	leftMargin := d.style.LogItemSelector.GetMarginLeft() + d.style.LogItemSelector.GetWidth() + d.style.LogItemInactive.GetMarginLeft()
	branch = types.TruncateString(branch, m.Width()-leftMargin, "…")
	if index == m.Index() {
		fmt.Fprint(w, d.style.LogItemSelector.Render(">")+
			d.style.LogItemActive.Render(branch))
	} else {
		fmt.Fprint(w, d.style.LogItemSelector.Render(" ")+
			d.style.LogItemInactive.Render(branch))
	}
}

type Bubble struct {
	repo         types.Repo
	list         list.Model
	style        *style.Styles
	width        int
	widthMargin  int
	height       int
	heightMargin int
}

func NewBubble(repo types.Repo, style *style.Styles, width, widthMargin, height, heightMargin int) *Bubble {
	l := list.NewModel([]list.Item{}, itemDelegate{style}, width-widthMargin, height-heightMargin)
	l.SetShowFilter(false)
	l.SetShowHelp(false)
	l.SetShowPagination(false)
	l.SetShowStatusBar(false)
	l.SetShowTitle(false)
	b := &Bubble{
		repo:         repo,
		style:        style,
		width:        width,
		height:       height,
		widthMargin:  widthMargin,
		heightMargin: heightMargin,
		list:         l,
	}
	b.SetSize(width, height)
	return b
}

func (b *Bubble) SetBranch(ref *plumbing.Reference) {
	b.repo.SetReference(ref)
}

func (b *Bubble) Init() tea.Cmd {
	return nil
}

func (b *Bubble) SetSize(width, height int) {
	b.width = width
	b.height = height
	b.list.SetSize(width-b.widthMargin, height-b.heightMargin)
}

func (b *Bubble) Help() []types.HelpEntry {
	return []types.HelpEntry{
		{"enter", "select"},
	}
}

func (b *Bubble) UpdateItems() tea.Cmd {
	its := make(items, 0)
	ri, err := b.repo.Repository().References()
	if err != nil {
		return nil
	}
	err = ri.ForEach(func(r *plumbing.Reference) error {
		if r.Name().Short() != "HEAD" {
			its = append(its, item{r})
		}
		return nil
	})
	if err != nil {
		return nil
	}
	sort.Sort(its)
	itt := make([]list.Item, len(its))
	for i, it := range its {
		itt[i] = it
	}
	return b.list.SetItems(itt)
}

func (b *Bubble) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmds := make([]tea.Cmd, 0)
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		b.SetSize(msg.Width, msg.Height)

	case tea.KeyMsg:
		switch msg.String() {
		case "R":
			cmds = append(cmds, b.UpdateItems())
		case "down", "j":
			b.list.CursorDown()
		case "up", "k":
			b.list.CursorUp()
		case "enter":
			if b.list.Index() >= 0 {
				ref := b.list.SelectedItem().(item).Reference
				b.SetBranch(ref)
			}
		}
	}
	return b, tea.Batch(cmds...)
}

func (b *Bubble) View() string {
	return b.list.View()
}
