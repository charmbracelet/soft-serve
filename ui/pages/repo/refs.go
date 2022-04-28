package repo

import (
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	ggit "github.com/charmbracelet/soft-serve/git"
	"github.com/charmbracelet/soft-serve/ui/common"
	"github.com/charmbracelet/soft-serve/ui/components/selector"
	"github.com/charmbracelet/soft-serve/ui/components/tabs"
	"github.com/charmbracelet/soft-serve/ui/git"
)

type RefItemsMsg struct {
	prefix string
	items  []selector.IdentifiableItem
}

type Refs struct {
	common    common.Common
	selector  *selector.Selector
	repo      git.GitRepo
	ref       *ggit.Reference
	activeRef *ggit.Reference
	refPrefix string
}

func NewRefs(common common.Common, refPrefix string) *Refs {
	r := &Refs{
		common:    common,
		refPrefix: refPrefix,
	}
	s := selector.New(common, []selector.IdentifiableItem{}, RefItemDelegate{common.Styles})
	s.SetShowFilter(false)
	s.SetShowHelp(false)
	s.SetShowPagination(true)
	s.SetShowStatusBar(false)
	s.SetShowTitle(false)
	s.SetFilteringEnabled(false)
	s.DisableQuitKeybindings()
	r.selector = s
	return r
}

func (r *Refs) SetSize(width, height int) {
	r.common.SetSize(width, height)
	r.selector.SetSize(width, height)
}

func (r *Refs) Init() tea.Cmd {
	return r.updateItemsCmd
}

func (r *Refs) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmds := make([]tea.Cmd, 0)
	switch msg := msg.(type) {
	case RepoMsg:
		r.selector.Select(0)
		r.repo = git.GitRepo(msg)
		cmds = append(cmds, r.Init())
	case RefMsg:
		r.ref = msg
		cmds = append(cmds, r.Init())
	case RefItemsMsg:
		cmds = append(cmds, r.selector.SetItems(msg.items))
		r.activeRef = r.selector.SelectedItem().(RefItem).Reference
	case selector.ActiveMsg:
		switch sel := msg.IdentifiableItem.(type) {
		case RefItem:
			r.activeRef = sel.Reference
		}
		cmds = append(cmds, updateStatusBarCmd)
	case selector.SelectMsg:
		switch i := msg.IdentifiableItem.(type) {
		case RefItem:
			cmds = append(cmds,
				switchRefCmd(i.Reference),
				tabs.SelectTabCmd(int(filesTab)),
			)
		}
	case tea.KeyMsg:
		switch msg.String() {
		case "l", "right":
			cmds = append(cmds, r.selector.SelectItem)
		}
	}
	m, cmd := r.selector.Update(msg)
	r.selector = m.(*selector.Selector)
	if cmd != nil {
		cmds = append(cmds, cmd)
	}
	return r, tea.Batch(cmds...)
}

func (r *Refs) View() string {
	return r.selector.View()
}

func (r *Refs) StatusBarValue() string {
	if r.activeRef == nil {
		return ""
	}
	return r.activeRef.Name().String()
}

func (r *Refs) StatusBarInfo() string {
	return ""
}

func (r *Refs) updateItemsCmd() tea.Msg {
	its := make(RefItems, 0)
	refs, err := r.repo.References()
	if err != nil {
		return common.ErrorMsg(err)
	}
	for _, ref := range refs {
		if strings.HasPrefix(ref.Name().String(), r.refPrefix) {
			its = append(its, RefItem{ref})
		}
	}
	sort.Sort(its)
	items := make([]selector.IdentifiableItem, len(its))
	for i, it := range its {
		items[i] = it
	}
	return RefItemsMsg{
		items:  items,
		prefix: r.refPrefix,
	}
}

func switchRefCmd(ref *ggit.Reference) tea.Cmd {
	return func() tea.Msg {
		return RefMsg(ref)
	}
}
