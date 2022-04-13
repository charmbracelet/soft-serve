package footer

import (
	"github.com/charmbracelet/bubbles/help"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/soft-serve/ui/common"
)

type Footer struct {
	common common.Common
	help   help.Model
	keymap help.KeyMap
}

func New(c common.Common, keymap help.KeyMap) *Footer {
	h := help.New()
	h.Styles.ShortKey = c.Styles.HelpKey
	h.Styles.FullKey = c.Styles.HelpKey
	f := &Footer{
		common: c,
		help:   h,
		keymap: keymap,
	}
	return f
}

func (f *Footer) SetSize(width, height int) {
	f.common.Width = width
	f.common.Height = height
}

func (f *Footer) Init() tea.Cmd {
	return nil
}

func (f *Footer) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return f, nil
}

func (f *Footer) View() string {
	if f.keymap == nil {
		return ""
	}
	s := f.common.Styles.Footer.Copy().Width(f.common.Width)
	return s.Render(f.help.View(f.keymap))
}
