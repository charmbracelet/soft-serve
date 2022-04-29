package statusbar

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/soft-serve/ui/common"
	"github.com/muesli/reflow/truncate"
)

type StatusBarMsg struct {
	Key    string
	Value  string
	Info   string
	Branch string
}

type StatusBar struct {
	common common.Common
	msg    StatusBarMsg
}

type Model interface {
	StatusBarValue() string
	StatusBarInfo() string
}

func New(c common.Common) *StatusBar {
	s := &StatusBar{
		common: c,
	}
	return s
}

func (s *StatusBar) SetSize(width, height int) {
	s.common.Width = width
	s.common.Height = height
}

func (s *StatusBar) Init() tea.Cmd {
	return nil
}

func (s *StatusBar) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case StatusBarMsg:
		s.msg = msg
	}
	return s, nil
}

func (s *StatusBar) View() string {
	st := s.common.Styles
	w := lipgloss.Width
	key := st.StatusBarKey.Render(s.msg.Key)
	info := ""
	if s.msg.Info != "" {
		info = st.StatusBarInfo.Render(s.msg.Info)
	}
	branch := st.StatusBarBranch.Render(s.msg.Branch)
	maxWidth := s.common.Width - w(key) - w(info) - w(branch)
	v := truncate.StringWithTail(s.msg.Value, uint(maxWidth-st.StatusBarValue.GetHorizontalFrameSize()), "…")
	value := st.StatusBarValue.
		Width(maxWidth).
		Render(v)

	return lipgloss.JoinHorizontal(lipgloss.Top,
		key,
		value,
		info,
		branch,
	)
}
