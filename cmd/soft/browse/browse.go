package browse

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/charmbracelet/bubbles/v2/key"
	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/lipgloss/v2"
	"github.com/charmbracelet/soft-serve/git"
	"github.com/charmbracelet/soft-serve/pkg/proto"
	"github.com/charmbracelet/soft-serve/pkg/ui/common"
	"github.com/charmbracelet/soft-serve/pkg/ui/components/footer"
	"github.com/charmbracelet/soft-serve/pkg/ui/pages/repo"
	"github.com/spf13/cobra"
)

// Command is the browse command.
var Command = &cobra.Command{
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

		r, err := git.Open(abs)
		if err != nil {
			return fmt.Errorf("failed to open repository: %w", err)
		}

		// Bubble Tea uses Termenv default output so we have to use the same
		// thing here.
		ctx := cmd.Context()
		c := common.NewCommon(ctx, 0, 0)
		c.HideCloneCmd = true
		comps := []common.TabComponent{
			repo.NewReadme(c),
			repo.NewFiles(c),
			repo.NewLog(c),
		}
		if !r.IsBare {
			comps = append(comps, repo.NewStash(c))
		}
		comps = append(comps, repo.NewRefs(c, git.RefsHeads), repo.NewRefs(c, git.RefsTags))
		m := &model{
			model:  repo.New(c, comps...),
			repo:   repository{r},
			common: c,
		}

		m.footer = footer.New(c, m)
		p := tea.NewProgram(m,
			tea.WithAltScreen(),
			tea.WithMouseCellMotion(),
		)

		_, err = p.Run()
		return err
	},
}

type state int

const (
	startState state = iota
	errorState
)

type model struct {
	model      *repo.Repo
	footer     *footer.Footer
	repo       proto.Repository
	common     common.Common
	state      state
	showFooter bool
	error      error
}

var _ tea.Model = &model{}

func (m *model) SetSize(w, h int) {
	m.common.SetSize(w, h)
	style := m.common.Styles.App
	wm := style.GetHorizontalFrameSize()
	hm := style.GetVerticalFrameSize()
	if m.showFooter {
		hm += m.footer.Height()
	}

	m.footer.SetSize(w-wm, h-hm)
	m.model.SetSize(w-wm, h-hm)
}

// ShortHelp implements help.KeyMap.
func (m model) ShortHelp() []key.Binding {
	switch m.state {
	case errorState:
		return []key.Binding{
			m.common.KeyMap.Back,
			m.common.KeyMap.Quit,
			m.common.KeyMap.Help,
		}
	case startState:
		return m.model.ShortHelp()
	default:
		return m.model.ShortHelp()
	}
}

// FullHelp implements help.KeyMap.
func (m model) FullHelp() [][]key.Binding {
	switch m.state {
	case errorState:
		return [][]key.Binding{
			{
				m.common.KeyMap.Back,
			},
			{
				m.common.KeyMap.Quit,
				m.common.KeyMap.Help,
			},
		}
	case startState:
		return m.model.FullHelp()
	default:
		return m.model.FullHelp()
	}
}

// Init implements tea.Model.
func (m *model) Init() tea.Cmd {
	return tea.Batch(
		m.model.Init(),
		m.footer.Init(),
		func() tea.Msg {
			return repo.RepoMsg(m.repo)
		},
		repo.UpdateRefCmd(m.repo),
	)
}

// Update implements tea.Model.
func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	m.common.Logger.Debugf("msg received: %T", msg)
	cmds := make([]tea.Cmd, 0)
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.SetSize(msg.Width, msg.Height)
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, m.common.KeyMap.Back) && m.error != nil:
			m.error = nil
			m.state = startState
			// Always show the footer on error.
			m.showFooter = m.footer.ShowAll()
		case key.Matches(msg, m.common.KeyMap.Help):
			cmds = append(cmds, footer.ToggleFooterCmd)
		case key.Matches(msg, m.common.KeyMap.Quit):
			// Stop bubblezone background workers.
			m.common.Zone.Close()
			return m, tea.Quit
		}
	case tea.MouseClickMsg:
		mouse := msg.Mouse()
		switch mouse.Button {
		case tea.MouseLeft:
			switch {
			case m.common.Zone.Get("footer").InBounds(msg):
				cmds = append(cmds, footer.ToggleFooterCmd)
			}
		}
	case footer.ToggleFooterMsg:
		m.footer.SetShowAll(!m.footer.ShowAll())
		m.showFooter = !m.showFooter
	case common.ErrorMsg:
		m.error = msg
		m.state = errorState
		m.showFooter = true
	}

	f, cmd := m.footer.Update(msg)
	m.footer = f.(*footer.Footer)
	if cmd != nil {
		cmds = append(cmds, cmd)
	}

	r, cmd := m.model.Update(msg)
	m.model = r.(*repo.Repo)
	if cmd != nil {
		cmds = append(cmds, cmd)
	}

	// This fixes determining the height margin of the footer.
	m.SetSize(m.common.Width, m.common.Height)

	return m, tea.Batch(cmds...)
}

// View implements tea.Model.
func (m *model) View() string {
	style := m.common.Styles.App
	wm, hm := style.GetHorizontalFrameSize(), style.GetVerticalFrameSize()
	if m.showFooter {
		hm += m.footer.Height()
	}

	var view string
	switch m.state {
	case startState:
		view = m.model.View()
	case errorState:
		err := m.common.Styles.ErrorTitle.Render("Bummer")
		err += m.common.Styles.ErrorBody.Render(m.error.Error())
		view = m.common.Styles.Error.
			Width(m.common.Width -
				wm -
				m.common.Styles.ErrorBody.GetHorizontalFrameSize()).
			Height(m.common.Height -
				hm -
				m.common.Styles.Error.GetVerticalFrameSize()).
			Render(err)
	}

	if m.showFooter {
		view = lipgloss.JoinVertical(lipgloss.Left, view, m.footer.View())
	}

	return m.common.Zone.Scan(style.Render(view))
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

// CreatedAt implements proto.Repository.
func (r repository) CreatedAt() time.Time {
	return time.Time{}
}
