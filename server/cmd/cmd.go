package cmd

import (
	"fmt"

	appCfg "github.com/charmbracelet/soft-serve/internal/config"
	"github.com/gliderlabs/ssh"
	"github.com/spf13/cobra"
)

var (
	// ErrUnauthorized is returned when the user is not authorized to perform action.
	ErrUnauthorized = fmt.Errorf("Unauthorized")
	// ErrRepoNotFound is returned when the repo is not found.
	ErrRepoNotFound = fmt.Errorf("Repository not found")
	// ErrFileNotFound is returned when the file is not found.
	ErrFileNotFound = fmt.Errorf("File not found")

	usageTemplate = `Usage:{{if .Runnable}}{{if .HasParent }}
  {{.Parent.Use}} {{end}}{{.Use}}{{if .HasAvailableFlags }} [flags]{{end}}{{end}}{{if .HasAvailableSubCommands}}
  {{if .HasParent }}{{.Parent.Use}} {{end}}{{.Use}} [command]{{end}}{{if gt (len .Aliases) 0}}

Aliases:
  {{.NameAndAliases}}{{end}}{{if .HasExample}}

Examples:
{{.Example}}{{end}}{{if .HasAvailableSubCommands}}

Available Commands:{{range .Commands}}{{if (or .IsAvailableCommand (eq .Name "help"))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableLocalFlags}}

Flags:
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableInheritedFlags}}

Global Flags:
{{.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasHelpSubCommands}}

Additional help topics:{{range .Commands}}{{if .IsAdditionalHelpTopicCommand}}
  {{rpad .CommandPath .CommandPathPadding}} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableSubCommands}}

Use "{{.UseLine}} [command] --help" for more information about a command.{{end}}
`
)

// RootCommand is the root command for the server.
func RootCommand() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:                   "ssh [-p PORT] HOST",
		Long:                  "Soft Serve is a self-hostable Git server for the command line.",
		Args:                  cobra.MinimumNArgs(1),
		DisableFlagsInUseLine: true,
	}
	rootCmd.SetUsageTemplate(usageTemplate)
	rootCmd.CompletionOptions.DisableDefaultCmd = true
	rootCmd.AddCommand(
		ReloadCommand(),
		CatCommand(),
		ListCommand(),
		GitCommand(),
	)

	return rootCmd
}

func fromContext(cmd *cobra.Command) (*appCfg.Config, ssh.Session) {
	ctx := cmd.Context()
	ac := ctx.Value("config").(*appCfg.Config)
	s := ctx.Value("session").(ssh.Session)
	return ac, s
}
