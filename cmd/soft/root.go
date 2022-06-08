package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime/debug"

	"github.com/aymanbagabas/go-osc52"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	ggit "github.com/charmbracelet/soft-serve/git"
	"github.com/charmbracelet/soft-serve/ui/common"
	"github.com/charmbracelet/soft-serve/ui/components/footer"
	"github.com/charmbracelet/soft-serve/ui/git"
	"github.com/charmbracelet/soft-serve/ui/keymap"
	"github.com/charmbracelet/soft-serve/ui/pages/repo"
	"github.com/charmbracelet/soft-serve/ui/styles"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var (
	// Version contains the application version number. It's set via ldflags
	// when building.
	Version = ""

	// CommitSHA contains the SHA of the commit that this application was built
	// against. It's set via ldflags when building.
	CommitSHA = ""

	rootCmd = &cobra.Command{
		Use:   "soft",
		Short: "Soft Serve, a self-hostable Git server for the command line.",
		Long:  "Soft Serve is a self-hostable Git server for the command line.",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := os.Getwd()
			if err != nil {
				return err
			}
			if len(args) > 0 {
				p := args[0]
				if filepath.IsAbs(p) {
					path = p
				} else {
					path = filepath.Join(path, p)
				}
			}
			path = filepath.Clean(path)
			w, h, _ := term.GetSize(int(os.Stdout.Fd()))
			c := common.Common{
				Styles: styles.DefaultStyles(),
				KeyMap: keymap.DefaultKeyMap(),
				Copy:   osc52.NewOutput(os.Stdout, os.Environ()),
				Width:  w,
				Height: h,
			}
			repo := repo.New(nil, c)
			repo.BackKey.SetHelp("esc", "quit")
			ui := &ui{
				c:    c,
				repo: repo,
				path: path,
			}
			ui.footer = footer.New(c, ui)
			p := tea.NewProgram(ui,
				tea.WithMouseCellMotion(),
				tea.WithAltScreen(),
			)
			if len(os.Getenv("DEBUG")) > 0 {
				f, err := tea.LogToFile("soft.log", "")
				if err != nil {
					log.Fatal(err)
				}
				defer f.Close() // nolint: errcheck
			}
			return p.Start()
		},
	}
)

type state int

const (
	stateLoading state = iota
	stateReady
	stateError
)

type ui struct {
	c      common.Common
	state  state
	repo   *repo.Repo
	footer *footer.Footer
	path   string
	ref    *ggit.Reference
	r      git.GitRepo
	error  error
}

func (u *ui) ShortHelp() []key.Binding {
	return u.repo.ShortHelp()
}

func (u *ui) FullHelp() [][]key.Binding {
	return u.repo.FullHelp()
}

func (u *ui) SetSize(width, height int) {
	u.c.SetSize(width, height)
	hm := u.c.Styles.App.GetVerticalFrameSize()
	wm := u.c.Styles.App.GetHorizontalFrameSize()
	if u.footer.ShowAll() {
		hm += u.footer.Height()
	}
	u.footer.SetSize(width-wm, height-hm)
	u.repo.SetSize(width-wm, height-hm)
}

func (u *ui) Init() tea.Cmd {
	r, err := git.NewRepo(u.path)
	if err != nil {
		return common.ErrorCmd(err)
	}
	h, err := r.HEAD()
	if err != nil {
		return common.ErrorCmd(err)
	}
	u.r = r
	u.ref = h
	return tea.Batch(
		func() tea.Msg {
			return repo.RefMsg(h)
		},
		func() tea.Msg {
			return repo.RepoMsg(r)
		},
	)
}

func (u *ui) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmds := make([]tea.Cmd, 0)
	switch msg := msg.(type) {
	case repo.RefMsg, repo.RepoMsg:
		if u.ref != nil && u.r != nil {
			u.state = stateReady
		}
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, u.c.KeyMap.Help):
			u.footer.SetShowAll(!u.footer.ShowAll())
		case key.Matches(msg, u.c.KeyMap.Quit), key.Matches(msg, u.c.KeyMap.Back):
			return u, tea.Quit
		}
	case tea.WindowSizeMsg:
		u.SetSize(msg.Width, msg.Height)
	case common.ErrorMsg:
		if u.state != stateLoading {
			u.error = msg
			u.state = stateError
		}
	}
	r, cmd := u.repo.Update(msg)
	u.repo = r.(*repo.Repo)
	if cmd != nil {
		cmds = append(cmds, cmd)
	}
	// This fixes determining the height margin of the footer.
	u.SetSize(u.c.Width, u.c.Height)
	return u, tea.Batch(cmds...)
}

func (u *ui) View() string {
	var view string
	switch u.state {
	case stateLoading:
		view = "Loading..."
	case stateReady:
		view = u.repo.View()
		if u.footer.ShowAll() {
			view = lipgloss.JoinVertical(lipgloss.Top,
				view,
				u.footer.View(),
			)
		}
	case stateError:
		view = fmt.Sprintf("Error: %s", u.error)
	}
	return u.c.Styles.App.Render(view)
}

func init() {
	if len(CommitSHA) >= 7 {
		vt := rootCmd.VersionTemplate()
		rootCmd.SetVersionTemplate(vt[:len(vt)-1] + " (" + CommitSHA[0:7] + ")\n")
	}
	if Version == "" {
		if info, ok := debug.ReadBuildInfo(); ok && info.Main.Sum != "" {
			Version = info.Main.Version
		} else {
			Version = "unknown (built from source)"
		}
	}
	rootCmd.Version = Version

	rootCmd.AddCommand(
		serveCmd,
	)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
