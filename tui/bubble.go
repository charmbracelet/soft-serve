package tui

import (
	"fmt"
	"log"
	"smoothie/git"
	"smoothie/tui/bubbles/commits"
	"smoothie/tui/bubbles/selection"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/gliderlabs/ssh"
)

type sessionState int

const (
	startState sessionState = iota
	errorState
	loadedState
	quittingState
	quitState
)

type Bubble struct {
	state          sessionState
	error          string
	width          int
	height         int
	windowChanges  <-chan ssh.Window
	repoSource     *git.RepoSource
	repos          []*git.Repo
	boxes          []tea.Model
	activeBox      int
	repoSelect     *selection.Bubble
	commitsLog     *commits.Bubble
	readmeViewport *ViewportBubble
}

func NewBubble(width int, height int, windowChanges <-chan ssh.Window, repoSource *git.RepoSource) *Bubble {
	b := &Bubble{
		width:         width,
		height:        height,
		windowChanges: windowChanges,
		repoSource:    repoSource,
		boxes:         make([]tea.Model, 2),
		readmeViewport: &ViewportBubble{
			Viewport: &viewport.Model{
				Width:  boxRightWidth - horizontalPadding - 2,
				Height: height - verticalPadding - viewportHeightConstant,
			},
		},
	}
	b.state = startState
	return b
}

func (b *Bubble) Init() tea.Cmd {
	return tea.Batch(b.windowChangesCmd, b.loadGitCmd)
}

func (b *Bubble) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmds := make([]tea.Cmd, 0)
	// Always allow state, error, info, window resize and quit messages
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return b, tea.Quit
		case "tab":
			b.activeBox = (b.activeBox + 1) % 2
		}
	case errMsg:
		b.error = msg.Error()
		b.state = errorState
		return b, nil
	case windowMsg:
		cmds = append(cmds, b.windowChangesCmd)
	case tea.WindowSizeMsg:
		b.width = msg.Width
		b.height = msg.Height
	case selection.SelectedMsg:
		cmds = append(cmds, b.getRepoCmd(b.repos[msg.Index].Name))
	}
	if b.state == loadedState {
		ab, cmd := b.boxes[b.activeBox].Update(msg)
		b.boxes[b.activeBox] = ab
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}
	return b, tea.Batch(cmds...)
}

func (b *Bubble) viewForBox(i int, width int) string {
	var ls lipgloss.Style
	if i == b.activeBox {
		ls = activeBoxStyle.Width(width)
	} else {
		ls = inactiveBoxStyle.Width(width)
	}
	return ls.Render(b.boxes[i].View())
}

func (b *Bubble) View() string {
	h := headerStyle.Width(b.width - horizontalPadding).Render("Charm Beta")
	f := footerStyle.Render("")
	s := ""
	content := ""
	switch b.state {
	case loadedState:
		lb := b.viewForBox(0, boxLeftWidth)
		rb := b.viewForBox(1, boxRightWidth)
		s += lipgloss.JoinHorizontal(lipgloss.Top, lb, rb)
	case errorState:
		s += errorStyle.Render(fmt.Sprintf("Bummer: %s", b.error))
	default:
		s = normalStyle.Render(fmt.Sprintf("Doing something weird %d", b.state))
	}
	content = h + "\n\n" + s + "\n" + f
	return appBoxStyle.Render(content)
}

func glamourReadme(md string) string {
	tr, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(boxRightWidth-2),
	)
	if err != nil {
		log.Fatal(err)
	}
	mdt, err := tr.Render(md)
	if err != nil {
		return md
	}
	return mdt
}

func SessionHandler(reposPath string) func(ssh.Session) (tea.Model, []tea.ProgramOption) {
	rs := git.NewRepoSource(reposPath, time.Second*10, glamourReadme)
	return func(s ssh.Session) (tea.Model, []tea.ProgramOption) {
		if len(s.Command()) == 0 {
			pty, changes, active := s.Pty()
			if !active {
				return nil, nil
			}
			return NewBubble(
					pty.Window.Width,
					pty.Window.Height,
					changes,
					rs),
				[]tea.ProgramOption{tea.WithAltScreen()}
		}
		return nil, nil
	}
}
