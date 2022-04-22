package footer

import (
	"github.com/charmbracelet/bubbles/help"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/soft-serve/ui/common"
)

// Footer is a Bubble Tea model that displays help and other info.
type Footer struct {
	common common.Common
	help   help.Model
	keymap help.KeyMap
}

// New creates a new Footer.
func New(c common.Common, keymap help.KeyMap) *Footer {
	h := help.New()
	h.Styles.ShortKey = c.Styles.HelpKey
	h.Styles.ShortDesc = c.Styles.HelpValue
	h.Styles.FullKey = c.Styles.HelpKey
	h.Styles.FullDesc = c.Styles.HelpValue
	f := &Footer{
		common: c,
		help:   h,
		keymap: keymap,
	}
	return f
}

// SetSize implements common.Component.
func (f *Footer) SetSize(width, height int) {
	f.common.Width = width
	f.common.Height = height
}

// Init implements tea.Model.
func (f *Footer) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model.
func (f *Footer) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return f, nil
}

// View implements tea.Model.
func (f *Footer) View() string {
	if f.keymap == nil {
		return ""
	}
	s := f.common.Styles.Footer.Copy().Width(f.common.Width)
	return s.Render(f.help.View(f.keymap))
}
