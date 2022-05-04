package copy

import (
	"github.com/aymanbagabas/go-osc52"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// CopyMsg is a message that is sent when the user copies text.
type CopyMsg string

// CopyCmd is a command that copies text to the clipboard using OSC52.
func CopyCmd(output *osc52.Output, str string) tea.Cmd {
	return func() tea.Msg {
		output.Copy(str)
		return CopyMsg(str)
	}
}

type Copy struct {
	output      *osc52.Output
	text        string
	copied      bool
	CopiedStyle lipgloss.Style
	TextStyle   lipgloss.Style
}

func New(output *osc52.Output, text string) *Copy {
	copy := &Copy{
		output: output,
		text:   text,
	}
	return copy
}

func (c *Copy) SetText(text string) {
	c.text = text
}

func (c *Copy) Init() tea.Cmd {
	c.copied = false
	return nil
}

func (c *Copy) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg.(type) {
	case CopyMsg:
		c.copied = true
	default:
		c.copied = false
	}
	return c, nil
}

func (c *Copy) View() string {
	if c.copied {
		return c.CopiedStyle.String()
	}
	return c.TextStyle.Render(c.text)
}

func (c *Copy) CopyCmd() tea.Cmd {
	return CopyCmd(c.output, c.text)
}
