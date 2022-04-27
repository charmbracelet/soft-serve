package tabs

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/soft-serve/ui/common"
)

type SelectTabMsg int

type ActiveTabMsg int

type Tabs struct {
	common    common.Common
	tabs      []string
	activeTab int
}

func New(c common.Common, tabs []string) *Tabs {
	r := &Tabs{
		common:    c,
		tabs:      tabs,
		activeTab: 0,
	}
	return r
}

func (t *Tabs) SetSize(width, height int) {
	t.common.SetSize(width, height)
}

func (t *Tabs) Init() tea.Cmd {
	t.activeTab = 0
	return nil
}

func (t *Tabs) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmds := make([]tea.Cmd, 0)
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "tab":
			t.activeTab = (t.activeTab + 1) % len(t.tabs)
			cmds = append(cmds, t.activeTabCmd)
		case "shift+tab":
			t.activeTab = (t.activeTab - 1 + len(t.tabs)) % len(t.tabs)
			cmds = append(cmds, t.activeTabCmd)
		}
	case SelectTabMsg:
		t.activeTab = int(msg)
	}
	return t, tea.Batch(cmds...)
}

func (t *Tabs) View() string {
	s := strings.Builder{}
	sep := t.common.Styles.TabSeparator
	for i, tab := range t.tabs {
		style := t.common.Styles.TabInactive.Copy()
		if i == t.activeTab {
			style = t.common.Styles.TabActive.Copy()
		}
		s.WriteString(style.Render(tab))
		if i != len(t.tabs)-1 {
			s.WriteString(sep.String())
		}
	}
	return s.String()
}

func (t *Tabs) activeTabCmd() tea.Msg {
	return ActiveTabMsg(t.activeTab)
}

func SelectTabCmd(tab int) tea.Cmd {
	return func() tea.Msg {
		return SelectTabMsg(tab)
	}
}
