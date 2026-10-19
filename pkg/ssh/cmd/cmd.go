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

// denyReadAs is the one answer for a caller who may not read a repository,
// whether it is missing or private: anon for anonymous callers (HTTP 401),
// registered for everyone else (HTTP 404).
func denyReadAs(anonymous bool, anon, registered error) error {
	if anonymous {
		return anon
	}
	return registered
}

// denyRead is denyReadAs for repo commands. Server admins always get
// ErrRepoNotFound since nothing is hidden from them.
func denyRead(ctx context.Context) error {
	anonymous := proto.UserFromContext(ctx) == nil && !isServerAdmin(ctx)
	return denyReadAs(anonymous, proto.ErrUnauthorized, proto.ErrRepoNotFound)
}

// repoContextKey holds the repoStash a gate stored for the command.
type repoContextKey struct{}

// repoStash is the repository a gate resolved for a command and the
// caller's access level on it. repo is nil only for server admins acting
// on a name with no repository behind it.
type repoStash struct {
	name  string
	repo  proto.Repository
	level access.AccessLevel
}

// requireRepo gates a repo-scoped command: the caller must be able to read
// the repository named by the first argument and hold at least need on it.
// Readability is decided before existence, so a caller who cannot read gets
// denyRead's answer whether or not the repository exists. Server admins
// pass with a nil repository so RunE can act on orphaned rows.
func requireRepo(need access.AccessLevel) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		name := repoArg(args)
		repo, level := readableRepo(ctx, name)
		if repo == nil && !isServerAdmin(ctx) {
			return denyRead(ctx)
		}

		cmd.SetContext(context.WithValue(ctx, repoContextKey{}, repoStash{name: name, repo: repo, level: level}))
		return requireLevel(cmd.Context(), need)
	}
}

// requireLevel checks the stashed access level against need; use it in
// setter arms of commands gated with requireRepo.
func requireLevel(ctx context.Context, need access.AccessLevel) error {
	if _, level, _ := repoFromContext(ctx); level < need {
		return proto.ErrUnauthorized
	}
	return nil
}

// readableRepo is backend.Readable with the server admin override.
func readableRepo(ctx context.Context, name string) (proto.Repository, access.AccessLevel) {
	be := backend.FromContext(ctx)
	if isServerAdmin(ctx) {
		repo, _ := be.Repository(ctx, name)
		return repo, access.AdminAccess
	}
	return be.Readable(ctx, name, proto.UserFromContext(ctx))
}

// repoFromContext returns the repository and access level stored by
// requireRepo. The error is denyRead's answer when no repository exists.
func repoFromContext(ctx context.Context) (proto.Repository, access.AccessLevel, error) {
	stash, _ := ctx.Value(repoContextKey{}).(repoStash)
	if stash.repo == nil {
		return nil, stash.level, denyRead(ctx)
	}
	return stash.repo, stash.level, nil
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

// repoAccessLevel returns the caller's access level for the sanitized name.
func repoAccessLevel(ctx context.Context, repo string) access.AccessLevel {
	_, level := readableRepo(ctx, repo)
	return level
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

// checkIfRepoCollab gates create and import, which name a repository that
// should not exist yet, on write access to that name. It cannot hide
// whether the name is taken: with one flat namespace, "already exists" is
// the answer either way.
func checkIfRepoCollab(cmd *cobra.Command, args []string) error {
	if repoAccessLevel(cmd.Context(), repoArg(args)) < access.ReadWriteAccess {
		return proto.ErrUnauthorized
	}
	return nil
}
