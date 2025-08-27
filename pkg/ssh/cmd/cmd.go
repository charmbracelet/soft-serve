package cmd

import (
	"fmt"
	"net/url"
	"strings"
	"text/template"
	"unicode"

	"github.com/charmbracelet/soft-serve/pkg/access"
	"github.com/charmbracelet/soft-serve/pkg/backend"
	"github.com/charmbracelet/soft-serve/pkg/config"
	"github.com/charmbracelet/soft-serve/pkg/proto"
	"github.com/charmbracelet/soft-serve/pkg/sshutils"
	"github.com/charmbracelet/soft-serve/pkg/utils"
	"github.com/charmbracelet/ssh"
	"github.com/spf13/cobra"
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
	// UsageTemplate is the template used for the help output.
	UsageTemplate = `Usage:{{if .Runnable}}
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

// UsageFunc is a function that can be used as a cobra.Command's
// UsageFunc to render the help output.
func UsageFunc(c *cobra.Command) error {
	ctx := c.Context()
	cfg := config.FromContext(ctx)
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
	t := template.New("usage")
	t.Funcs(templateFuncs)
	template.Must(t.Parse(c.UsageTemplate()))
	return t.Execute(c.OutOrStderr(), struct { //nolint:wrapcheck
		*cobra.Command
		SSHCommand string
	}{
		Command:    c,
		SSHCommand: sshCmd,
	})
}

func trimRightSpace(s string) string {
	return strings.TrimRightFunc(s, unicode.IsSpace)
}

// rpad adds padding to the right of a string.
func rpad(s string, padding int) string {
	template := fmt.Sprintf("%%-%ds", padding)
	return fmt.Sprintf(template, s)
}

// CommandName returns the name of the command from the args.
func CommandName(args []string) string {
	if len(args) == 0 {
		return ""
	}
	return args[0]
}

func checkIfReadable(cmd *cobra.Command, args []string) error {
	var repo string
	if len(args) > 0 {
		repo = args[0]
	}

	ctx := cmd.Context()
	be := backend.FromContext(ctx)
	rn := utils.SanitizeRepo(repo)
	user := proto.UserFromContext(ctx)
	auth := be.AccessLevelForUser(cmd.Context(), rn, user)
	if auth < access.ReadOnlyAccess {
		return proto.ErrRepoNotFound
	}
	return nil
}

// IsPublicKeyAdmin returns true if the given public key is an admin key from
// the initial_admin_keys config or environment field.
func IsPublicKeyAdmin(cfg *config.Config, pk ssh.PublicKey) bool {
	for _, k := range cfg.AdminKeys() {
		if sshutils.KeysEqual(pk, k) {
			return true
		}
	}
	return false
}

func checkIfAdmin(cmd *cobra.Command, args []string) error {
	var repo string
	if len(args) > 0 {
		repo = args[0]
	}

	ctx := cmd.Context()
	cfg := config.FromContext(ctx)
	be := backend.FromContext(ctx)
	rn := utils.SanitizeRepo(repo)
	pk := sshutils.PublicKeyFromContext(ctx)
	if IsPublicKeyAdmin(cfg, pk) {
		return nil
	}

	user := proto.UserFromContext(ctx)
	if user == nil {
		return proto.ErrUnauthorized
	}

	if user.IsAdmin() {
		return nil
	}

	auth := be.AccessLevelForUser(cmd.Context(), rn, user)
	if auth >= access.AdminAccess {
		return nil
	}

	return proto.ErrUnauthorized
}

func checkIfCollab(cmd *cobra.Command, args []string) error {
	var repo string
	if len(args) > 0 {
		repo = args[0]
	}

	ctx := cmd.Context()
	be := backend.FromContext(ctx)
	rn := utils.SanitizeRepo(repo)
	user := proto.UserFromContext(ctx)
	auth := be.AccessLevelForUser(cmd.Context(), rn, user)
	if auth < access.ReadWriteAccess {
		return proto.ErrUnauthorized
	}
	return nil
}

func checkIfReadableAndCollab(cmd *cobra.Command, args []string) error {
	if err := checkIfReadable(cmd, args); err != nil {
		return err
	}
	if err := checkIfCollab(cmd, args); err != nil {
		return err
	}
	return nil
}
