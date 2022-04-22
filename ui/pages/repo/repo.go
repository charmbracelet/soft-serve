package repo

import (
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
)

type tab int

const (
	readmeTab tab = iota
	filesTab
	commitsTab
	branchesTab
	tagsTab
)

type RepoMsg git.GitRepo

type Repo struct {
	common       common.Common
	rs           git.GitRepoSource
	selectedRepo git.GitRepo
	activeTab    tab
	tabs         *tabs.Tabs
	statusbar    *statusbar.StatusBar
	readme       *code.Code
	log          *Log
	ref          *ggit.Reference
}

func New(common common.Common, rs git.GitRepoSource) *Repo {
	sb := statusbar.New(common)
	tb := tabs.New(common, []string{"Readme", "Files", "Commits", "Branches", "Tags"})
	readme := code.New(common, "", "")
	readme.NoContentStyle = readme.NoContentStyle.SetString("No readme found.")
	r := &Repo{
		common:    common,
		rs:        rs,
		tabs:      tb,
		statusbar: sb,
		readme:    readme,
	}
	return r
}

func (r *Repo) SetSize(width, height int) {
	r.common.SetSize(width, height)
	hm := 4
	r.tabs.SetSize(width, height-hm)
	r.statusbar.SetSize(width, height-hm)
	r.readme.SetSize(width, height-hm)
	if r.log != nil {
		r.log.SetSize(width, height-hm)
	}
}

func (r *Repo) ShortHelp() []key.Binding {
	b := make([]key.Binding, 0)
	tab := r.common.Keymap.Section
	tab.SetHelp("tab", "switch tab")
	b = append(b, r.common.Keymap.Back)
	b = append(b, tab)
	return b
}

func (r *Repo) FullHelp() [][]key.Binding {
	b := make([][]key.Binding, 0)
	return b
}

func (r *Repo) Init() tea.Cmd {
	return nil
}

func (r *Repo) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmds := make([]tea.Cmd, 0)
	switch msg := msg.(type) {
	case selector.SelectMsg:
		r.activeTab = 0
		cmds = append(cmds, r.tabs.Init(), r.setRepoCmd(string(msg)))
	case RepoMsg:
		r.selectedRepo = git.GitRepo(msg)
		cmds = append(cmds, r.updateStatusBarCmd, r.updateReadmeCmd)
	case tabs.ActiveTabMsg:
		r.activeTab = tab(msg)
	}
	t, cmd := r.tabs.Update(msg)
	r.tabs = t.(*tabs.Tabs)
	if cmd != nil {
		cmds = append(cmds, cmd)
	}
	s, cmd := r.statusbar.Update(msg)
	r.statusbar = s.(*statusbar.StatusBar)
	if cmd != nil {
		cmds = append(cmds, cmd)
	}
	switch r.activeTab {
	case readmeTab:
		b, cmd := r.readme.Update(msg)
		r.readme = b.(*code.Code)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	case filesTab:
	case commitsTab:
		if r.log == nil {
			r.log = NewLog(r.common)
			cmds = append(cmds, r.log.Init())
		}
		l, cmd := r.log.Update(msg)
		r.log = l.(*Log)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	case branchesTab:
	case tagsTab:
	}
	return r, tea.Batch(cmds...)
}

func (r *Repo) View() string {
	s := r.common.Styles.RepoBody.Copy().
		Width(r.common.Width).
		Height(r.common.Height)
	mainStyle := lipgloss.NewStyle().
		Height(r.common.Height-4).
		Margin(1, 0)
	main := mainStyle.Render("")
	switch r.activeTab {
	case readmeTab:
		main = mainStyle.Render(r.readme.View())
	case filesTab:
	case commitsTab:
		if r.log != nil {
			main = mainStyle.Render(r.log.View())
		}
	}
	view := lipgloss.JoinVertical(lipgloss.Top,
		r.tabs.View(),
		main,
		r.statusbar.View(),
	)
	return s.Render(view)
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
	branch, err := r.selectedRepo.HEAD()
	if err != nil {
		return common.ErrorMsg(err)
	}
	return statusbar.StatusBarMsg{
		Key:    r.selectedRepo.Name(),
		Value:  "",
		Info:   "",
		Branch: branch.Name().Short(),
	}
}

func (r *Repo) updateReadmeCmd() tea.Msg {
	if r.selectedRepo == nil {
		return common.ErrorCmd(git.ErrMissingRepo)
	}
	rm, rp := r.selectedRepo.Readme()
	return r.readme.SetContent(rm, rp)
}
