package main

import (
	"io"
	"os"
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

		rp = abs

		r, err := git.Open(rp)
		if err != nil {
			return err
		}

		// Bubble Tea uses Termenv default output so we have to use the same
		// thing here.
		output := termenv.DefaultOutput()
		ctx := cmd.Context()
		c := common.NewCommon(ctx, output, 0, 0)
		m := &model{
			m: repo.New(c),
			r: r,
			c: c,
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

type model struct {
	m          *repo.Repo
	f          *footer.Footer
	r          *git.Repository
	c          common.Common
	showFooter bool
}

var _ tea.Model = &model{}

func (m model) repo() proto.Repository {
	return repository{r: m.r}
}

func (m model) repoCmd() tea.Msg {
	return repo.RepoMsg(m.repo())
}

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
	return m.m.ShortHelp()
}

// FullHelp implements help.KeyMap.
func (m model) FullHelp() [][]key.Binding {
	return m.m.FullHelp()
}

// Init implements tea.Model.
func (m *model) Init() tea.Cmd {
	return tea.Batch(
		m.m.Init(),
		m.f.Init(),
		m.repoCmd,
		repo.UpdateRefCmd(m.repo()),
	)
}

// Update implements tea.Model.
func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmds := make([]tea.Cmd, 0)
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.SetSize(msg.Width, msg.Height)
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.c.KeyMap.Help):
			cmds = append(cmds, footer.ToggleFooterCmd)
		case key.Matches(msg, m.c.KeyMap.Quit):
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
	view := m.m.View()
	if m.showFooter {
		view = lipgloss.JoinVertical(lipgloss.Left, view, m.f.View())
	}

	return m.c.Zone.Scan(m.c.Styles.App.Render(view))
}

type repository struct {
	r *git.Repository
}

var _ proto.Repository = repository{}

// Description implements proto.Repository.
func (r repository) Description() string {
	fp := filepath.Join(r.r.Path, "description")
	f, err := os.Open(fp)
	if err != nil {
		return ""
	}

	defer f.Close() // nolint: errcheck
	bts, err := io.ReadAll(f)
	if err != nil {
		return ""
	}

	return string(bts)
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
