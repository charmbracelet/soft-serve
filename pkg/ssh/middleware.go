package ssh

import (
	"fmt"
	"time"

	"github.com/charmbracelet/log"
	"github.com/charmbracelet/soft-serve/pkg/backend"
	"github.com/charmbracelet/soft-serve/pkg/config"
	"github.com/charmbracelet/soft-serve/pkg/db"
	"github.com/charmbracelet/soft-serve/pkg/proto"
	"github.com/charmbracelet/soft-serve/pkg/ssh/cmd"
	"github.com/charmbracelet/soft-serve/pkg/sshutils"
	"github.com/charmbracelet/soft-serve/pkg/store"
	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/spf13/cobra"
	gossh "golang.org/x/crypto/ssh"
)

// ErrPermissionDenied is returned when a user is not allowed connect.
var ErrPermissionDenied = fmt.Errorf("permission denied")

// AuthenticationMiddleware handles authentication.
func AuthenticationMiddleware(sh ssh.Handler) ssh.Handler {
	return func(s ssh.Session) {
		// XXX: The authentication key is set in the context but gossh doesn't
		// validate the authentication. We need to verify that the _last_ key
		// that was approved is the one that's being used.

		pk := s.PublicKey()
		if pk != nil {
			// There is no public key stored in the context, public-key auth
			// was never requested, skip
			perms := s.Permissions().Permissions
			if perms == nil {
				wish.Fatalln(s, ErrPermissionDenied)
				return
			}

			// Check if the key is the same as the one we have in context
			fp := perms.Extensions["pubkey-fp"]
			if fp != gossh.FingerprintSHA256(pk) {
				wish.Fatalln(s, ErrPermissionDenied)
				return
			}
		}

		sh(s)
	}
}

// ContextMiddleware adds the config, backend, and logger to the session context.
func ContextMiddleware(cfg *config.Config, dbx *db.DB, datastore store.Store, be *backend.Backend, logger *log.Logger) func(ssh.Handler) ssh.Handler {
	return func(sh ssh.Handler) ssh.Handler {
		return func(s ssh.Session) {
			s.Context().SetValue(sshutils.ContextKeySession, s)
			s.Context().SetValue(config.ContextKey, cfg)
			s.Context().SetValue(db.ContextKey, dbx)
			s.Context().SetValue(store.ContextKey, datastore)
			s.Context().SetValue(backend.ContextKey, be)
			s.Context().SetValue(log.ContextKey, logger.WithPrefix("ssh"))
			sh(s)
		}
	}
}

var cliCommandCounter = promauto.NewCounterVec(prometheus.CounterOpts{
	Namespace: "soft_serve",
	Subsystem: "cli",
	Name:      "commands_total",
	Help:      "Total times each command was called",
}, []string{"command"})

// CommandMiddleware handles git commands and CLI commands.
// This middleware must be run after the ContextMiddleware.
func CommandMiddleware(sh ssh.Handler) ssh.Handler {
	return func(s ssh.Session) {
		func() {
			_, _, ptyReq := s.Pty()
			if ptyReq {
				return
			}

			ctx := s.Context()
			cfg := config.FromContext(ctx)

			args := s.Command()
			cliCommandCounter.WithLabelValues(cmd.CommandName(args)).Inc()
			rootCmd := &cobra.Command{
				Short:        "Soft Serve is a self-hostable Git server for the command line.",
				SilenceUsage: true,
			}
			rootCmd.CompletionOptions.DisableDefaultCmd = true

			rootCmd.SetUsageTemplate(cmd.UsageTemplate)
			rootCmd.SetUsageFunc(cmd.UsageFunc)
			rootCmd.AddCommand(
				cmd.GitUploadPackCommand(),
				cmd.GitUploadArchiveCommand(),
				cmd.GitReceivePackCommand(),
				cmd.RepoCommand(),
				cmd.SettingsCommand(),
				cmd.UserCommand(),
				cmd.OrgCommand(),
				cmd.InfoCommand(),
				cmd.PubkeyCommand(),
				cmd.SetUsernameCommand(),
				cmd.JWTCommand(),
				cmd.TokenCommand(),
			)

			if cfg.LFS.Enabled {
				rootCmd.AddCommand(
					cmd.GitLFSAuthenticateCommand(),
				)

				if cfg.LFS.SSHEnabled {
					rootCmd.AddCommand(
						cmd.GitLFSTransfer(),
					)
				}
			}

			rootCmd.SetArgs(args)
			if len(args) == 0 {
				// otherwise it'll default to os.Args, which is not what we want.
				rootCmd.SetArgs([]string{"--help"})
			}
			rootCmd.SetIn(s)
			rootCmd.SetOut(s)
			rootCmd.SetErr(s.Stderr())
			rootCmd.SetContext(ctx)

			if err := rootCmd.ExecuteContext(ctx); err != nil {
				s.Exit(1) // nolint: errcheck
				return
			}
		}()
		sh(s)
	}
}

// LoggingMiddleware logs the ssh connection and command.
func LoggingMiddleware(sh ssh.Handler) ssh.Handler {
	return func(s ssh.Session) {
		ctx := s.Context()
		logger := log.FromContext(ctx).WithPrefix("ssh")
		ct := time.Now()
		hpk := sshutils.MarshalAuthorizedKey(s.PublicKey())
		ptyReq, _, isPty := s.Pty()
		addr := s.RemoteAddr().String()
		user := proto.UserFromContext(ctx)
		logArgs := []interface{}{
			"addr",
			addr,
			"cmd",
			s.Command(),
		}

		if user != nil {
			logArgs = append([]interface{}{
				"username",
				user.Username(),
			}, logArgs...)
		}

		if isPty {
			logArgs = []interface{}{
				"term", ptyReq.Term,
				"width", ptyReq.Window.Width,
				"height", ptyReq.Window.Height,
			}
		}

		if config.IsVerbose() {
			logArgs = append(logArgs,
				"key", hpk,
				"envs", s.Environ(),
			)
		}

		msg := fmt.Sprintf("user %q", s.User())
		logger.Debug(msg+" connected", logArgs...)
		sh(s)
		logger.Debug(msg+" disconnected", append(logArgs, "duration", time.Since(ct))...)
	}
}
