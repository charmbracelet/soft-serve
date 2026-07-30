package cmd

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"text/template"
	"unicode"

	"charm.land/ssh"
	"github.com/charmbracelet/soft-serve/pkg/access"
	"github.com/charmbracelet/soft-serve/pkg/backend"
	"github.com/charmbracelet/soft-serve/pkg/config"
	"github.com/charmbracelet/soft-serve/pkg/proto"
	"github.com/charmbracelet/soft-serve/pkg/sshutils"
	"github.com/charmbracelet/soft-serve/pkg/utils"
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
	return t.Execute(c.OutOrStderr(), struct {
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
	ctx := cmd.Context()
	if repoAccessLevel(ctx, repoArg(args)) < access.ReadOnlyAccess {
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

// isServerAdmin reports whether the caller is a server administrator: either
// their public key is one of the configured admin keys, or their account has
// the admin flag set.
//
// This is the single source of truth for "is this caller a server admin".
// Every authorization gate defers to it so the definition cannot drift.
func isServerAdmin(ctx context.Context) bool {
	cfg := config.FromContext(ctx)
	pk := sshutils.PublicKeyFromContext(ctx)
	if IsPublicKeyAdmin(cfg, pk) {
		return true
	}

	user := proto.UserFromContext(ctx)
	return user != nil && user.IsAdmin()
}

// repoArg returns the repository name from a repo-scoped command's arguments.
// Repo-scoped commands always take the repository as their first argument.
func repoArg(args []string) string {
	if len(args) == 0 {
		return ""
	}
	return utils.SanitizeRepo(args[0])
}

// repoAccessLevel returns the caller's access level for the named repository.
// The repository name must already be sanitized, e.g. via repoArg.
func repoAccessLevel(ctx context.Context, repo string) access.AccessLevel {
	be := backend.FromContext(ctx)
	return be.AccessLevelForUser(ctx, repo, proto.UserFromContext(ctx))
}

// checkIfServerAdmin is the authorization gate for global (non-repo-scoped)
// commands such as `user` and `settings`. It allows server admins only.
//
// Unlike checkIfRepoAdmin, it never consults repository access levels, so it
// cannot be bypassed by creating a repository whose name matches the command
// argument. Attach it to the parent command so that every subcommand,
// including ones added later, is gated by default.
func checkIfServerAdmin(cmd *cobra.Command, _ []string) error {
	if !isServerAdmin(cmd.Context()) {
		return proto.ErrUnauthorized
	}
	return nil
}

// checkIfRepoAdmin is the authorization gate for repo-scoped commands that
// require admin access to the repository named by the first argument.
//
// Only use this on commands whose first argument is a repository name. For
// global commands, use checkIfServerAdmin instead.
func checkIfRepoAdmin(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	if isServerAdmin(ctx) {
		return nil
	}

	if proto.UserFromContext(ctx) == nil {
		return proto.ErrUnauthorized
	}

	if repoAccessLevel(ctx, repoArg(args)) < access.AdminAccess {
		return proto.ErrUnauthorized
	}

	return nil
}

// checkIfRepoCollab is the authorization gate for repo-scoped commands that
// require write access to the repository named by the first argument.
//
// Only use this on commands whose first argument is a repository name.
func checkIfRepoCollab(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	if repoAccessLevel(ctx, repoArg(args)) < access.ReadWriteAccess {
		return proto.ErrUnauthorized
	}
	return nil
}

func checkIfReadableAndCollab(cmd *cobra.Command, args []string) error {
	if err := checkIfReadable(cmd, args); err != nil {
		return err
	}
	if err := checkIfRepoCollab(cmd, args); err != nil {
		return err
	}
	return nil
}
