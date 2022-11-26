package server

import (
	"bytes"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	appCfg "github.com/charmbracelet/soft-serve/config"
	cm "github.com/charmbracelet/soft-serve/server/cmd"
	"github.com/charmbracelet/soft-serve/server/config"
	bm "github.com/charmbracelet/wish/bubbletea"
	"github.com/charmbracelet/wish/testsession"
	"github.com/gliderlabs/ssh"
	"github.com/matryer/is"
	"github.com/muesli/termenv"
	gossh "golang.org/x/crypto/ssh"
)

func TestSession(t *testing.T) {
	is := is.New(t)
	t.Run("unauthorized repo access", func(t *testing.T) {
		var out bytes.Buffer
		s := setup(t)
		s.Stderr = &out
		defer s.Close()
		err := s.RequestPty("xterm", 80, 40, nil)
		is.NoErr(err)
		err = s.Run("config")
		// Session writes error and exits
		is.True(strings.Contains(out.String(), cm.ErrUnauthorized.Error()))
		var ee *gossh.ExitError
		is.True(errors.As(err, &ee) && ee.ExitStatus() == 1)
	})
	t.Run("authorized repo access", func(t *testing.T) {
		s := setup(t)
		s.Stderr = os.Stderr
		defer s.Close()
		err := s.RequestPty("xterm", 80, 40, nil)
		is.NoErr(err)
		go func() {
			time.Sleep(1 * time.Second)
			s.Signal(gossh.SIGTERM)
			// FIXME: exit with code 0 instead of forcibly closing the session
			s.Close()
		}()
		err = s.Run("test")
		var ee *gossh.ExitMissingError
		is.True(errors.As(err, &ee))
	})
}

func setup(tb testing.TB) *gossh.Session {
	is := is.New(tb)
	tb.Helper()
	cfg.DataPath = tb.TempDir()
	ac, err := appCfg.NewConfig(&config.Config{
		SSH:      config.SSHConfig{Port: 22226},
		DataPath: tb.TempDir(),
		InitialAdminKeys: []string{
			"ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIMJlb/qf2B2kMNdBxfpCQqI2ctPcsOkdZGVh5zTRhKtH",
		},
	})
	ac.AnonAccess = "read-only"
	is.NoErr(err)
	return testsession.New(tb, &ssh.Server{
		Handler: bm.MiddlewareWithProgramHandler(SessionHandler(ac), termenv.ANSI256)(func(s ssh.Session) {
			_, _, active := s.Pty()
			tb.Logf("PTY active %v", active)
			tb.Log(s.Command())
			s.Exit(0)
		}),
	}, nil)
}
