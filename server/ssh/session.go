package ssh

import (
	"context"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/log"
	. "github.com/charmbracelet/soft-serve/internal/log"
	"github.com/charmbracelet/soft-serve/server/backend"
	"github.com/charmbracelet/soft-serve/server/config"
	"github.com/charmbracelet/soft-serve/server/errors"
	"github.com/charmbracelet/soft-serve/server/store"
	"github.com/charmbracelet/soft-serve/server/ui"
	"github.com/charmbracelet/soft-serve/server/ui/common"
	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
	bm "github.com/charmbracelet/wish/bubbletea"
	"github.com/muesli/termenv"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var tuiSessionCounter = promauto.NewCounterVec(prometheus.CounterOpts{
	Namespace: "soft_serve",
	Subsystem: "ssh",
	Name:      "tui_session_total",
	Help:      "The total number of TUI sessions",
}, []string{"repo", "term"})

var tuiSessionDuration = promauto.NewCounterVec(prometheus.CounterOpts{
	Namespace: "soft_serve",
	Subsystem: "ssh",
	Name:      "tui_session_seconds_total",
	Help:      "The total number of TUI sessions",
}, []string{"repo", "term"})

// SessionHandler is the soft-serve bubbletea ssh session handler.
func SessionHandler(be *backend.Backend, cfg *config.Config) bm.ProgramHandler {
	return func(s ssh.Session) *tea.Program {
		pty, _, active := s.Pty()
		if !active {
			return nil
		}

		var ctx context.Context = s.Context()
		ctx = backend.WithContext(ctx, be)
		ctx = config.WithContext(ctx, cfg)
		cmd := s.Command()
		initialRepo := ""
		if len(cmd) == 1 {
			initialRepo = cmd[0]
			auth := be.AccessLevelByPublicKey(ctx, initialRepo, s.PublicKey())
			if auth < store.ReadOnlyAccess {
				wish.Fatalln(s, errors.ErrUnauthorized)
				return nil
			}
		}

		envs := &sessionEnv{s}
		output := termenv.NewOutput(s, termenv.WithColorCache(true), termenv.WithEnvironment(envs))
		logger := NewDefaultLogger()
		ctx = log.WithContext(ctx, logger)
		c := common.NewCommon(ctx, output, pty.Window.Width, pty.Window.Height)
		c.SetValue(common.ConfigKey, cfg)
		m := ui.New(c, initialRepo)
		p := tea.NewProgram(m,
			tea.WithInput(s),
			tea.WithOutput(s),
			tea.WithAltScreen(),
			tea.WithoutCatchPanics(),
			tea.WithMouseCellMotion(),
			tea.WithContext(ctx),
		)

		tuiSessionCounter.WithLabelValues(initialRepo, pty.Term).Inc()

		start := time.Now()
		go func() {
			<-ctx.Done()
			tuiSessionDuration.WithLabelValues(initialRepo, pty.Term).Add(time.Since(start).Seconds())
		}()

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
