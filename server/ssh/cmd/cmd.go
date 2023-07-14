package cmd

import (
	"fmt"
	"net/url"
	"strings"
	"text/template"
	"unicode"

	"github.com/charmbracelet/soft-serve/server/access"
	"github.com/charmbracelet/soft-serve/server/backend"
	"github.com/charmbracelet/soft-serve/server/config"
	"github.com/charmbracelet/soft-serve/server/proto"
	"github.com/charmbracelet/soft-serve/server/sshutils"
	"github.com/charmbracelet/soft-serve/server/utils"
	"github.com/charmbracelet/ssh"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/spf13/cobra"
)

var cliCommandCounter = promauto.NewCounterVec(prometheus.CounterOpts{
	Namespace: "soft_serve",
	Subsystem: "cli",
	Name:      "commands_total",
	Help:      "Total times each command was called",
}, []string{"command"})

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

func cmdName(args []string) string {
	if len(args) == 0 {
		return ""
	}
	return args[0]
}

// RootCommand returns a new cli root command.
func RootCommand(s ssh.Session) *cobra.Command {
	ctx := s.Context()
	cfg := config.FromContext(ctx)
	be := backend.FromContext(ctx)

	args := s.Command()
	cliCommandCounter.WithLabelValues(cmdName(args)).Inc()
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
		repoCommand(),
	)

	rootCmd.SetArgs(args)
	if len(args) == 0 {
		// otherwise it'll default to os.Args, which is not what we want.
		rootCmd.SetArgs([]string{"--help"})
	}
	rootCmd.SetIn(s)
	rootCmd.SetOut(s)
	rootCmd.CompletionOptions.DisableDefaultCmd = true
	rootCmd.SetErr(s.Stderr())

	user, _ := be.UserByPublicKey(s.Context(), s.PublicKey())
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

func checkIfReadable(cmd *cobra.Command, args []string) error {
	var repo string
	if len(args) > 0 {
		repo = args[0]
	}

	ctx := cmd.Context()
	be := backend.FromContext(ctx)
	rn := utils.SanitizeRepo(repo)
	pk := sshutils.PublicKeyFromContext(ctx)
	auth := be.AccessLevelByPublicKey(cmd.Context(), rn, pk)
	if auth < access.ReadOnlyAccess {
		return proto.ErrUnauthorized
	}
	return nil
}

func isPublicKeyAdmin(cfg *config.Config, pk ssh.PublicKey) bool {
	for _, k := range cfg.AdminKeys() {
		if sshutils.KeysEqual(pk, k) {
			return true
		}
	}
	return false
}

func checkIfAdmin(cmd *cobra.Command, _ []string) error {
	ctx := cmd.Context()
	be := backend.FromContext(ctx)
	cfg := config.FromContext(ctx)
	pk := sshutils.PublicKeyFromContext(ctx)
	if isPublicKeyAdmin(cfg, pk) {
		return nil
	}

	user, _ := be.UserByPublicKey(ctx, pk)
	if user == nil {
		return proto.ErrUnauthorized
	}

	if !user.IsAdmin() {
		return proto.ErrUnauthorized
	}

	return nil
}

func checkIfCollab(cmd *cobra.Command, args []string) error {
	var repo string
	if len(args) > 0 {
		repo = args[0]
	}

	ctx := cmd.Context()
	be := backend.FromContext(ctx)
	pk := sshutils.PublicKeyFromContext(ctx)
	rn := utils.SanitizeRepo(repo)
	auth := be.AccessLevelByPublicKey(ctx, rn, pk)
	if auth < access.ReadWriteAccess {
		return proto.ErrUnauthorized
	}
	return nil
}
