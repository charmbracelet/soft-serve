package yankable

import (
	"fmt"
	"log"
	"strings"

	"github.com/aymanbagabas/go-osc52"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/gliderlabs/ssh"
)

type Yankable struct {
	yankStyle lipgloss.Style
	style     lipgloss.Style
	text      string
	clicked   bool
	osc52     *osc52.Output
}

func New(s ssh.Session, style, yankStyle lipgloss.Style, text string) *Yankable {
	environ := s.Environ()
	termExists := false
	for _, env := range environ {
		if strings.HasPrefix(env, "TERM=") {
			termExists = true
			break
		}
	}
	if !termExists {
		pty, _, _ := s.Pty()
		environ = append(environ, fmt.Sprintf("TERM=%s", pty.Term))
	}
	log.Print(environ)
	return &Yankable{
		yankStyle: yankStyle,
		style:     style,
		text:      text,
		clicked:   false,
		osc52:     osc52.NewOutput(s, environ),
	}
}

func (y *Yankable) SetText(text string) {
	y.text = text
}

func (y *Yankable) Init() tea.Cmd {
	return nil
}

func (y *Yankable) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.MouseMsg:
		switch msg.Type {
		case tea.MouseRight:
			y.clicked = true
			return y, y.copy()
		}
	default:
		y.clicked = false
	}
	return y, nil
}

func (y *Yankable) View() string {
	if y.clicked {
		return y.yankStyle.String()
	}
	return y.style.Render(y.text)
}

func (y *Yankable) copy() tea.Cmd {
	y.osc52.Copy(y.text)
	return nil
}
