package cmd

import (
	"context"
	"fmt"

	"github.com/charmbracelet/soft-serve/server/backend"
	"github.com/charmbracelet/soft-serve/server/config"
	"github.com/charmbracelet/soft-serve/server/utils"
	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
	"github.com/spf13/cobra"
)

// ContextKey is a type that can be used as a key in a context.
type ContextKey string

// String returns the string representation of the ContextKey.
func (c ContextKey) String() string {
	return string(c) + "ContextKey"
}

var (
	// ConfigCtxKey is the key for the config in the context.
	ConfigCtxKey = ContextKey("config")
	// SessionCtxKey is the key for the session in the context.
	SessionCtxKey = ContextKey("session")
)

var (
	// ErrUnauthorized is returned when the user is not authorized to perform action.
	ErrUnauthorized = fmt.Errorf("Unauthorized")
	// ErrRepoNotFound is returned when the repo is not found.
	ErrRepoNotFound = fmt.Errorf("Repository not found")
	// ErrFileNotFound is returned when the file is not found.
	ErrFileNotFound = fmt.Errorf("File not found")
)

// rootCommand is the root command for the server.
func rootCommand() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:          "soft",
		Short:        "Soft Serve is a self-hostable Git server for the command line.",
		SilenceUsage: true,
	}
	// TODO: use command usage template to include hostname and port
	rootCmd.CompletionOptions.DisableDefaultCmd = true
	rootCmd.AddCommand(
		adminCommand(),
		branchCommand(),
		collabCommand(),
		createCommand(),
		deleteCommand(),
		descriptionCommand(),
		listCommand(),
		privateCommand(),
		renameCommand(),
		blobCommand(),
		tagCommand(),
		treeCommand(),
	)

	return rootCmd
}

func fromContext(cmd *cobra.Command) (*config.Config, ssh.Session) {
	ctx := cmd.Context()
	cfg := ctx.Value(ConfigCtxKey).(*config.Config)
	s := ctx.Value(SessionCtxKey).(ssh.Session)
	return cfg, s
}

func checkIfReadable(cmd *cobra.Command, args []string) error {
	var repo string
	if len(args) > 0 {
		repo = args[0]
	}
	cfg, s := fromContext(cmd)
	rn := utils.SanitizeRepo(repo)
	auth := cfg.Backend.AccessLevel(rn, s.PublicKey())
	if auth < backend.ReadOnlyAccess {
		return ErrUnauthorized
	}
	return nil
}

func checkIfAdmin(cmd *cobra.Command, args []string) error {
	cfg, s := fromContext(cmd)
	if !cfg.Backend.IsAdmin(s.PublicKey()) {
		return ErrUnauthorized
	}
	return nil
}

func checkIfCollab(cmd *cobra.Command, args []string) error {
	var repo string
	if len(args) > 0 {
		repo = args[0]
	}
	cfg, s := fromContext(cmd)
	rn := utils.SanitizeRepo(repo)
	auth := cfg.Backend.AccessLevel(rn, s.PublicKey())
	if auth < backend.ReadWriteAccess {
		return ErrUnauthorized
	}
	return nil
}

// Middleware is the Soft Serve middleware that handles SSH commands.
func Middleware(cfg *config.Config) wish.Middleware {
	return func(sh ssh.Handler) ssh.Handler {
		return func(s ssh.Session) {
			func() {
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

				ctx := context.WithValue(s.Context(), ConfigCtxKey, cfg)
				ctx = context.WithValue(ctx, SessionCtxKey, s)

				rootCmd := rootCommand()
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
			}()
			sh(s)
		}
	}
}
