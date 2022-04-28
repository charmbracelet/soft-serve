package repo

import (
	"fmt"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	ggit "github.com/charmbracelet/soft-serve/git"
	"github.com/charmbracelet/soft-serve/ui/common"
	"github.com/charmbracelet/soft-serve/ui/components/code"
	"github.com/charmbracelet/soft-serve/ui/components/selector"
	"github.com/charmbracelet/soft-serve/ui/components/statusbar"
	"github.com/charmbracelet/soft-serve/ui/components/tabs"
	"github.com/charmbracelet/soft-serve/ui/git"
	"github.com/charmbracelet/soft-serve/ui/pages/selection"
)

type tab int

const (
	readmeTab tab = iota
	filesTab
	commitsTab
	branchesTab
	tagsTab
)

// UpdateStatusBarMsg updates the status bar.
type UpdateStatusBarMsg struct{}

// RepoMsg is a message that contains a git.Repository.
type RepoMsg git.GitRepo

// RefMsg is a message that contains a git.Reference.
type RefMsg *ggit.Reference

// Repo is a view for a git repository.
type Repo struct {
	common       common.Common
	rs           git.GitRepoSource
	selectedRepo git.GitRepo
	selectedItem selection.Item
	activeTab    tab
	tabs         *tabs.Tabs
	statusbar    *statusbar.StatusBar
	boxes        []common.Component
	ref          *ggit.Reference
}

// New returns a new Repo.
func New(c common.Common, rs git.GitRepoSource) *Repo {
	sb := statusbar.New(c)
	tb := tabs.New(c, []string{"Readme", "Files", "Commits", "Branches", "Tags"})
	readme := code.New(c, "", "")
	readme.NoContentStyle = readme.NoContentStyle.SetString("No readme found.")
	log := NewLog(c)
	files := NewFiles(c)
	branches := NewRefs(c, ggit.RefsHeads)
	tags := NewRefs(c, ggit.RefsTags)
	boxes := []common.Component{
		readme,
		files,
		log,
		branches,
		tags,
	}
	r := &Repo{
		common:    c,
		rs:        rs,
		tabs:      tb,
		statusbar: sb,
		boxes:     boxes,
	}
	return r
}

// SetSize implements common.Component.
func (r *Repo) SetSize(width, height int) {
	r.common.SetSize(width, height)
	hm := r.common.Styles.RepoBody.GetVerticalFrameSize() +
		r.common.Styles.RepoHeader.GetHeight() +
		r.common.Styles.RepoHeader.GetVerticalFrameSize() +
		r.common.Styles.StatusBar.GetHeight() +
		r.common.Styles.Tabs.GetHeight() +
		r.common.Styles.Tabs.GetVerticalFrameSize()
	r.tabs.SetSize(width, height-hm)
	r.statusbar.SetSize(width, height-hm)
	for _, b := range r.boxes {
		b.SetSize(width, height-hm)
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
	switch r.activeTab {
	case readmeTab:
		b = append(b, r.common.KeyMap.UpDown)
	default:
		b = append(b, r.boxes[commitsTab].(help.KeyMap).ShortHelp()...)
	}
	return b
}

// FullHelp implements help.KeyMap.
func (r *Repo) FullHelp() [][]key.Binding {
	b := make([][]key.Binding, 0)
	b = append(b, r.commonHelp())
	switch r.activeTab {
	case readmeTab:
		k := r.boxes[readmeTab].(*code.Code).KeyMap
		b = append(b, [][]key.Binding{
			{
				k.PageDown,
				k.PageUp,
			},
			{
				k.HalfPageDown,
				k.HalfPageUp,
			},
			{
				k.Down,
				k.Up,
			},
		}...)
	default:
		b = append(b, r.boxes[r.activeTab].(help.KeyMap).FullHelp()...)
	}
	return b
}

// Init implements tea.View.
func (r *Repo) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model.
func (r *Repo) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmds := make([]tea.Cmd, 0)
	switch msg := msg.(type) {
	case selector.SelectMsg:
		switch msg.IdentifiableItem.(type) {
		case selection.Item:
			r.selectedItem = msg.IdentifiableItem.(selection.Item)
		}
	case RepoMsg:
		r.activeTab = 0
		r.selectedRepo = git.GitRepo(msg)
		r.boxes[readmeTab].(*code.Code).GotoTop()
		cmds = append(cmds,
			r.tabs.Init(),
			r.updateReadmeCmd,
			r.updateRefCmd,
			r.updateModels(msg),
		)
	case RefMsg:
		r.ref = msg
		for _, b := range r.boxes {
			cmds = append(cmds, b.Init())
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
			cmds = append(cmds, r.updateStatusBarCmd)
		}
	case tea.KeyMsg, tea.MouseMsg:
		t, cmd := r.tabs.Update(msg)
		r.tabs = t.(*tabs.Tabs)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
		if r.selectedRepo != nil {
			cmds = append(cmds, r.updateStatusBarCmd)
		}
	case FileItemsMsg:
		f, cmd := r.boxes[filesTab].Update(msg)
		r.boxes[filesTab] = f.(*Files)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	case LogCountMsg, LogItemsMsg:
		l, cmd := r.boxes[commitsTab].Update(msg)
		r.boxes[commitsTab] = l.(*Log)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	case RefItemsMsg:
		switch msg.prefix {
		case ggit.RefsHeads:
			b, cmd := r.boxes[branchesTab].Update(msg)
			r.boxes[branchesTab] = b.(*Refs)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		case ggit.RefsTags:
			t, cmd := r.boxes[tagsTab].Update(msg)
			r.boxes[tagsTab] = t.(*Refs)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
	case UpdateStatusBarMsg:
		cmds = append(cmds, r.updateStatusBarCmd)
	case tea.WindowSizeMsg:
		b, cmd := r.boxes[readmeTab].Update(msg)
		r.boxes[readmeTab] = b.(*code.Code)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
		cmds = append(cmds, r.updateModels(msg))
	}
	s, cmd := r.statusbar.Update(msg)
	r.statusbar = s.(*statusbar.StatusBar)
	if cmd != nil {
		cmds = append(cmds, cmd)
	}
	m, cmd := r.boxes[r.activeTab].Update(msg)
	r.boxes[r.activeTab] = m.(common.Component)
	if cmd != nil {
		cmds = append(cmds, cmd)
	}
	return r, tea.Batch(cmds...)
}

// View implements tea.Model.
func (r *Repo) View() string {
	s := r.common.Styles.Repo.Copy().
		Width(r.common.Width).
		Height(r.common.Height)
	repoBodyStyle := r.common.Styles.RepoBody.Copy()
	hm := repoBodyStyle.GetVerticalFrameSize() +
		r.common.Styles.RepoHeader.GetHeight() +
		r.common.Styles.RepoHeader.GetVerticalFrameSize() +
		r.common.Styles.StatusBar.GetHeight() +
		r.common.Styles.Tabs.GetHeight() +
		r.common.Styles.Tabs.GetVerticalFrameSize()
	mainStyle := repoBodyStyle.
		Height(r.common.Height - hm)
	main := r.boxes[r.activeTab].View()
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
	name := r.common.Styles.RepoHeaderName.Render(r.selectedItem.Title())
	// TODO move this into a style.
	url := lipgloss.NewStyle().
		MarginLeft(1).
		Width(r.common.Width - lipgloss.Width(name) - 1).
		Align(lipgloss.Right).
		Render(r.selectedItem.URL())
	desc := r.common.Styles.RepoHeaderDesc.Render(r.selectedItem.Description())
	style := r.common.Styles.RepoHeader.Copy().Width(r.common.Width)
	return style.Render(
		lipgloss.JoinVertical(lipgloss.Top,
			lipgloss.JoinHorizontal(lipgloss.Left,
				name,
				url,
			),
			desc,
		),
	)
}

func (r *Repo) setRepoCmd(repo string) tea.Cmd {
	return func() tea.Msg {
		for _, r := range r.rs.AllRepos() {
			if r.Name() == repo {
				return RepoMsg(r)
			}
		}
		return common.ErrorMsg(git.ErrMissingRepo)
	}
}

func (r *Repo) updateStatusBarCmd() tea.Msg {
	var info, value string
	switch r.activeTab {
	case readmeTab:
		info = fmt.Sprintf("%.f%%", r.boxes[readmeTab].(*code.Code).ScrollPercent()*100)
	default:
		value = r.boxes[r.activeTab].(statusbar.Model).StatusBarValue()
		info = r.boxes[r.activeTab].(statusbar.Model).StatusBarInfo()
	}
	return statusbar.StatusBarMsg{
		Key:    r.selectedRepo.Name(),
		Value:  value,
		Info:   info,
		Branch: fmt.Sprintf(" %s", r.ref.Name().Short()),
	}
}

func (r *Repo) updateReadmeCmd() tea.Msg {
	if r.selectedRepo == nil {
		return common.ErrorCmd(git.ErrMissingRepo)
	}
	rm, rp := r.selectedRepo.Readme()
	return r.boxes[readmeTab].(*code.Code).SetContent(rm, rp)
}

func (r *Repo) updateRefCmd() tea.Msg {
	head, err := r.selectedRepo.HEAD()
	if err != nil {
		return common.ErrorMsg(err)
	}
	return RefMsg(head)
}

func (r *Repo) updateModels(msg tea.Msg) tea.Cmd {
	cmds := make([]tea.Cmd, 0)
	for i, b := range r.boxes {
		m, cmd := b.Update(msg)
		r.boxes[i] = m.(common.Component)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}
	return tea.Batch(cmds...)
}

func updateStatusBarCmd() tea.Msg {
	return UpdateStatusBarMsg{}
}
