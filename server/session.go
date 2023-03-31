package server

import (
	"fmt"

	"github.com/aymanbagabas/go-osc52"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/soft-serve/server/backend"
	cm "github.com/charmbracelet/soft-serve/server/cmd"
	"github.com/charmbracelet/soft-serve/server/config"
	"github.com/charmbracelet/soft-serve/ui"
	"github.com/charmbracelet/soft-serve/ui/common"
	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
	bm "github.com/charmbracelet/wish/bubbletea"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	tuiSessionCounter = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "soft_serve",
		Subsystem: "ssh",
		Name:      "tui_session_total",
		Help:      "The total number of TUI sessions",
	}, []string{"key", "user", "repo", "term"})
)

// SessionHandler is the soft-serve bubbletea ssh session handler.
func SessionHandler(cfg *config.Config) bm.ProgramHandler {
	return func(s ssh.Session) *tea.Program {
		ak := backend.MarshalAuthorizedKey(s.PublicKey())
		pty, _, active := s.Pty()
		if !active {
			return nil
		}

		cmd := s.Command()
		initialRepo := ""
		if len(cmd) == 1 {
			initialRepo = cmd[0]
			auth := cfg.Backend.AccessLevel(initialRepo, s.PublicKey())
			if auth < backend.ReadOnlyAccess {
				wish.Fatalln(s, cm.ErrUnauthorized)
				return nil
			}
		}

		envs := s.Environ()
		envs = append(envs, fmt.Sprintf("TERM=%s", pty.Term))
		output := osc52.NewOutput(s, envs)
		c := common.NewCommon(s.Context(), output, pty.Window.Width, pty.Window.Height)
		c.SetValue(common.ConfigKey, cfg)
		m := ui.New(c, initialRepo)
		p := tea.NewProgram(m,
			tea.WithInput(s),
			tea.WithOutput(s),
			tea.WithAltScreen(),
			tea.WithoutCatchPanics(),
			tea.WithMouseCellMotion(),
		)

		tuiSessionCounter.WithLabelValues(ak, s.User(), initialRepo, pty.Term).Inc()

		return p
	}
}
