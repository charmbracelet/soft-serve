package repo

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/soft-serve/git"
	"github.com/charmbracelet/soft-serve/server/proto"
	"github.com/charmbracelet/soft-serve/server/ui/common"
	"github.com/charmbracelet/soft-serve/server/ui/components/selector"
)

// RefMsg is a message that contains a git.Reference.
type RefMsg *git.Reference

// RefItemsMsg is a message that contains a list of RefItem.
type RefItemsMsg struct {
	prefix string
	items  []selector.IdentifiableItem
}

// Refs is a component that displays a list of references.
type Refs struct {
	common    common.Common
	selector  *selector.Selector
	repo      proto.Repository
	ref       *git.Reference
	activeRef *git.Reference
	refPrefix string
	spinner   spinner.Model
	isLoading bool
}

// NewRefs creates a new Refs component.
func NewRefs(common common.Common, refPrefix string) *Refs {
	r := &Refs{
		common:    common,
		refPrefix: refPrefix,
		isLoading: true,
	}
	s := selector.New(common, []selector.IdentifiableItem{}, RefItemDelegate{&common})
	s.SetShowFilter(false)
	s.SetShowHelp(false)
	s.SetShowPagination(false)
	s.SetShowStatusBar(false)
	s.SetShowTitle(false)
	s.SetFilteringEnabled(false)
	s.DisableQuitKeybindings()
	r.selector = s
	sp := spinner.New(spinner.WithSpinner(spinner.Dot),
		spinner.WithStyle(common.Styles.Spinner))
	r.spinner = sp
	return r
}

// TabName returns the name of the tab.
func (r *Refs) TabName() string {
	if r.refPrefix == git.RefsHeads {
		return "Branches"
	} else if r.refPrefix == git.RefsTags {
		return "Tags"
	} else {
		return "Refs"
	}
}

// SetSize implements common.Component.
func (r *Refs) SetSize(width, height int) {
	r.common.SetSize(width, height)
	r.selector.SetSize(width, height)
}

// ShortHelp implements help.KeyMap.
func (r *Refs) ShortHelp() []key.Binding {
	copyKey := r.common.KeyMap.Copy
	copyKey.SetHelp("c", "copy ref")
	k := r.selector.KeyMap
	return []key.Binding{
		r.common.KeyMap.SelectItem,
		k.CursorUp,
		k.CursorDown,
		copyKey,
	}
}

// FullHelp implements help.KeyMap.
func (r *Refs) FullHelp() [][]key.Binding {
	copyKey := r.common.KeyMap.Copy
	copyKey.SetHelp("c", "copy ref")
	k := r.selector.KeyMap
	return [][]key.Binding{
		{r.common.KeyMap.SelectItem},
		{
			k.CursorUp,
			k.CursorDown,
			k.NextPage,
			k.PrevPage,
		},
		{
			k.GoToStart,
			k.GoToEnd,
			copyKey,
		},
	}
}

// Init implements tea.Model.
func (r *Refs) Init() tea.Cmd {
	r.isLoading = true
	return tea.Batch(r.spinner.Tick, r.updateItemsCmd)
}

// Update implements tea.Model.
func (r *Refs) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmds := make([]tea.Cmd, 0)
	switch msg := msg.(type) {
	case RepoMsg:
		r.selector.Select(0)
		r.repo = msg
	case RefMsg:
		r.ref = msg
		cmds = append(cmds, r.Init())
	case RefItemsMsg:
		if r.refPrefix == msg.prefix {
			cmds = append(cmds, r.selector.SetItems(msg.items))
			i := r.selector.SelectedItem()
			if i != nil {
				r.activeRef = i.(RefItem).Reference
			}
			r.isLoading = false
		}
	case selector.ActiveMsg:
		switch sel := msg.IdentifiableItem.(type) {
		case RefItem:
			r.activeRef = sel.Reference
		}
	case selector.SelectMsg:
		switch i := msg.IdentifiableItem.(type) {
		case RefItem:
			cmds = append(cmds,
				switchRefCmd(i.Reference),
				switchTabCmd(&Files{}),
			)
		}
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, r.common.KeyMap.SelectItem):
			cmds = append(cmds, r.selector.SelectItemCmd)
		}
	case EmptyRepoMsg:
		r.ref = nil
		cmds = append(cmds, r.setItems([]selector.IdentifiableItem{}))
	case spinner.TickMsg:
		if r.isLoading && r.spinner.ID() == msg.ID {
			s, cmd := r.spinner.Update(msg)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
			r.spinner = s
		}
	}
	m, cmd := r.selector.Update(msg)
	r.selector = m.(*selector.Selector)
	if cmd != nil {
		cmds = append(cmds, cmd)
	}
	return r, tea.Batch(cmds...)
}

// View implements tea.Model.
func (r *Refs) View() string {
	if r.isLoading {
		return renderLoading(r.common, r.spinner)
	}
	return r.selector.View()
}

// SpinnerID implements common.TabComponent.
func (r *Refs) SpinnerID() int {
	return r.spinner.ID()
}

// StatusBarValue implements statusbar.StatusBar.
func (r *Refs) StatusBarValue() string {
	if r.activeRef == nil {
		return ""
	}
	return r.activeRef.Name().String()
}

// StatusBarInfo implements statusbar.StatusBar.
func (r *Refs) StatusBarInfo() string {
	totalPages := r.selector.TotalPages()
	if totalPages > 1 {
		return fmt.Sprintf("p. %d/%d", r.selector.Page()+1, totalPages)
	}
	return ""
}

func (r *Refs) updateItemsCmd() tea.Msg {
	its := make(RefItems, 0)
	rr, err := r.repo.Open()
	if err != nil {
		return common.ErrorMsg(err)
	}
	refs, err := rr.References()
	if err != nil {
		r.common.Logger.Debugf("ui: error getting references: %v", err)
		return common.ErrorMsg(err)
	}
	for _, ref := range refs {
		if strings.HasPrefix(ref.Name().String(), r.refPrefix) {
			refItem := RefItem{
				Reference: ref,
			}

			if ref.IsTag() {
				refItem.Tag, _ = rr.Tag(ref.Name().Short())
				if refItem.Tag != nil {
					refItem.Commit, _ = refItem.Tag.Commit()
				}
			} else {
				refItem.Commit, _ = rr.CatFileCommit(ref.ID)
			}
			its = append(its, refItem)
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

func (r *Refs) setItems(items []selector.IdentifiableItem) tea.Cmd {
	return func() tea.Msg {
		return RefItemsMsg{
			items:  items,
			prefix: r.refPrefix,
		}
	}
}

func switchRefCmd(ref *git.Reference) tea.Cmd {
	return func() tea.Msg {
		return RefMsg(ref)
	}
}

// UpdateRefCmd gets the repository's HEAD reference and sends a RefMsg.
func UpdateRefCmd(repo proto.Repository) tea.Cmd {
	return func() tea.Msg {
		r, err := repo.Open()
		if err != nil {
			return common.ErrorMsg(err)
		}
		bs, _ := r.Branches()
		if len(bs) == 0 {
			return EmptyRepoMsg{}
		}
		ref, err := r.HEAD()
		if err != nil {
			return common.ErrorMsg(err)
		}
		return RefMsg(ref)
	}
}
