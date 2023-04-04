package main

import (
	"os"
	"runtime/debug"

	_ "github.com/charmbracelet/soft-serve/log"
	"github.com/spf13/cobra"
)

var (
	// Version contains the application version number. It's set via ldflags
	// when building.
	Version = ""

	// CommitSHA contains the SHA of the commit that this application was built
	// against. It's set via ldflags when building.
	CommitSHA = ""

	rootCmd = &cobra.Command{
		Use:          "soft",
		Short:        "A self-hostable Git server for the command line",
		Long:         "Soft Serve is a self-hostable Git server for the command line.",
		SilenceUsage: true,
	}
)

func init() {
	rootCmd.AddCommand(
		serveCmd,
		manCmd,
		hookCmd,
		migrateConfig,
	)
	rootCmd.CompletionOptions.HiddenDefaultCmd = true

	if len(CommitSHA) >= 7 {
		vt := rootCmd.VersionTemplate()
		rootCmd.SetVersionTemplate(vt[:len(vt)-1] + " (" + CommitSHA[0:7] + ")\n")
	}
	if Version == "" {
		if info, ok := debug.ReadBuildInfo(); ok && info.Main.Sum != "" {
			Version = info.Main.Version
		} else {
			Version = "unknown (built from source)"
		}
	}
	rootCmd.Version = Version
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
