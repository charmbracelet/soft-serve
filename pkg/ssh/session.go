package ssh

import (
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/colorprofile"
	"github.com/charmbracelet/soft-serve/pkg/access"
	"github.com/charmbracelet/soft-serve/pkg/backend"
	"github.com/charmbracelet/soft-serve/pkg/config"
	"github.com/charmbracelet/soft-serve/pkg/proto"
	"github.com/charmbracelet/soft-serve/pkg/ui/common"
	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish/v2"
	bm "github.com/charmbracelet/wish/v2/bubbletea"
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
	Help:      "The total time spent in TUI sessions",
}, []string{"repo", "term"})

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

	opts := bm.MakeOptions(s)
	opts = append(opts,
		tea.WithAltScreen(),
		tea.WithoutCatchPanics(),
		tea.WithMouseCellMotion(),
		tea.WithContext(ctx),
	)

	if testrun, ok := os.LookupEnv("SOFT_SERVE_NO_COLOR"); ok && testrun == "1" {
		// Disable colors when running tests.
		opts = append(opts, tea.WithColorProfile(colorprofile.NoTTY))
	}

	c := common.NewCommon(ctx, pty.Window.Width, pty.Window.Height)
	c.SetValue(common.ConfigKey, cfg)
	m := NewUI(c, initialRepo)
	p := tea.NewProgram(m, opts...)

	tuiSessionCounter.WithLabelValues(initialRepo, pty.Term).Inc()

	start := time.Now()
	go func() {
		<-ctx.Done()
		tuiSessionDuration.WithLabelValues(initialRepo, pty.Term).Add(time.Since(start).Seconds())
	}()

	return p
}
