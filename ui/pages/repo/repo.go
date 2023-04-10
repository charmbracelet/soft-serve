package repo

import (
	"fmt"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/log"
	"github.com/charmbracelet/soft-serve/git"
	"github.com/charmbracelet/soft-serve/server/backend"
	"github.com/charmbracelet/soft-serve/ui/common"
	"github.com/charmbracelet/soft-serve/ui/components/footer"
	"github.com/charmbracelet/soft-serve/ui/components/statusbar"
	"github.com/charmbracelet/soft-serve/ui/components/tabs"
)

var (
	logger = log.WithPrefix("ui.repo")
)

type state int

const (
	loadingState state = iota
	readyState
)

type tab int

const (
	readmeTab tab = iota
	filesTab
	commitsTab
	branchesTab
	tagsTab
	lastTab
)

func (t tab) String() string {
	return []string{
		"Readme",
		"Files",
		"Commits",
		"Branches",
		"Tags",
	}[t]
}

// EmptyRepoMsg is a message to indicate that the repository is empty.
type EmptyRepoMsg struct{}

// CopyURLMsg is a message to copy the URL of the current repository.
type CopyURLMsg struct{}

// UpdateStatusBarMsg updates the status bar.
type UpdateStatusBarMsg struct{}

// RepoMsg is a message that contains a git.Repository.
type RepoMsg backend.Repository

// BackMsg is a message to go back to the previous view.
type BackMsg struct{}

// CopyMsg is a message to indicate copied text.
type CopyMsg struct {
	Text    string
	Message string
}

// Repo is a view for a git repository.
type Repo struct {
	common       common.Common
	selectedRepo backend.Repository
	activeTab    tab
	tabs         *tabs.Tabs
	statusbar    *statusbar.StatusBar
	panes        []common.Component
	ref          *git.Reference
	state        state
	spinner      spinner.Model
	panesReady   [lastTab]bool
}

// New returns a new Repo.
func New(c common.Common) *Repo {
	sb := statusbar.New(c)
	ts := make([]string, lastTab)
	// Tabs must match the order of tab constants above.
	for i, t := range []tab{readmeTab, filesTab, commitsTab, branchesTab, tagsTab} {
		ts[i] = t.String()
	}
	tb := tabs.New(c, ts)
	readme := NewReadme(c)
	log := NewLog(c)
	files := NewFiles(c)
	branches := NewRefs(c, git.RefsHeads)
	tags := NewRefs(c, git.RefsTags)
	// Make sure the order matches the order of tab constants above.
	panes := []common.Component{
		readme,
		files,
		log,
		branches,
		tags,
	}
	s := spinner.New(spinner.WithSpinner(spinner.Dot),
		spinner.WithStyle(c.Styles.Spinner))
	r := &Repo{
		common:    c,
		tabs:      tb,
		statusbar: sb,
		panes:     panes,
		state:     loadingState,
		spinner:   s,
	}
	return r
}

// SetSize implements common.Component.
func (r *Repo) SetSize(width, height int) {
	r.common.SetSize(width, height)
	hm := r.common.Styles.Repo.Body.GetVerticalFrameSize() +
		r.common.Styles.Repo.Header.GetHeight() +
		r.common.Styles.Repo.Header.GetVerticalFrameSize() +
		r.common.Styles.StatusBar.GetHeight()
	r.tabs.SetSize(width, height-hm)
	r.statusbar.SetSize(width, height-hm)
	for _, p := range r.panes {
		p.SetSize(width, height-hm)
	}
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
	return tea.Batch(
		r.tabs.Init(),
		r.statusbar.Init(),
	)
}

// Update implements tea.Model.
func (r *Repo) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmds := make([]tea.Cmd, 0)
	switch msg := msg.(type) {
	case RepoMsg:
		// Set the state to loading when we get a new repository.
		r.state = loadingState
		r.panesReady = [lastTab]bool{}
		r.activeTab = 0
		r.selectedRepo = msg
		cmds = append(cmds,
			r.tabs.Init(),
			// This will set the selected repo in each pane's model.
			r.updateModels(msg),
			r.spinner.Tick,
		)
	case RefMsg:
		r.ref = msg
		for _, p := range r.panes {
			// Init will initiate each pane's model with its contents.
			cmds = append(cmds, p.Init())
		}
		cmds = append(cmds,
			r.updateStatusBarCmd,
			r.updateModels(msg),
		)
	case tabs.SelectTabMsg:
		r.activeTab = tab(msg)
		t, cmd := r.tabs.Update(msg)
		r.tabs = t.(*tabs.Tabs)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	case tabs.ActiveTabMsg:
		r.activeTab = tab(msg)
		if r.selectedRepo != nil {
			cmds = append(cmds,
				r.updateStatusBarCmd,
			)
		}
	case tea.KeyMsg, tea.MouseMsg:
		t, cmd := r.tabs.Update(msg)
		r.tabs = t.(*tabs.Tabs)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
		if r.selectedRepo != nil {
			cmds = append(cmds, r.updateStatusBarCmd)
			urlID := fmt.Sprintf("%s-url", r.selectedRepo.Name())
			cmd := common.CloneCmd(r.common.Config().SSH.PublicURL, r.selectedRepo.Name())
			if msg, ok := msg.(tea.MouseMsg); ok && r.common.Zone.Get(urlID).InBounds(msg) {
				cmds = append(cmds, copyCmd(cmd, "Command copied to clipboard"))
			}
		}
		switch msg := msg.(type) {
		case tea.MouseMsg:
			switch msg.Type {
			case tea.MouseLeft:
				switch {
				case r.common.Zone.Get("repo-help").InBounds(msg):
					cmds = append(cmds, footer.ToggleFooterCmd)
				}
			case tea.MouseRight:
				switch {
				case r.common.Zone.Get("repo-main").InBounds(msg):
					cmds = append(cmds, backCmd)
				}
			}
		}
	case CopyMsg:
		txt := msg.Text
		if cfg := r.common.Config(); cfg != nil {
			r.common.Copy.Copy(txt)
		}
		cmds = append(cmds, func() tea.Msg {
			return statusbar.StatusBarMsg{
				Value: msg.Message,
			}
		})
	case ReadmeMsg, FileItemsMsg, LogCountMsg, LogItemsMsg, RefItemsMsg:
		cmds = append(cmds, r.updateRepo(msg))
	// We have two spinners, one is used to when loading the repository and the
	// other is used when loading the log.
	// Check if the spinner ID matches the spinner model.
	case spinner.TickMsg:
		switch msg.ID {
		case r.spinner.ID():
			if r.state == loadingState {
				s, cmd := r.spinner.Update(msg)
				r.spinner = s
				if cmd != nil {
					cmds = append(cmds, cmd)
				}
			}
		default:
			cmds = append(cmds, r.updateRepo(msg))
		}
	case UpdateStatusBarMsg:
		cmds = append(cmds, r.updateStatusBarCmd)
	case tea.WindowSizeMsg:
		cmds = append(cmds, r.updateModels(msg))
	case EmptyRepoMsg:
		r.ref = nil
		r.state = readyState
		cmds = append(cmds,
			r.updateModels(msg),
			r.updateStatusBarCmd,
		)
	case common.ErrorMsg:
		r.state = readyState
	}
	s, cmd := r.statusbar.Update(msg)
	r.statusbar = s.(*statusbar.StatusBar)
	if cmd != nil {
		cmds = append(cmds, cmd)
	}
	m, cmd := r.panes[r.activeTab].Update(msg)
	r.panes[r.activeTab] = m.(common.Component)
	if cmd != nil {
		cmds = append(cmds, cmd)
	}
	return r, tea.Batch(cmds...)
}

// View implements tea.Model.
func (r *Repo) View() string {
	s := r.common.Styles.Repo.Base.Copy().
		Width(r.common.Width).
		Height(r.common.Height)
	repoBodyStyle := r.common.Styles.Repo.Body.Copy()
	hm := repoBodyStyle.GetVerticalFrameSize() +
		r.common.Styles.Repo.Header.GetHeight() +
		r.common.Styles.Repo.Header.GetVerticalFrameSize() +
		r.common.Styles.StatusBar.GetHeight() +
		r.common.Styles.Tabs.GetHeight() +
		r.common.Styles.Tabs.GetVerticalFrameSize()
	mainStyle := repoBodyStyle.
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
	view := lipgloss.JoinVertical(lipgloss.Top,
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
	name := r.selectedRepo.ProjectName()
	if name == "" {
		name = r.selectedRepo.Name()
	}
	name = r.common.Styles.Repo.HeaderName.Render(name)
	desc := r.selectedRepo.Description()
	if desc == "" {
		desc = name
		name = ""
	} else {
		desc = r.common.Styles.Repo.HeaderDesc.Render(desc)
	}
	urlStyle := r.common.Styles.URLStyle.Copy().
		Width(r.common.Width - lipgloss.Width(desc) - 1).
		Align(lipgloss.Right)
	var url string
	if cfg := r.common.Config(); cfg != nil {
		url = common.CloneCmd(cfg.SSH.PublicURL, r.selectedRepo.Name())
	}
	url = common.TruncateString(url, r.common.Width-lipgloss.Width(desc)-1)
	url = r.common.Zone.Mark(
		fmt.Sprintf("%s-url", r.selectedRepo.Name()),
		urlStyle.Render(url),
	)
	style := r.common.Styles.Repo.Header.Copy().Width(r.common.Width)
	return style.Render(
		lipgloss.JoinVertical(lipgloss.Top,
			truncate.Render(name),
			truncate.Render(lipgloss.JoinHorizontal(lipgloss.Left,
				desc,
				url,
			)),
		),
	)
}

func (r *Repo) updateStatusBarCmd() tea.Msg {
	if r.selectedRepo == nil {
		return nil
	}
	value := r.panes[r.activeTab].(statusbar.Model).StatusBarValue()
	info := r.panes[r.activeTab].(statusbar.Model).StatusBarInfo()
	branch := "*"
	if r.ref != nil {
		branch += " " + r.ref.Name().Short()
	}
	return statusbar.StatusBarMsg{
		Key:   r.selectedRepo.Name(),
		Value: value,
		Info:  info,
		Extra: branch,
	}
}

func (r *Repo) updateModels(msg tea.Msg) tea.Cmd {
	cmds := make([]tea.Cmd, 0)
	for i, b := range r.panes {
		m, cmd := b.Update(msg)
		r.panes[i] = m.(common.Component)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}
	return tea.Batch(cmds...)
}

func (r *Repo) updateRepo(msg tea.Msg) tea.Cmd {
	cmds := make([]tea.Cmd, 0)
	switch msg := msg.(type) {
	case LogCountMsg, LogItemsMsg, spinner.TickMsg:
		switch msg.(type) {
		case LogItemsMsg:
			r.panesReady[commitsTab] = true
		}
		l, cmd := r.panes[commitsTab].Update(msg)
		r.panes[commitsTab] = l.(*Log)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	case FileItemsMsg:
		r.panesReady[filesTab] = true
		f, cmd := r.panes[filesTab].Update(msg)
		r.panes[filesTab] = f.(*Files)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	case RefItemsMsg:
		switch msg.prefix {
		case git.RefsHeads:
			r.panesReady[branchesTab] = true
			b, cmd := r.panes[branchesTab].Update(msg)
			r.panes[branchesTab] = b.(*Refs)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		case git.RefsTags:
			r.panesReady[tagsTab] = true
			t, cmd := r.panes[tagsTab].Update(msg)
			r.panes[tagsTab] = t.(*Refs)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
	case ReadmeMsg:
		r.panesReady[readmeTab] = true
	}
	if r.isReady() {
		r.state = readyState
	}
	return tea.Batch(cmds...)
}

func (r *Repo) isReady() bool {
	ready := true
	// We purposely ignore the log pane here because it has its own spinner.
	for _, b := range []bool{
		r.panesReady[filesTab], r.panesReady[branchesTab],
		r.panesReady[tagsTab], r.panesReady[readmeTab],
	} {
		if !b {
			ready = false
			break
		}
	}
	return ready
}

func copyCmd(text, msg string) tea.Cmd {
	return func() tea.Msg {
		return CopyMsg{
			Text:    text,
			Message: msg,
		}
	}
}

func updateStatusBarCmd() tea.Msg {
	return UpdateStatusBarMsg{}
}

func backCmd() tea.Msg {
	return BackMsg{}
}
