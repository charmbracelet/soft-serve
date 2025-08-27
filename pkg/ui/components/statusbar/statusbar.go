// Package statusbar provides status bar UI components.
package statusbar

import (
	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/lipgloss/v2"
	"github.com/charmbracelet/soft-serve/pkg/ui/common"
	"github.com/charmbracelet/x/ansi"
)

// Model is a status bar model.
type Model struct {
	common common.Common
	key    string
	value  string
	info   string
	extra  string
}

// New creates a new status bar component.
func New(c common.Common) *Model {
	s := &Model{
		common: c,
	}
	return s
}

// SetSize implements common.Component.
func (s *Model) SetSize(width, height int) {
	s.common.Width = width
	s.common.Height = height
}

// SetStatus sets the status bar status.
func (s *Model) SetStatus(key, value, info, extra string) {
	if key != "" {
		s.key = key
	}
	if value != "" {
		s.value = value
	}
	if info != "" {
		s.info = info
	}
	if extra != "" {
		s.extra = extra
	}
}

// Init implements tea.Model.
func (s *Model) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model.
func (s *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		s.SetSize(msg.Width, msg.Height)
	}
	return s, nil
}

// View implements tea.Model.
func (s *Model) View() string {
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
	v := ansi.Truncate(s.value, maxWidth-st.StatusBarValue.GetHorizontalFrameSize(), "…")
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
