package main

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"runtime/debug"

	"github.com/charmbracelet/log"
	"github.com/charmbracelet/soft-serve/internal/logger"
	"github.com/charmbracelet/soft-serve/server/access"
	_ "github.com/charmbracelet/soft-serve/server/access/sqlite" // access driver
	"github.com/charmbracelet/soft-serve/server/auth"
	_ "github.com/charmbracelet/soft-serve/server/auth/sqlite" // auth driver
	"github.com/charmbracelet/soft-serve/server/backend"
	"github.com/charmbracelet/soft-serve/server/cache"
	"github.com/charmbracelet/soft-serve/server/cache/lru"
	_ "github.com/charmbracelet/soft-serve/server/cache/lru"  // cache driver
	_ "github.com/charmbracelet/soft-serve/server/cache/noop" // cache driver
	"github.com/charmbracelet/soft-serve/server/config"
	"github.com/charmbracelet/soft-serve/server/db"
	_ "github.com/charmbracelet/soft-serve/server/db/sqlite" // db driver
	"github.com/charmbracelet/soft-serve/server/settings"
	_ "github.com/charmbracelet/soft-serve/server/settings/sqlite" // settings driver
	"github.com/charmbracelet/soft-serve/server/store"
	_ "github.com/charmbracelet/soft-serve/server/store/sqlite" // store driver
	"github.com/go-git/go-billy/v5/osfs"
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

	configPath string

	ojson bool

	rootCmd = &cobra.Command{
		Use:          "soft",
		Short:        "A self-hostable Git server for the command line",
		Long:         "Soft Serve is a self-hostable Git server for the command line.",
		SilenceUsage: true,
	}
)

func init() {
	rootCmd.PersistentFlags().StringVarP(&configPath, "config", "c", "", "path to config file")
	rootCmd.AddCommand(
		serveCmd,
		manCmd,
		hookCmd,
		migrateConfig,
		authCmd,
		uiCmd,
	)

	for _, cmd := range []*cobra.Command{
		authCmd,
	} {
		cmd.PersistentFlags().BoolVar(&ojson, "json", false, "output as JSON")
	}

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
	ctx := context.Background()

	cfg := config.DefaultConfig()
	if configPath != "" {
		var err error
		if err = config.ParseConfig(cfg, configPath); err != nil {
			log.Fatal(err)
		}
	} else if !cfg.Exist() {
		// Write config to disk.
		if err := cfg.WriteConfig(); err != nil {
			log.Fatalf("write default config: %w", err)
		}
	} else {
		if err := cfg.ReadConfig(); err != nil {
			log.Fatalf("read config: %w", err)
		}
	}

	ctx = config.WithContext(ctx, cfg)
	logger := logger.NewDefaultLogger(ctx)

	// Set global logger
	log.SetDefault(logger)

	// Set the max number of processes to the number of CPUs
	// This is useful when running soft serve in a container
	if _, err := maxprocs.Set(maxprocs.Logger(log.Debugf)); err != nil {
		log.Warn("couldn't set automaxprocs", "error", err)
	}

	ctx = log.WithContext(ctx, logger)

	// Set up cache
	var cacheOpts []cache.Option
	cacheBackend := "noop"
	switch cfg.Cache.Backend {
	case "lru":
		// TODO: make this configurable
		cacheOpts = append(cacheOpts, lru.WithSize(1000))
		cacheBackend = "lru"
	}

	ca, err := cache.New(ctx, cacheBackend, cacheOpts...)
	if err != nil {
		log.Fatalf("create default cache: %w", err)
	}

	ctx = cache.WithContext(ctx, ca)

	// FIXME: move this somewhere and make order not required
	// Set up database
	sdb, err := db.New(ctx, cfg.Database.Driver, cfg.Database.DataSource)
	if err != nil {
		log.Fatalf("create sqlite database: %w", err)
	}

	ctx = db.WithContext(ctx, sdb)

	// Set up auth backend.
	a, err := auth.New(ctx, cfg.Backend.Auth)
	if err != nil {
		log.Fatalf("create auth backend: %w", err)
	}

	ctx = auth.WithContext(ctx, a)

	// Set up store backend
	fs := osfs.New(filepath.Join(cfg.DataPath, "repos"))
	st, err := store.New(ctx, fs, cfg.Backend.Store)
	if err != nil {
		log.Fatalf("create store backend: %w", err)
	}

	ctx = store.WithContext(ctx, st)

	// Set up settings backend.
	s, err := settings.New(ctx, cfg.Backend.Settings)
	if err != nil {
		log.Fatalf("create settings backend: %w", err)
	}

	ctx = settings.WithContext(ctx, s)

	// Set up access backend.
	ac, err := access.New(ctx, cfg.Backend.Access)
	if err != nil {
		log.Fatalf("create access backend: %w", err)
	}

	ctx = access.WithContext(ctx, ac)

	// Set up backend
	be, err := backend.NewBackend(ctx, s, st, a, ac)
	if err != nil {
		log.Fatalf("create sqlite backend: %w", err)
	}

	ctx = backend.WithContext(ctx, be)

	if rootCmd.ExecuteContext(ctx) != nil {
		os.Exit(1)
	}
}

func writeJSON(w io.Writer, t any) error {
	bts, err := json.Marshal(t)
	if err != nil {
		return err
	}
	_, err = w.Write(bts)
	return err
}
