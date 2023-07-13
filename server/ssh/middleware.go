package ssh

import (
	"strings"

	"github.com/charmbracelet/log"
	"github.com/charmbracelet/soft-serve/server/backend"
	"github.com/charmbracelet/soft-serve/server/config"
	"github.com/charmbracelet/ssh"
)

// ContextMiddleware adds the config, backend, and logger to the session context.
func ContextMiddleware(cfg *config.Config, be *backend.Backend, logger *log.Logger) func(ssh.Handler) ssh.Handler {
	return func(sh ssh.Handler) ssh.Handler {
		return func(s ssh.Session) {
			s.Context().SetValue(config.ContextKey, cfg)
			s.Context().SetValue(backend.ContextKey, be)
			s.Context().SetValue(log.ContextKey, logger.WithPrefix("ssh"))
			sh(s)
		}
	}
}

// CommandMiddleware handles git commands and CLI commands.
// This middleware must be run after the ContextMiddleware.
func CommandMiddleware(sh ssh.Handler) ssh.Handler {
	return func(s ssh.Session) {
		func() {
			cmdLine := s.Command()
			_, _, ptyReq := s.Pty()
			if ptyReq {
				return
			}

			switch {
			case len(cmdLine) >= 2 && strings.HasPrefix(cmdLine[0], "git-"):
				handleGit(s)
			default:
				handleCli(s)
			}
		}()
		sh(s)
	}
}
