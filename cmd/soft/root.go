package main

import (
	"context"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"
	"time"

	"github.com/charmbracelet/log"
	"github.com/charmbracelet/soft-serve/server/config"
	_ "github.com/lib/pq" // postgres driver
	"github.com/spf13/cobra"
	"go.uber.org/automaxprocs/maxprocs"
	_ "modernc.org/sqlite" // sqlite driver
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
	logger, f, err := newDefaultLogger()
	if err != nil {
		log.Errorf("failed to create logger: %v", err)
	}

	if f != nil {
		defer f.Close() // nolint: errcheck
	}

	// Set global logger
	log.SetDefault(logger)

	var opts []maxprocs.Option
	if config.IsVerbose() {
		opts = append(opts, maxprocs.Logger(log.Debugf))
	}

	// Set the max number of processes to the number of CPUs
	// This is useful when running soft serve in a container
	if _, err := maxprocs.Set(opts...); err != nil {
		log.Warn("couldn't set automaxprocs", "error", err)
	}

	ctx := log.WithContext(context.Background(), logger)
	if err := rootCmd.ExecuteContext(ctx); err != nil {
		os.Exit(1)
	}
}

// newDefaultLogger returns a new logger with default settings.
func newDefaultLogger() (*log.Logger, *os.File, error) {
	dp := config.DataPath()
	cfg, err := config.ParseConfig(filepath.Join(dp, "config.yaml"))
	if err != nil {
		log.Errorf("failed to parse config: %v", err)
	}

	logger := log.NewWithOptions(os.Stderr, log.Options{
		ReportTimestamp: true,
		TimeFormat:      time.DateOnly,
	})

	switch {
	case config.IsVerbose():
		logger.SetReportCaller(true)
		fallthrough
	case config.IsDebug():
		logger.SetLevel(log.DebugLevel)
	}

	logger.SetTimeFormat(cfg.Log.TimeFormat)

	switch strings.ToLower(cfg.Log.Format) {
	case "json":
		logger.SetFormatter(log.JSONFormatter)
	case "logfmt":
		logger.SetFormatter(log.LogfmtFormatter)
	case "text":
		logger.SetFormatter(log.TextFormatter)
	}

	var f *os.File
	if cfg.Log.Path != "" {
		f, err = os.OpenFile(cfg.Log.Path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return nil, nil, err
		}
		logger.SetOutput(f)
	}

	return logger, f, nil
}
