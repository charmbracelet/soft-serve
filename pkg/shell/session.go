package shell

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/soft-serve/pkg/access"
	"github.com/charmbracelet/soft-serve/pkg/backend"
	"github.com/charmbracelet/soft-serve/pkg/config"
	"github.com/charmbracelet/soft-serve/pkg/proto"
	"github.com/charmbracelet/soft-serve/pkg/ui/common"
	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
	"github.com/muesli/termenv"
)

// var tuiSessionCounter = promauto.NewCounterVec(prometheus.CounterOpts{
// 	Namespace: "soft_serve",
// 	Subsystem: "ssh",
// 	Name:      "tui_session_total",
// 	Help:      "The total number of TUI sessions",
// }, []string{"repo", "term"})
//
// var tuiSessionDuration = promauto.NewCounterVec(prometheus.CounterOpts{
// 	Namespace: "soft_serve",
// 	Subsystem: "ssh",
// 	Name:      "tui_session_seconds_total",
// 	Help:      "The total number of TUI sessions",
// }, []string{"repo", "term"})

// SessionHandler is the soft-serve bubbletea ssh session handler.
// This middleware must be run after the ContextMiddleware.
func SessionHandler(s ssh.Session) *tea.Program {
	pty, _, active := s.Pty()
	if !active {
		return nil
	}

	ctx := s.Context()
	be := backend.FromContext(ctx)
	cfg := config.FromContext(ctx)
	cmd := s.Command()
	initialRepo := ""
	if len(cmd) == 1 {
		initialRepo = cmd[0]
		auth := be.AccessLevelByPublicKey(ctx, initialRepo, s.PublicKey())
		if auth < access.ReadOnlyAccess {
			wish.Fatalln(s, proto.ErrUnauthorized)
			return nil
		}
	}

	envs := &sessionEnv{s}
	output := termenv.NewOutput(s, termenv.WithColorCache(true), termenv.WithEnvironment(envs))
	c := common.NewCommon(ctx, output, pty.Window.Width, pty.Window.Height)
	c.SetValue(common.ConfigKey, cfg)
	m := NewUI(c, initialRepo)
	p := tea.NewProgram(m,
		tea.WithInput(s),
		tea.WithOutput(s),
		tea.WithAltScreen(),
		tea.WithoutCatchPanics(),
		tea.WithMouseCellMotion(),
		tea.WithContext(ctx),
	)

	// tuiSessionCounter.WithLabelValues(initialRepo, pty.Term).Inc()
	//
	// start := time.Now()
	// go func() {
	// 	<-ctx.Done()
	// 	tuiSessionDuration.WithLabelValues(initialRepo, pty.Term).Add(time.Since(start).Seconds())
	// }()

	return p
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
