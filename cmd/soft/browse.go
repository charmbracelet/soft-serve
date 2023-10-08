package main

import (
	"path/filepath"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/soft-serve/git"
	"github.com/charmbracelet/soft-serve/server/proto"
	"github.com/charmbracelet/soft-serve/server/ui/common"
	"github.com/charmbracelet/soft-serve/server/ui/components/footer"
	"github.com/charmbracelet/soft-serve/server/ui/pages/repo"
	"github.com/muesli/termenv"
	"github.com/spf13/cobra"
)

var browseCmd = &cobra.Command{
	Use:   "browse PATH",
	Short: "Browse a repository",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		rp := "."
		if len(args) > 0 {
			rp = args[0]
		}

		abs, err := filepath.Abs(rp)
		if err != nil {
			return err
		}

		// Bubble Tea uses Termenv default output so we have to use the same
		// thing here.
		output := termenv.DefaultOutput()
		ctx := cmd.Context()
		c := common.NewCommon(ctx, output, 0, 0)
		m := &model{
			m: repo.New(c,
				repo.NewReadme(c),
				repo.NewFiles(c),
				repo.NewLog(c),
				repo.NewRefs(c, git.RefsHeads),
				repo.NewRefs(c, git.RefsRemotes),
				repo.NewRefs(c, git.RefsTags),
			),
			repoPath: abs,
			c:        c,
		}

		m.f = footer.New(c, m)
		p := tea.NewProgram(m,
			tea.WithAltScreen(),
			tea.WithMouseCellMotion(),
		)

		_, err = p.Run()
		return err
	},
}

func init() {
	// HACK: This is a hack to hide the clone url
	// TODO: Make this configurable
	common.CloneCmd = func(publicURL, name string) string { return "" }
	rootCmd.AddCommand(browseCmd)
}

type state int

const (
	startState state = iota
	errorState
)

type model struct {
	m          *repo.Repo
	f          *footer.Footer
	repoPath   string
	c          common.Common
	state      state
	showFooter bool
	error      error
}

var _ tea.Model = &model{}

func (m *model) SetSize(w, h int) {
	m.c.SetSize(w, h)
	style := m.c.Styles.App.Copy()
	wm := style.GetHorizontalFrameSize()
	hm := style.GetVerticalFrameSize()
	if m.showFooter {
		hm += m.f.Height()
	}

	m.f.SetSize(w-wm, h-hm)
	m.m.SetSize(w-wm, h-hm)
}

// ShortHelp implements help.KeyMap.
func (m model) ShortHelp() []key.Binding {
	switch m.state {
	case errorState:
		return []key.Binding{
			m.c.KeyMap.Back,
			m.c.KeyMap.Quit,
			m.c.KeyMap.Help,
		}
	default:
		return m.m.ShortHelp()
	}
}

// FullHelp implements help.KeyMap.
func (m model) FullHelp() [][]key.Binding {
	switch m.state {
	case errorState:
		return [][]key.Binding{
			{
				m.c.KeyMap.Back,
			},
			{
				m.c.KeyMap.Quit,
				m.c.KeyMap.Help,
			},
		}
	default:
		return m.m.FullHelp()
	}
}

// Init implements tea.Model.
func (m *model) Init() tea.Cmd {
	rr, err := git.Open(m.repoPath)
	if err != nil {
		return common.ErrorCmd(err)
	}

	r := repository{rr}
	return tea.Batch(
		m.m.Init(),
		m.f.Init(),
		func() tea.Msg {
			return repo.RepoMsg(r)
		},
		repo.UpdateRefCmd(r),
	)
}

// Update implements tea.Model.
func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	m.c.Logger.Debugf("msg received: %T", msg)
	cmds := make([]tea.Cmd, 0)
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.SetSize(msg.Width, msg.Height)
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.c.KeyMap.Back) && m.error != nil:
			m.error = nil
			m.state = startState
			// Always show the footer on error.
			m.showFooter = m.f.ShowAll()
		case key.Matches(msg, m.c.KeyMap.Help):
			cmds = append(cmds, footer.ToggleFooterCmd)
		case key.Matches(msg, m.c.KeyMap.Quit):
			// Stop bubblezone background workers.
			m.c.Zone.Close()
			return m, tea.Quit
		}
	case tea.MouseMsg:
		switch msg.Type {
		case tea.MouseLeft:
			switch {
			case m.c.Zone.Get("footer").InBounds(msg):
				cmds = append(cmds, footer.ToggleFooterCmd)
			}
		}
	case footer.ToggleFooterMsg:
		m.f.SetShowAll(!m.f.ShowAll())
		m.showFooter = !m.showFooter
	case common.ErrorMsg:
		m.error = msg
		m.state = errorState
		m.showFooter = true
	}

	f, cmd := m.f.Update(msg)
	m.f = f.(*footer.Footer)
	if cmd != nil {
		cmds = append(cmds, cmd)
	}

	r, cmd := m.m.Update(msg)
	m.m = r.(*repo.Repo)
	if cmd != nil {
		cmds = append(cmds, cmd)
	}

	// This fixes determining the height margin of the footer.
	m.SetSize(m.c.Width, m.c.Height)

	return m, tea.Batch(cmds...)
}

// View implements tea.Model.
func (m *model) View() string {
	style := m.c.Styles.App.Copy()
	wm, hm := style.GetHorizontalFrameSize(), style.GetVerticalFrameSize()
	if m.showFooter {
		hm += m.f.Height()
	}

	var view string
	switch m.state {
	case startState:
		view = m.m.View()
	case errorState:
		err := m.c.Styles.ErrorTitle.Render("Bummer")
		err += m.c.Styles.ErrorBody.Render(m.error.Error())
		view = m.c.Styles.Error.Copy().
			Width(m.c.Width -
				wm -
				m.c.Styles.ErrorBody.GetHorizontalFrameSize()).
			Height(m.c.Height -
				hm -
				m.c.Styles.Error.GetVerticalFrameSize()).
			Render(err)
	}

	if m.showFooter {
		view = lipgloss.JoinVertical(lipgloss.Top, view, m.f.View())
	}

	return m.c.Zone.Scan(style.Render(view))
}

type repository struct {
	r *git.Repository
}

var _ proto.Repository = repository{}

// Description implements proto.Repository.
func (r repository) Description() string {
	return ""
}

// ID implements proto.Repository.
func (r repository) ID() int64 {
	return 0
}

// IsHidden implements proto.Repository.
func (repository) IsHidden() bool {
	return false
}

// IsMirror implements proto.Repository.
func (repository) IsMirror() bool {
	return false
}

// IsPrivate implements proto.Repository.
func (repository) IsPrivate() bool {
	return false
}

// Name implements proto.Repository.
func (r repository) Name() string {
	return filepath.Base(r.r.Path)
}

// Open implements proto.Repository.
func (r repository) Open() (*git.Repository, error) {
	return r.r, nil
}

// ProjectName implements proto.Repository.
func (r repository) ProjectName() string {
	return r.Name()
}

// UpdatedAt implements proto.Repository.
func (r repository) UpdatedAt() time.Time {
	t, err := r.r.LatestCommitTime()
	if err != nil {
		return time.Time{}
	}

	return t
}

// UserID implements proto.Repository.
func (r repository) UserID() int64 {
	return 0
}
