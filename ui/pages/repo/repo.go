package repo

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/soft-serve/config"
	ggit "github.com/charmbracelet/soft-serve/git"
	"github.com/charmbracelet/soft-serve/ui/common"
	"github.com/charmbracelet/soft-serve/ui/components/statusbar"
	"github.com/charmbracelet/soft-serve/ui/components/tabs"
	"github.com/charmbracelet/soft-serve/ui/git"
)

type state int

const (
	loadingState state = iota
	loadedState
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

// CopyURLMsg is a message to copy the URL of the current repository.
type CopyURLMsg struct{}

// ResetURLMsg is a message to reset the URL string.
type ResetURLMsg struct{}

// UpdateStatusBarMsg updates the status bar.
type UpdateStatusBarMsg struct{}

// RepoMsg is a message that contains a git.Repository.
type RepoMsg git.GitRepo

// RefMsg is a message that contains a git.Reference.
type RefMsg *ggit.Reference

// BackMsg is a message to go back to the previous view.
type BackMsg struct{}

// Repo is a view for a git repository.
type Repo struct {
	common       common.Common
	cfg          *config.Config
	selectedRepo git.GitRepo
	activeTab    tab
	tabs         *tabs.Tabs
	statusbar    *statusbar.StatusBar
	panes        []common.Component
	ref          *ggit.Reference
	copyURL      time.Time
}

// New returns a new Repo.
func New(cfg *config.Config, c common.Common) *Repo {
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
	branches := NewRefs(c, ggit.RefsHeads)
	tags := NewRefs(c, ggit.RefsTags)
	// Make sure the order matches the order of tab constants above.
	panes := []common.Component{
		readme,
		files,
		log,
		branches,
		tags,
	}
	r := &Repo{
		cfg:       cfg,
		common:    c,
		tabs:      tb,
		statusbar: sb,
		panes:     panes,
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
		r.activeTab = 0
		r.selectedRepo = git.GitRepo(msg)
		cmds = append(cmds,
			r.tabs.Init(),
			r.updateRefCmd,
			r.updateModels(msg),
		)
	case RefMsg:
		r.ref = msg
		for _, p := range r.panes {
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
			switch msg := msg.(type) {
			case tea.MouseMsg:
				switch msg.Type {
				case tea.MouseLeft:
					id := fmt.Sprintf("%s-url", r.selectedRepo.Repo())
					if r.common.Zone.Get(id).InBounds(msg) {
						cmds = append(cmds, r.copyUrlCmd())
					}
				case tea.MouseRight:
					if r.common.Zone.Get("repo-main").InBounds(msg) {
						cmds = append(cmds, backCmd)
					}
				}
			}
		}
	case CopyURLMsg:
		r.common.Copy.Copy(
			git.RepoURL(r.cfg.Host, r.cfg.Port, r.selectedRepo.Repo()),
		)
	case ResetURLMsg:
		r.copyURL = time.Time{}
	case ReadmeMsg:
	case FileItemsMsg:
		f, cmd := r.panes[filesTab].Update(msg)
		r.panes[filesTab] = f.(*Files)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	// The Log bubble is the only bubble that uses a spinner, so this is fine
	// for now. We need to pass the TickMsg to the Log bubble when the Log is
	// loading but not the current selected tab so that the spinner works.
	case LogCountMsg, LogItemsMsg, spinner.TickMsg:
		l, cmd := r.panes[commitsTab].Update(msg)
		r.panes[commitsTab] = l.(*Log)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	case RefItemsMsg:
		switch msg.prefix {
		case ggit.RefsHeads:
			b, cmd := r.panes[branchesTab].Update(msg)
			r.panes[branchesTab] = b.(*Refs)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		case ggit.RefsTags:
			t, cmd := r.panes[tagsTab].Update(msg)
			r.panes[tagsTab] = t.(*Refs)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
	case UpdateStatusBarMsg:
		cmds = append(cmds, r.updateStatusBarCmd)
	case tea.WindowSizeMsg:
		cmds = append(cmds, r.updateModels(msg))
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
	main := r.common.Zone.Mark(
		"repo-main",
		r.panes[r.activeTab].View(),
	)
	view := lipgloss.JoinVertical(lipgloss.Top,
		r.headerView(),
		r.tabs.View(),
		mainStyle.Render(main),
		r.statusbar.View(),
	)
	return s.Render(view)
}

func (r *Repo) headerView() string {
	if r.selectedRepo == nil {
		return ""
	}
	cfg := r.cfg
	truncate := lipgloss.NewStyle().MaxWidth(r.common.Width)
	name := r.common.Styles.Repo.HeaderName.Render(r.selectedRepo.Name())
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
	url := git.RepoURL(cfg.Host, cfg.Port, r.selectedRepo.Repo())
	if !r.copyURL.IsZero() && r.copyURL.Add(time.Second).After(time.Now()) {
		url = "copied!"
	}
	url = common.TruncateString(url, r.common.Width-lipgloss.Width(desc)-1)
	url = r.common.Zone.Mark(
		fmt.Sprintf("%s-url", r.selectedRepo.Repo()),
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
	ref := ""
	if r.ref != nil {
		ref = r.ref.Name().Short()
	}
	return statusbar.StatusBarMsg{
		Key:    r.selectedRepo.Repo(),
		Value:  value,
		Info:   info,
		Branch: fmt.Sprintf("* %s", ref),
	}
}

func (r *Repo) updateRefCmd() tea.Msg {
	if r.selectedRepo == nil {
		return nil
	}
	head, err := r.selectedRepo.HEAD()
	if err != nil {
		return common.ErrorMsg(err)
	}
	return RefMsg(head)
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

func (r *Repo) copyUrlCmd() tea.Cmd {
	r.copyURL = time.Now()
	return tea.Batch(
		func() tea.Msg {
			return CopyURLMsg{}
		},
		tea.Tick(time.Second, func(time.Time) tea.Msg {
			return ResetURLMsg{}
		}),
	)
}

func updateStatusBarCmd() tea.Msg {
	return UpdateStatusBarMsg{}
}

func backCmd() tea.Msg {
	return BackMsg{}
}
