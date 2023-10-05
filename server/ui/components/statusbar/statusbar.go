package statusbar

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/soft-serve/server/ui/common"
	"github.com/muesli/reflow/truncate"
)

// StatusBarMsg is a message sent to the status bar.
type StatusBarMsg struct { //nolint:revive
	Key   string
	Value string
	Info  string
	Extra string
}

// StatusBar is a status bar model.
type StatusBar struct {
	common common.Common
	key    string
	value  string
	info   string
	extra  string
}

// New creates a new status bar component.
func New(c common.Common) *StatusBar {
	s := &StatusBar{
		common: c,
	}
	return s
}

// SetSize implements common.Component.
func (s *StatusBar) SetSize(width, height int) {
	s.common.Width = width
	s.common.Height = height
}

// Init implements tea.Model.
func (s *StatusBar) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model.
func (s *StatusBar) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case StatusBarMsg:
		s.key = msg.Key
		s.value = msg.Value
		s.info = msg.Info
		s.extra = msg.Extra
	}
	return s, nil
}

// View implements tea.Model.
func (s *StatusBar) View() string {
	st := s.common.Styles
	w := lipgloss.Width
	help := s.common.Zone.Mark(
		"repo-help",
		st.StatusBarHelp.Render("? Help"),
	)
	key := st.StatusBarKey.Render(s.key)
	info := ""
	if s.info != "" {
		info = st.StatusBarInfo.Render(s.info)
	}
	branch := st.StatusBarBranch.Render(s.extra)
	maxWidth := s.common.Width - w(key) - w(info) - w(branch) - w(help)
	v := truncate.StringWithTail(s.value, uint(maxWidth-st.StatusBarValue.GetHorizontalFrameSize()), "…")
	value := st.StatusBarValue.
		Width(maxWidth).
		Render(v)

	return lipgloss.NewStyle().MaxWidth(s.common.Width).
		Render(
			lipgloss.JoinHorizontal(lipgloss.Top,
				key,
				value,
				info,
				branch,
				help,
			),
		)
}
