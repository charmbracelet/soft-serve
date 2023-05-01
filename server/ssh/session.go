package ssh

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/soft-serve/server/backend"
	"github.com/charmbracelet/soft-serve/server/config"
	"github.com/charmbracelet/soft-serve/server/errors"
	"github.com/charmbracelet/soft-serve/server/ui"
	"github.com/charmbracelet/soft-serve/server/ui/common"
	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
	bm "github.com/charmbracelet/wish/bubbletea"
	"github.com/muesli/termenv"
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
			auth := cfg.Backend.AccessLevelByPublicKey(initialRepo, s.PublicKey())
			if auth < backend.ReadOnlyAccess {
				wish.Fatalln(s, errors.ErrUnauthorized)
				return nil
			}
		}

		envs := &sessionEnv{s}
		output := termenv.NewOutput(s, termenv.WithColorCache(true), termenv.WithEnvironment(envs))
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

var _ termenv.Environ = &sessionEnv{}

type sessionEnv struct {
	ssh.Session
}

func (s *sessionEnv) Environ() []string {
	pty, _, _ := s.Pty()
	return append(s.Session.Environ(), "TERM="+pty.Term)
}

func (s *sessionEnv) Getenv(key string) string {
	for _, env := range s.Environ() {
		if strings.HasPrefix(env, key+"=") {
			return strings.TrimPrefix(env, key+"=")
		}
	}
	return ""
}
