package internal

import (
	"context"

	"github.com/charmbracelet/soft-serve/server/config"
	"github.com/charmbracelet/soft-serve/server/hooks"
	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
	"github.com/spf13/cobra"
)

var (
	hooksCtxKey   = "hooks"
	sessionCtxKey = "session"
	configCtxKey  = "config"
)

// rootCommand is the root command for the server.
func rootCommand(cfg *config.Config, s ssh.Session) *cobra.Command {
	rootCmd := &cobra.Command{
		Short:        "Soft Serve internal API.",
		SilenceUsage: true,
	}

	rootCmd.SetIn(s)
	rootCmd.SetOut(s)
	rootCmd.SetErr(s)
	rootCmd.CompletionOptions.DisableDefaultCmd = true

	rootCmd.AddCommand(
		hookCommand(),
	)

	return rootCmd
}

// Middleware returns the middleware for the server.
func (i *InternalServer) Middleware(hooks hooks.Hooks) wish.Middleware {
	return func(sh ssh.Handler) ssh.Handler {
		return func(s ssh.Session) {
			_, _, active := s.Pty()
			if active {
				return
			}

			// Ignore git server commands.
			args := s.Command()
			if len(args) > 0 {
				if args[0] == "git-receive-pack" ||
					args[0] == "git-upload-pack" ||
					args[0] == "git-upload-archive" {
					return
				}
			}

			ctx := context.WithValue(s.Context(), hooksCtxKey, hooks)
			ctx = context.WithValue(ctx, sessionCtxKey, s)
			ctx = context.WithValue(ctx, configCtxKey, i.cfg)

			rootCmd := rootCommand(i.cfg, s)
			rootCmd.SetArgs(args)
			if len(args) == 0 {
				// otherwise it'll default to os.Args, which is not what we want.
				rootCmd.SetArgs([]string{"--help"})
			}
			rootCmd.SetIn(s)
			rootCmd.SetOut(s)
			rootCmd.CompletionOptions.DisableDefaultCmd = true
			rootCmd.SetErr(s.Stderr())
			if err := rootCmd.ExecuteContext(ctx); err != nil {
				_ = s.Exit(1)
			}
			sh(s)
		}
	}
}

func fromContext(cmd *cobra.Command) (*config.Config, ssh.Session) {
	ctx := cmd.Context()
	cfg := ctx.Value(configCtxKey).(*config.Config)
	s := ctx.Value(sessionCtxKey).(ssh.Session)
	return cfg, s
}
