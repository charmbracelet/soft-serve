package server

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/charmbracelet/soft-serve/server/backend/file"
	"github.com/charmbracelet/soft-serve/server/config"
	"github.com/charmbracelet/ssh"
	bm "github.com/charmbracelet/wish/bubbletea"
	"github.com/charmbracelet/wish/testsession"
	"github.com/matryer/is"
	"github.com/muesli/termenv"
	gossh "golang.org/x/crypto/ssh"
)

func TestSession(t *testing.T) {
	is := is.New(t)
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
	tb.Helper()
	is := is.New(tb)
	dp := tb.TempDir()
	is.NoErr(os.Setenv("SOFT_SERVE_DATA_PATH", dp))
	is.NoErr(os.Setenv("SOFT_SERVE_GIT_LISTEN_ADDR", ":9418"))
	is.NoErr(os.Setenv("SOFT_SERVE_SSH_LISTEN_ADDR", fmt.Sprintf(":%d", randomPort())))
	tb.Cleanup(func() {
		is.NoErr(os.Unsetenv("SOFT_SERVE_DATA_PATH"))
		is.NoErr(os.Unsetenv("SOFT_SERVE_GIT_LISTEN_ADDR"))
		is.NoErr(os.Unsetenv("SOFT_SERVE_SSH_LISTEN_ADDR"))
		is.NoErr(os.RemoveAll(dp))
	})
	fb, err := file.NewFileBackend(filepath.Join(dp, "repos"))
	if err != nil {
		log.Fatal(err)
	}
	cfg := config.DefaultConfig().WithBackend(fb).WithAccessMethod(fb)
	return testsession.New(tb, &ssh.Server{
		Handler: bm.MiddlewareWithProgramHandler(SessionHandler(cfg), termenv.ANSI256)(func(s ssh.Session) {
			_, _, active := s.Pty()
			tb.Logf("PTY active %v", active)
			tb.Log(s.Command())
			s.Exit(0)
		}),
	}, nil)
}
