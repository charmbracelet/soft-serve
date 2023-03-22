package cmd

import (
	"context"
	"fmt"

	"github.com/charmbracelet/soft-serve/server/config"
	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
)

// Middleware is the Soft Serve middleware that handles SSH commands.
func Middleware(cfg *config.Config) wish.Middleware {
	return func(sh ssh.Handler) ssh.Handler {
		return func(s ssh.Session) {
			func() {
				_, _, active := s.Pty()
				if active {
					return
				}
				ctx := context.WithValue(s.Context(), ConfigCtxKey, cfg)
				ctx = context.WithValue(ctx, SessionCtxKey, s)

				use := "ssh"
				port := cfg.Backend.ServerPort()
				if port != "22" {
					use += fmt.Sprintf(" -p%d", port)
				}
				use += fmt.Sprintf(" %s", cfg.Backend.ServerHost())
				cmd := RootCommand()
				cmd.Use = use
				cmd.CompletionOptions.DisableDefaultCmd = true
				cmd.SetIn(s)
				cmd.SetOut(s)
				cmd.SetErr(s.Stderr())
				cmd.SetArgs(s.Command())
				err := cmd.ExecuteContext(ctx)
				if err != nil {
					_, _ = s.Write([]byte(err.Error()))
					_ = s.Exit(1)
					return
				}
			}()
			sh(s)
		}
	}
}
