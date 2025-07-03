package repo

import (
	"fmt"

	gitm "github.com/aymanbagabas/git-module"
	"github.com/charmbracelet/bubbles/v2/key"
	"github.com/charmbracelet/bubbles/v2/spinner"
	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/lipgloss/v2"
	"github.com/charmbracelet/soft-serve/git"
	"github.com/charmbracelet/soft-serve/pkg/proto"
	"github.com/charmbracelet/soft-serve/pkg/ui/common"
	"github.com/charmbracelet/soft-serve/pkg/ui/components/code"
	"github.com/charmbracelet/soft-serve/pkg/ui/components/selector"
)

type stashState int

const (
	stashStateLoading stashState = iota
	stashStateList
	stashStatePatch
)

// StashListMsg is a message sent when the stash list is loaded.
type StashListMsg []*gitm.Stash

// StashPatchMsg is a message sent when the stash patch is loaded.
type StashPatchMsg struct{ *git.Diff }

// Stash is the stash component page.
type Stash struct {
	common       common.Common
	code         *code.Code
	ref          RefMsg
	repo         proto.Repository
	spinner      spinner.Model
	list         *selector.Selector
	state        stashState
	currentPatch StashPatchMsg
}

// NewStash creates a new stash model.
func NewStash(common common.Common) *Stash {
	code := code.New(common, "", "")
	s := spinner.New(spinner.WithSpinner(spinner.Dot),
		spinner.WithStyle(common.Styles.Spinner))
	selector := selector.New(common, []selector.IdentifiableItem{}, StashItemDelegate{&common})
	selector.SetShowFilter(false)
	selector.SetShowHelp(false)
	selector.SetShowPagination(false)
	selector.SetShowStatusBar(false)
	selector.SetShowTitle(false)
	selector.SetFilteringEnabled(false)
	selector.DisableQuitKeybindings()
	selector.KeyMap.NextPage = common.KeyMap.NextPage
	selector.KeyMap.PrevPage = common.KeyMap.PrevPage
	return &Stash{
		code:    code,
		common:  common,
		spinner: s,
		list:    selector,
	}
}

// Path implements common.TabComponent.
func (s *Stash) Path() string {
	return ""
}

// TabName returns the name of the tab.
func (s *Stash) TabName() string {
	return "Stash"
}

// SetSize implements common.Component.
func (s *Stash) SetSize(width, height int) {
	s.common.SetSize(width, height)
	s.code.SetSize(width, height)
	s.list.SetSize(width, height)
}

// ShortHelp implements help.KeyMap.
func (s *Stash) ShortHelp() []key.Binding {
	return []key.Binding{
		s.common.KeyMap.Select,
		s.common.KeyMap.Back,
		s.common.KeyMap.UpDown,
	}
}

// FullHelp implements help.KeyMap.
func (s *Stash) FullHelp() [][]key.Binding {
	b := [][]key.Binding{
		{
			s.common.KeyMap.Select,
			s.common.KeyMap.Back,
			s.common.KeyMap.Copy,
		},
		{
			s.code.KeyMap.Down,
			s.code.KeyMap.Up,
			s.common.KeyMap.GotoTop,
			s.common.KeyMap.GotoBottom,
		},
	}
	return b
}

// StatusBarValue implements common.Component.
func (s *Stash) StatusBarValue() string {
	item, ok := s.list.SelectedItem().(StashItem)
	if !ok {
		return " "
	}
	idx := s.list.Index()
	return fmt.Sprintf("stash@{%d}: %s", idx, item.Title())
}

// StatusBarInfo implements common.Component.
func (s *Stash) StatusBarInfo() string {
	switch s.state {
	case stashStateList:
		totalPages := s.list.TotalPages()
		if totalPages <= 1 {
			return "p. 1/1"
		}
		return fmt.Sprintf("p. %d/%d", s.list.Page()+1, totalPages)
	case stashStatePatch:
		return common.ScrollPercent(s.code.ScrollPosition())
	default:
		return ""
	}
}

// SpinnerID implements common.Component.
func (s *Stash) SpinnerID() int {
	return s.spinner.ID()
}

// Init initializes the model.
func (s *Stash) Init() tea.Cmd {
	s.state = stashStateLoading
	return tea.Batch(s.spinner.Tick, s.fetchStash)
}

// Update updates the model.
func (s *Stash) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmds := make([]tea.Cmd, 0)
	switch msg := msg.(type) {
	case RepoMsg:
		s.repo = msg
	case RefMsg:
		s.ref = msg
		s.list.Select(0)
		cmds = append(cmds, s.Init())
	case tea.WindowSizeMsg:
		s.SetSize(msg.Width, msg.Height)
	case spinner.TickMsg:
		if s.state == stashStateLoading && s.spinner.ID() == msg.ID {
			sp, cmd := s.spinner.Update(msg)
			s.spinner = sp
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
	case tea.KeyPressMsg:
		switch s.state {
		case stashStateList:
			switch {
			case key.Matches(msg, s.common.KeyMap.BackItem):
				cmds = append(cmds, goBackCmd)
			case key.Matches(msg, s.common.KeyMap.Copy):
				cmds = append(cmds, copyCmd(s.list.SelectedItem().(StashItem).Title(), "Stash message copied to clipboard"))
			}
		case stashStatePatch:
			switch {
			case key.Matches(msg, s.common.KeyMap.BackItem):
				cmds = append(cmds, goBackCmd)
			case key.Matches(msg, s.common.KeyMap.Copy):
				if s.currentPatch.Diff != nil {
					patch := s.currentPatch.Diff
					cmds = append(cmds, copyCmd(patch.Patch(), "Stash patch copied to clipboard"))
				}
			}
		}
	case StashListMsg:
		s.state = stashStateList
		items := make([]selector.IdentifiableItem, len(msg))
		for i, stash := range msg {
			items[i] = StashItem{stash}
		}
		cmds = append(cmds, s.list.SetItems(items))
	case StashPatchMsg:
		s.state = stashStatePatch
		s.currentPatch = msg
		if msg.Diff != nil {
			title := s.common.Styles.Stash.Title.Render(s.list.SelectedItem().(StashItem).Title())
			content := lipgloss.JoinVertical(lipgloss.Left,
				title,
				"",
				renderSummary(msg.Diff, s.common.Styles, s.common.Width),
				renderDiff(msg.Diff, s.common.Width),
			)
			cmds = append(cmds, s.code.SetContent(content, ".diff"))
			s.code.GotoTop()
		}
	case selector.SelectMsg:
		switch msg.IdentifiableItem.(type) {
		case StashItem:
			cmds = append(cmds, s.fetchStashPatch)
		}
	case GoBackMsg:
		if s.state == stashStateList {
			s.list.Select(0)
		}
		s.state = stashStateList
	}
	switch s.state {
	case stashStateList:
		l, cmd := s.list.Update(msg)
		s.list = l.(*selector.Selector)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	case stashStatePatch:
		c, cmd := s.code.Update(msg)
		s.code = c.(*code.Code)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}
	return s, tea.Batch(cmds...)
}

// View returns the view.
func (s *Stash) View() string {
	switch s.state {
	case stashStateLoading:
		return renderLoading(s.common, s.spinner)
	case stashStateList:
		return s.list.View()
	case stashStatePatch:
		return s.code.View()
	}
	return ""
}

func (s *Stash) fetchStash() tea.Msg {
	if s.repo == nil {
		return StashListMsg(nil)
	}

	r, err := s.repo.Open()
	if err != nil {
		return common.ErrorMsg(err)
	}

	stash, err := r.StashList()
	if err != nil {
		return common.ErrorMsg(err)
	}

	return StashListMsg(stash)
}

func (s *Stash) fetchStashPatch() tea.Msg {
	if s.repo == nil {
		return StashPatchMsg{nil}
	}

	r, err := s.repo.Open()
	if err != nil {
		return common.ErrorMsg(err)
	}

	diff, err := r.StashDiff(s.list.Index())
	if err != nil {
		return common.ErrorMsg(err)
	}

	return StashPatchMsg{diff}
}
