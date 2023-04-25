package cmd

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"text/template"
	"unicode"

	"github.com/charmbracelet/log"
	"github.com/charmbracelet/soft-serve/server/backend"
	"github.com/charmbracelet/soft-serve/server/config"
	"github.com/charmbracelet/soft-serve/server/hooks"
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
	// HooksCtxKey is the key for the git hooks in the context.
	HooksCtxKey = ContextKey("hooks")
)

var (
	// ErrUnauthorized is returned when the user is not authorized to perform action.
	ErrUnauthorized = fmt.Errorf("Unauthorized")
	// ErrRepoNotFound is returned when the repo is not found.
	ErrRepoNotFound = fmt.Errorf("Repository not found")
	// ErrFileNotFound is returned when the file is not found.
	ErrFileNotFound = fmt.Errorf("File not found")
)

var (
	logger = log.WithPrefix("server.cmd")
)

var templateFuncs = template.FuncMap{
	"trim":                    strings.TrimSpace,
	"trimRightSpace":          trimRightSpace,
	"trimTrailingWhitespaces": trimRightSpace,
	"rpad":                    rpad,
	"gt":                      cobra.Gt,
	"eq":                      cobra.Eq,
}

const (
	usageTmpl = `Usage:{{if .Runnable}}
  {{.UseLine}}{{end}}{{if .HasAvailableSubCommands}}
  {{.SSHCommand}}{{.CommandPath}} [command]{{end}}{{if gt (len .Aliases) 0}}

Aliases:
  {{.NameAndAliases}}{{end}}{{if .HasExample}}

Examples:
{{.Example}}{{end}}{{if .HasAvailableSubCommands}}{{$cmds := .Commands}}{{if eq (len .Groups) 0}}

Available Commands:{{range $cmds}}{{if (or .IsAvailableCommand (eq .Name "help"))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{else}}{{range $group := .Groups}}

{{.Title}}{{range $cmds}}{{if (and (eq .GroupID $group.ID) (or .IsAvailableCommand (eq .Name "help")))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{if not .AllChildCommandsHaveGroup}}

Additional Commands:{{range $cmds}}{{if (and (eq .GroupID "") (or .IsAvailableCommand (eq .Name "help")))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{end}}{{end}}{{if .HasAvailableLocalFlags}}

Flags:
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableInheritedFlags}}

Global Flags:
{{.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasHelpSubCommands}}

Additional help topics:{{range .Commands}}{{if .IsAdditionalHelpTopicCommand}}
  {{rpad .CommandPath .CommandPathPadding}} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableSubCommands}}

Use "{{.SSHCommand}}{{.CommandPath}} [command] --help" for more information about a command.{{end}}
`
)

func trimRightSpace(s string) string {
	return strings.TrimRightFunc(s, unicode.IsSpace)
}

// rpad adds padding to the right of a string.
func rpad(s string, padding int) string {
	template := fmt.Sprintf("%%-%ds", padding)
	return fmt.Sprintf(template, s)
}

// rootCommand is the root command for the server.
func rootCommand(cfg *config.Config, s ssh.Session) *cobra.Command {
	rootCmd := &cobra.Command{
		Short:        "Soft Serve is a self-hostable Git server for the command line.",
		SilenceUsage: true,
	}

	hostname := "localhost"
	port := "23231"
	url, err := url.Parse(cfg.SSH.PublicURL)
	if err == nil {
		hostname = url.Hostname()
		port = url.Port()
	}

	sshCmd := "ssh"
	if port != "" && port != "22" {
		sshCmd += " -p " + port
	}

	sshCmd += " " + hostname
	rootCmd.SetUsageTemplate(usageTmpl)
	rootCmd.SetUsageFunc(func(c *cobra.Command) error {
		t := template.New("usage")
		t.Funcs(templateFuncs)
		template.Must(t.Parse(c.UsageTemplate()))
		return t.Execute(c.OutOrStderr(), struct {
			*cobra.Command
			SSHCommand string
		}{
			Command:    c,
			SSHCommand: sshCmd,
		})
	})
	rootCmd.CompletionOptions.DisableDefaultCmd = true
	rootCmd.AddCommand(
		hookCommand(),
		repoCommand(),
	)

	user, _ := cfg.Backend.UserByPublicKey(s.PublicKey())
	isAdmin := isPublicKeyAdmin(cfg, s.PublicKey()) || (user != nil && user.IsAdmin())
	if user != nil || isAdmin {
		if isAdmin {
			rootCmd.AddCommand(
				settingsCommand(),
				userCommand(),
			)
		}

		rootCmd.AddCommand(
			infoCommand(),
			pubkeyCommand(),
			setUsernameCommand(),
		)
	}

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
	auth := cfg.Backend.AccessLevelByPublicKey(rn, s.PublicKey())
	if auth < backend.ReadOnlyAccess {
		return ErrUnauthorized
	}
	return nil
}

func isPublicKeyAdmin(cfg *config.Config, pk ssh.PublicKey) bool {
	for _, k := range cfg.InitialAdminKeys {
		pk2, _, err := backend.ParseAuthorizedKey(k)
		if err == nil && backend.KeysEqual(pk, pk2) {
			return true
		}
	}
	return false
}

func checkIfAdmin(cmd *cobra.Command, _ []string) error {
	cfg, s := fromContext(cmd)
	if isPublicKeyAdmin(cfg, s.PublicKey()) {
		return nil
	}

	user, _ := cfg.Backend.UserByPublicKey(s.PublicKey())
	if user == nil {
		return ErrUnauthorized
	}

	if !user.IsAdmin() {
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
	auth := cfg.Backend.AccessLevelByPublicKey(rn, s.PublicKey())
	if auth < backend.ReadWriteAccess {
		return ErrUnauthorized
	}
	return nil
}

// Middleware is the Soft Serve middleware that handles SSH commands.
func Middleware(cfg *config.Config, hooks hooks.Hooks) wish.Middleware {
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
				ctx = context.WithValue(ctx, HooksCtxKey, hooks)

				rootCmd := rootCommand(cfg, s)
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
