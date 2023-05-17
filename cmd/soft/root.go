package main

import (
	"context"
	"os"
	"runtime/debug"

	"github.com/charmbracelet/log"
	_ "github.com/charmbracelet/soft-serve/internal/init" // initialize registry
	. "github.com/charmbracelet/soft-serve/internal/log"
	"github.com/spf13/cobra"
	"go.uber.org/automaxprocs/maxprocs"
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
	logger := NewDefaultLogger()

	// Set global logger
	log.SetDefault(logger)

	// Set the max number of processes to the number of CPUs
	// This is useful when running soft serve in a container
	if _, err := maxprocs.Set(maxprocs.Logger(logger.Debugf)); err != nil {
		logger.Warn("couldn't set automaxprocs", "error", err)
	}

	ctx := log.WithContext(context.Background(), logger)
	if err := rootCmd.ExecuteContext(ctx); err != nil {
		os.Exit(1)
	}
}
