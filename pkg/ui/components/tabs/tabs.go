package tabs

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/lipgloss/v2"
	"github.com/charmbracelet/soft-serve/pkg/ui/common"
)

// SelectTabMsg is a message that contains the index of the tab to select.
type SelectTabMsg int

// ActiveTabMsg is a message that contains the index of the current active tab.
type ActiveTabMsg int

// IdentifiableTab is a struct that encapsulates a tabs shown value with an id.
type IdentifiableTab struct {
	ID    string
	Value string
}

// SetTabValueMsg is a message that sets a tabs shown value.
type SetTabValueMsg struct {
	ID    string
	Value string
}

// Tabs is bubbletea component that displays a list of tabs.
type Tabs struct {
	common       common.Common
	tabs         []IdentifiableTab
	activeTab    int
	TabSeparator lipgloss.Style
	TabInactive  lipgloss.Style
	TabActive    lipgloss.Style
	TabDot       lipgloss.Style
	UseDot       bool
}

// New creates a new Tabs component.
func New(c common.Common, tabs []string) *Tabs {
	ts := make([]IdentifiableTab, 0)
	for _, t := range tabs {
		ts = append(ts, IdentifiableTab{
			ID:    t,
			Value: t,
		})
	}
	r := &Tabs{
		common:       c,
		tabs:         ts,
		activeTab:    0,
		TabSeparator: c.Styles.TabSeparator,
		TabInactive:  c.Styles.TabInactive,
		TabActive:    c.Styles.TabActive,
	}
	return r
}

// SetSize implements common.Component.
func (t *Tabs) SetSize(width, height int) {
	t.common.SetSize(width, height)
}

// Init implements tea.Model.
func (t *Tabs) Init() tea.Cmd {
	t.activeTab = 0
	return nil
}

// Update implements tea.Model.
func (t *Tabs) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmds := make([]tea.Cmd, 0)
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "tab":
			t.activeTab = (t.activeTab + 1) % len(t.tabs)
			cmds = append(cmds, t.activeTabCmd)
		case "shift+tab":
			t.activeTab = (t.activeTab - 1 + len(t.tabs)) % len(t.tabs)
			cmds = append(cmds, t.activeTabCmd)
		}
	case tea.MouseClickMsg:
		switch msg.Button {
		case tea.MouseLeft:
			for i, tab := range t.tabs {
				if t.common.Zone.Get(tab.ID).InBounds(msg) {
					t.activeTab = i
					cmds = append(cmds, t.activeTabCmd)
				}
			}
		}
	case SetTabValueMsg:
		for i, tab := range t.tabs {
			if tab.ID == msg.ID {
				t.tabs[i].Value = msg.Value
				break
			}
		}
	case SelectTabMsg:
		tab := int(msg)
		if tab >= 0 && tab < len(t.tabs) {
			t.activeTab = int(msg)
		}
	}
	return t, tea.Batch(cmds...)
}

// ResetTabNames resets all tab names to their IDs.
func (t *Tabs) ResetTabNames() {
	for i := range t.tabs {
		t.tabs[i].Value = t.tabs[i].ID
	}
}

// View implements tea.Model.
func (t *Tabs) View() string {
	s := strings.Builder{}
	sep := t.TabSeparator
	for i, tab := range t.tabs {
		style := t.TabInactive
		prefix := "  "
		if i == t.activeTab {
			style = t.TabActive
			prefix = t.TabDot.Render("• ")
		}
		if t.UseDot {
			s.WriteString(prefix)
		}
		s.WriteString(
			t.common.Zone.Mark(
				tab.ID,
				style.Render(tab.Value),
			),
		)
		if i != len(t.tabs)-1 {
			s.WriteString(sep.String())
		}
	}
	return lipgloss.NewStyle().
		MaxWidth(t.common.Width).
		Render(s.String())
}

func (t *Tabs) activeTabCmd() tea.Msg {
	return ActiveTabMsg(t.activeTab)
}

// SelectTabCmd is a bubbletea command that selects the tab at the given index.
func SelectTabCmd(tab int) tea.Cmd {
	return func() tea.Msg {
		return SelectTabMsg(tab)
	}
}
