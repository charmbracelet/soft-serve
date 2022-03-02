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

type RefMsg = *plumbing.Reference

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
	s := d.style
	i, ok := listItem.(item)
	if !ok {
		return
	}

	ref := i.Short()
	if i.Name().IsTag() {
		ref = s.RefItemTag.Render(ref)
	}
	ref = s.RefItemBranch.Render(ref)
	refMaxWidth := m.Width() -
		s.RefItemSelector.GetMarginLeft() -
		s.RefItemSelector.GetWidth() -
		s.RefItemInactive.GetMarginLeft()
	ref = types.TruncateString(ref, refMaxWidth, "…")
	if index == m.Index() {
		fmt.Fprint(w, s.RefItemSelector.Render(">")+
			s.RefItemActive.Render(ref))
	} else {
		fmt.Fprint(w, s.LogItemSelector.Render(" ")+
			s.RefItemInactive.Render(ref))
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
	ref          *plumbing.Reference
}

func NewBubble(repo types.Repo, styles *style.Styles, width, widthMargin, height, heightMargin int) *Bubble {
	l := list.NewModel([]list.Item{}, itemDelegate{styles}, width-widthMargin, height-heightMargin)
	l.SetShowFilter(false)
	l.SetShowHelp(false)
	l.SetShowPagination(true)
	l.SetShowStatusBar(false)
	l.SetShowTitle(false)
	l.SetFilteringEnabled(false)
	l.DisableQuitKeybindings()
	b := &Bubble{
		repo:         repo,
		style:        styles,
		width:        width,
		height:       height,
		widthMargin:  widthMargin,
		heightMargin: heightMargin,
		list:         l,
		ref:          repo.GetHEAD(),
	}
	b.SetSize(width, height)
	return b
}

func (b *Bubble) SetBranch(ref *plumbing.Reference) (tea.Model, tea.Cmd) {
	b.ref = ref
	return b, func() tea.Msg {
		return RefMsg(ref)
	}
}

func (b *Bubble) reset() tea.Cmd {
	cmd := b.updateItems()
	b.SetSize(b.width, b.height)
	return cmd
}

func (b *Bubble) Init() tea.Cmd {
	return nil
}

func (b *Bubble) SetSize(width, height int) {
	b.width = width
	b.height = height
	b.list.SetSize(width-b.widthMargin, height-b.heightMargin)
	b.list.Styles.PaginationStyle = b.style.RefPaginator.Copy().Width(width - b.widthMargin)
}

func (b *Bubble) Help() []types.HelpEntry {
	return nil
}

func (b *Bubble) updateItems() tea.Cmd {
	its := make(items, 0)
	tags := make(items, 0)
	for _, r := range b.repo.GetReferences() {
		if r.Type() != plumbing.HashReference {
			continue
		}
		n := r.Name()
		if n.IsTag() {
			tags = append(tags, item{r})
		} else if n.IsBranch() {
			its = append(its, item{r})
		}
	}
	sort.Sort(its)
	sort.Sort(tags)
	its = append(its, tags...)
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
		case "B":
			return b, b.reset()
		case "enter", "right", "l":
			if b.list.Index() >= 0 {
				ref := b.list.SelectedItem().(item).Reference
				return b.SetBranch(ref)
			}
		}
	}

	l, cmd := b.list.Update(msg)
	b.list = l
	cmds = append(cmds, cmd)

	return b, tea.Batch(cmds...)
}

func (b *Bubble) View() string {
	return b.list.View()
}
