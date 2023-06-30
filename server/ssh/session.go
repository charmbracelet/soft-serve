package ssh

import (
	"context"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/log"
	"github.com/charmbracelet/soft-serve/server/access"
	"github.com/charmbracelet/soft-serve/server/auth"
	"github.com/charmbracelet/soft-serve/server/backend"
	"github.com/charmbracelet/soft-serve/server/config"
	"github.com/charmbracelet/soft-serve/server/errors"
	"github.com/charmbracelet/soft-serve/server/sshutils"
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
func SessionHandler(ctx context.Context) bm.ProgramHandler {
	be := backend.FromContext(ctx)
	cfg := config.FromContext(ctx)
	logger := log.FromContext(ctx).WithPrefix("ssh-pty")
	return func(s ssh.Session) *tea.Program {
		ctx := backend.WithContext(s.Context(), be)
		ctx = config.WithContext(ctx, cfg)
		ctx = log.WithContext(ctx, logger)

		ak := sshutils.MarshalAuthorizedKey(s.PublicKey())
		pty, _, active := s.Pty()
		if !active {
			return nil
		}

		cmd := s.Command()
		var initialRepo string
		if len(cmd) == 1 {
			user, _ := be.Authenticate(ctx, auth.NewPublicKey(s.PublicKey()))
			initialRepo = cmd[0]
			auth, _ := be.AccessLevel(ctx, initialRepo, user)
			if auth < access.ReadOnlyAccess {
				wish.Fatalln(s, errors.ErrUnauthorized)
				return nil
			}
		}

		envs := &sessionEnv{s}
		output := lipgloss.NewRenderer(s, termenv.WithColorCache(true), termenv.WithEnvironment(envs))
		// FIXME: detect color profile and dark background from ssh.Session
		output.SetColorProfile(termenv.ANSI256)
		output.SetHasDarkBackground(true)
		c := common.NewCommon(ctx, output, pty.Window.Width, pty.Window.Height)
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
