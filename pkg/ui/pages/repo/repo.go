package repo

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/v2/help"
	"github.com/charmbracelet/bubbles/v2/key"
	"github.com/charmbracelet/bubbles/v2/spinner"
	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/lipgloss/v2"
	"github.com/charmbracelet/soft-serve/git"
	"github.com/charmbracelet/soft-serve/pkg/proto"
	"github.com/charmbracelet/soft-serve/pkg/ui/common"
	"github.com/charmbracelet/soft-serve/pkg/ui/components/footer"
	"github.com/charmbracelet/soft-serve/pkg/ui/components/selector"
	"github.com/charmbracelet/soft-serve/pkg/ui/components/statusbar"
	"github.com/charmbracelet/soft-serve/pkg/ui/components/tabs"
)

type state int

const (
	loadingState state = iota
	readyState
)

// EmptyRepoMsg is a message to indicate that the repository is empty.
type EmptyRepoMsg struct{}

// CopyURLMsg is a message to copy the URL of the current repository.
type CopyURLMsg struct{}

// RepoMsg is a message that contains a git.Repository.
type RepoMsg proto.Repository //nolint:revive

// GoBackMsg is a message to go back to the previous view.
type GoBackMsg struct{}

// CopyMsg is a message to indicate copied text.
type CopyMsg struct {
	Text    string
	Message string
}

// SwitchTabMsg is a message to switch tabs.
type SwitchTabMsg common.TabComponent

// Repo is a view for a git repository.
type Repo struct {
	common       common.Common
	selectedRepo proto.Repository
	activeTab    int
	tabs         *tabs.Tabs
	statusbar    *statusbar.Model
	panes        []common.TabComponent
	ref          *git.Reference
	state        state
	spinner      spinner.Model
	panesReady   []bool
}

// New returns a new Repo.
func New(c common.Common, comps ...common.TabComponent) *Repo {
	sb := statusbar.New(c)
	ts := make([]string, 0)
	for _, c := range comps {
		ts = append(ts, c.TabName())
	}
	c.Logger = c.Logger.WithPrefix("ui.repo")
	tb := tabs.New(c, ts)
	// Make sure the order matches the order of tab constants above.
	s := spinner.New(spinner.WithSpinner(spinner.Dot),
		spinner.WithStyle(c.Styles.Spinner))
	r := &Repo{
		common:     c,
		tabs:       tb,
		statusbar:  sb,
		panes:      comps,
		state:      loadingState,
		spinner:    s,
		panesReady: make([]bool, len(comps)),
	}
	return r
}

func (r *Repo) getMargins() (int, int) {
	hh := lipgloss.Height(r.headerView())
	hm := r.common.Styles.Repo.Body.GetVerticalFrameSize() +
		hh +
		r.common.Styles.Repo.Header.GetVerticalFrameSize() +
		r.common.Styles.StatusBar.GetHeight()
	return 0, hm
}

// SetSize implements common.Component.
func (r *Repo) SetSize(width, height int) {
	r.common.SetSize(width, height)
	_, hm := r.getMargins()
	r.tabs.SetSize(width, height-hm)
	r.statusbar.SetSize(width, height-hm)
	for _, p := range r.panes {
		p.SetSize(width, height-hm)
	}
}

// Path returns the current component path.
func (r *Repo) Path() string {
	return r.panes[r.activeTab].Path()
}

func (r *Repo) commonHelp() []key.Binding {
	b := make([]key.Binding, 0)
	back := r.common.KeyMap.Back
	back.SetHelp("esc", "back to menu")
	tab := r.common.KeyMap.Section
	tab.SetHelp("tab", "switch tab")
	b = append(b, back)
	b = append(b, tab)
	return b
}

// ShortHelp implements help.KeyMap.
func (r *Repo) ShortHelp() []key.Binding {
	b := r.commonHelp()
	b = append(b, r.panes[r.activeTab].(help.KeyMap).ShortHelp()...)
	return b
}

// FullHelp implements help.KeyMap.
func (r *Repo) FullHelp() [][]key.Binding {
	b := make([][]key.Binding, 0)
	b = append(b, r.commonHelp())
	b = append(b, r.panes[r.activeTab].(help.KeyMap).FullHelp()...)
	return b
}

// Init implements tea.View.
func (r *Repo) Init() tea.Cmd {
	r.state = loadingState
	r.activeTab = 0
	return tea.Batch(
		r.tabs.Init(),
		r.statusbar.Init(),
		r.spinner.Tick,
	)
}

// Update implements tea.Model.
func (r *Repo) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmds := make([]tea.Cmd, 0)
	switch msg := msg.(type) {
	case RepoMsg:
		// Set the state to loading when we get a new repository.
		r.selectedRepo = msg
		cmds = append(cmds,
			r.Init(),
			// This will set the selected repo in each pane's model.
			r.updateModels(msg),
		)
	case RefMsg:
		r.ref = msg
		cmds = append(cmds, r.updateModels(msg))
		r.state = readyState
	case tabs.SelectTabMsg:
		r.activeTab = int(msg)
		t, cmd := r.tabs.Update(msg)
		r.tabs = t.(*tabs.Tabs)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	case tabs.ActiveTabMsg:
		r.activeTab = int(msg)
	case tea.KeyPressMsg, tea.MouseClickMsg:
		t, cmd := r.tabs.Update(msg)
		r.tabs = t.(*tabs.Tabs)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
		if r.selectedRepo != nil {
			urlID := fmt.Sprintf("%s-url", r.selectedRepo.Name())
			cmd := r.common.CloneCmd(r.common.Config().SSH.PublicURL, r.selectedRepo.Name())
			if msg, ok := msg.(tea.MouseMsg); ok && r.common.Zone.Get(urlID).InBounds(msg) {
				cmds = append(cmds, copyCmd(cmd, "Command copied to clipboard"))
			}
		}
		switch msg := msg.(type) {
		case tea.MouseClickMsg:
			switch msg.Button {
			case tea.MouseLeft:
				switch {
				case r.common.Zone.Get("repo-help").InBounds(msg):
					cmds = append(cmds, footer.ToggleFooterCmd)
				}
			case tea.MouseRight:
				switch {
				case r.common.Zone.Get("repo-main").InBounds(msg):
					cmds = append(cmds, goBackCmd)
				}
			}
		}
		switch msg := msg.(type) {
		case tea.KeyPressMsg:
			switch {
			case key.Matches(msg, r.common.KeyMap.Back):
				cmds = append(cmds, goBackCmd)
			}
		}
	case CopyMsg:
		txt := msg.Text
		if cfg := r.common.Config(); cfg != nil {
			cmds = append(cmds, tea.SetClipboard(txt))
		}
		r.statusbar.SetStatus("", msg.Message, "", "")
	case ReadmeMsg:
		cmds = append(cmds, r.updateTabComponent(&Readme{}, msg))
	case FileItemsMsg, FileContentMsg:
		cmds = append(cmds, r.updateTabComponent(&Files{}, msg))
	case LogItemsMsg, LogDiffMsg, LogCountMsg:
		cmds = append(cmds, r.updateTabComponent(&Log{}, msg))
	case RefItemsMsg:
		cmds = append(cmds, r.updateTabComponent(&Refs{refPrefix: msg.prefix}, msg))
	case StashListMsg, StashPatchMsg:
		cmds = append(cmds, r.updateTabComponent(&Stash{}, msg))
	// We have two spinners, one is used to when loading the repository and the
	// other is used when loading the log.
	// Check if the spinner ID matches the spinner model.
	case spinner.TickMsg:
		if r.state == loadingState && r.spinner.ID() == msg.ID {
			s, cmd := r.spinner.Update(msg)
			r.spinner = s
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		} else {
			for i, c := range r.panes {
				if c.SpinnerID() == msg.ID {
					m, cmd := c.Update(msg)
					r.panes[i] = m.(common.TabComponent)
					if cmd != nil {
						cmds = append(cmds, cmd)
					}
					break
				}
			}
		}
	case tea.WindowSizeMsg:
		r.SetSize(msg.Width, msg.Height)
		cmds = append(cmds, r.updateModels(msg))
	case EmptyRepoMsg:
		r.ref = nil
		r.state = readyState
		cmds = append(cmds, r.updateModels(msg))
	case common.ErrorMsg:
		r.state = readyState
	case SwitchTabMsg:
		for i, c := range r.panes {
			if c.TabName() == msg.TabName() {
				cmds = append(cmds, tabs.SelectTabCmd(i))
				break
			}
		}
	}
	active := r.panes[r.activeTab]
	m, cmd := active.Update(msg)
	r.panes[r.activeTab] = m.(common.TabComponent)
	if cmd != nil {
		cmds = append(cmds, cmd)
	}

	// Update the status bar on these events
	// Must come after we've updated the active tab
	switch msg.(type) {
	case RepoMsg, RefMsg, tabs.ActiveTabMsg, tea.KeyPressMsg,
		tea.MouseClickMsg, tea.MouseWheelMsg, FileItemsMsg, FileContentMsg,
		FileBlameMsg, selector.ActiveMsg, LogItemsMsg, GoBackMsg, LogDiffMsg,
		EmptyRepoMsg, StashListMsg, StashPatchMsg:
		r.setStatusBarInfo()
	}

	s, cmd := r.statusbar.Update(msg)
	r.statusbar = s.(*statusbar.Model)
	if cmd != nil {
		cmds = append(cmds, cmd)
	}

	return r, tea.Batch(cmds...)
}

// View implements tea.Model.
func (r *Repo) View() string {
	wm, hm := r.getMargins()
	hm += r.common.Styles.Tabs.GetHeight() +
		r.common.Styles.Tabs.GetVerticalFrameSize()
	s := r.common.Styles.Repo.Base.
		Width(r.common.Width - wm).
		Height(r.common.Height - hm)
	mainStyle := r.common.Styles.Repo.Body.
		Height(r.common.Height - hm)
	var main string
	var statusbar string
	switch r.state {
	case loadingState:
		main = fmt.Sprintf("%s loading…", r.spinner.View())
	case readyState:
		main = r.panes[r.activeTab].View()
		statusbar = r.statusbar.View()
	}
	main = r.common.Zone.Mark(
		"repo-main",
		mainStyle.Render(main),
	)
	view := lipgloss.JoinVertical(lipgloss.Left,
		r.headerView(),
		r.tabs.View(),
		main,
		statusbar,
	)
	return s.Render(view)
}

func (r *Repo) headerView() string {
	if r.selectedRepo == nil {
		return ""
	}
	truncate := lipgloss.NewStyle().MaxWidth(r.common.Width)
	header := r.selectedRepo.ProjectName()
	if header == "" {
		header = r.selectedRepo.Name()
	}
	header = r.common.Styles.Repo.HeaderName.Render(header)
	desc := strings.TrimSpace(r.selectedRepo.Description())
	if desc != "" {
		header = lipgloss.JoinVertical(lipgloss.Left,
			header,
			r.common.Styles.Repo.HeaderDesc.Render(desc),
		)
	}
	urlStyle := r.common.Styles.URLStyle.
		Width(r.common.Width - lipgloss.Width(header) - 1).
		Align(lipgloss.Right)
	var url string
	if cfg := r.common.Config(); cfg != nil {
		url = r.common.CloneCmd(cfg.SSH.PublicURL, r.selectedRepo.Name())
	}
	url = common.TruncateString(url, r.common.Width-lipgloss.Width(header)-1)
	url = r.common.Zone.Mark(
		fmt.Sprintf("%s-url", r.selectedRepo.Name()),
		urlStyle.Render(url),
	)

	header = lipgloss.JoinHorizontal(lipgloss.Top, header, url)

	style := r.common.Styles.Repo.Header.Width(r.common.Width)
	return style.Render(
		truncate.Render(header),
	)
}

func (r *Repo) setStatusBarInfo() {
	if r.selectedRepo == nil {
		return
	}

	active := r.panes[r.activeTab]
	key := r.selectedRepo.Name()
	value := active.StatusBarValue()
	info := active.StatusBarInfo()
	extra := "*"
	if r.ref != nil {
		extra += " " + r.ref.Name().Short()
	}

	r.statusbar.SetStatus(key, value, info, extra)
}

func (r *Repo) updateTabComponent(c common.TabComponent, msg tea.Msg) tea.Cmd {
	cmds := make([]tea.Cmd, 0)
	for i, b := range r.panes {
		if b.TabName() == c.TabName() {
			m, cmd := b.Update(msg)
			r.panes[i] = m.(common.TabComponent)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
			break
		}
	}
	return tea.Batch(cmds...)
}

func (r *Repo) updateModels(msg tea.Msg) tea.Cmd {
	cmds := make([]tea.Cmd, 0)
	for i, b := range r.panes {
		m, cmd := b.Update(msg)
		r.panes[i] = m.(common.TabComponent)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}
	return tea.Batch(cmds...)
}

func copyCmd(text, msg string) tea.Cmd {
	return func() tea.Msg {
		return CopyMsg{
			Text:    text,
			Message: msg,
		}
	}
}

func goBackCmd() tea.Msg {
	return GoBackMsg{}
}

func switchTabCmd(m common.TabComponent) tea.Cmd {
	return func() tea.Msg {
		return SwitchTabMsg(m)
	}
}

func renderLoading(c common.Common, s spinner.Model) string {
	msg := fmt.Sprintf("%s loading…", s.View())
	return c.Styles.SpinnerContainer.
		Height(c.Height).
		Render(msg)
}
