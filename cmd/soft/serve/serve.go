package serve

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/charmbracelet/soft-serve/cmd"
	"github.com/charmbracelet/soft-serve/pkg/backend"
	"github.com/charmbracelet/soft-serve/pkg/config"
	"github.com/charmbracelet/soft-serve/pkg/db"
	"github.com/charmbracelet/soft-serve/pkg/db/migrate"
	"github.com/spf13/cobra"
)

var (
	syncHooks bool

	// Command is the serve command.
	Command = &cobra.Command{
		Use:                "serve",
		Short:              "Start the server",
		Args:               cobra.NoArgs,
		PersistentPreRunE:  cmd.InitBackendContext,
		PersistentPostRunE: cmd.CloseDBContext,
		RunE: func(c *cobra.Command, _ []string) error {
			ctx := c.Context()
			cfg := config.DefaultConfig()
			if cfg.Exist() {
				if err := cfg.ParseFile(); err != nil {
					return fmt.Errorf("parse config file: %w", err)
				}
			} else {
				if err := cfg.WriteConfig(); err != nil {
					return fmt.Errorf("write config file: %w", err)
				}
			}

			if err := cfg.ParseEnv(); err != nil {
				return fmt.Errorf("parse environment variables: %w", err)
			}

			// Create custom hooks directory if it doesn't exist
			customHooksPath := filepath.Join(cfg.DataPath, "hooks")
			if _, err := os.Stat(customHooksPath); err != nil && os.IsNotExist(err) {
				os.MkdirAll(customHooksPath, os.ModePerm) //nolint:errcheck
				// Generate update hook example without executable permissions
				hookPath := filepath.Join(customHooksPath, "update.sample")
				//nolint:gosec
				if err := os.WriteFile(hookPath, []byte(updateHookExample), 0o744); err != nil {
					return fmt.Errorf("failed to generate update hook example: %w", err)
				}
			}

			// Create log directory if it doesn't exist
			logPath := filepath.Join(cfg.DataPath, "log")
			if _, err := os.Stat(logPath); err != nil && os.IsNotExist(err) {
				os.MkdirAll(logPath, os.ModePerm) //nolint:errcheck
			}

			db := db.FromContext(ctx)
			if err := migrate.Migrate(ctx, db); err != nil {
				return fmt.Errorf("migration error: %w", err)
			}

			s, err := NewServer(ctx)
			if err != nil {
				return fmt.Errorf("start server: %w", err)
			}

			if syncHooks {
				be := backend.FromContext(ctx)
				if err := cmd.InitializeHooks(ctx, cfg, be); err != nil {
					return fmt.Errorf("initialize hooks: %w", err)
				}
			}

			lch := make(chan error, 1)
			done := make(chan os.Signal, 1)
			doneOnce := sync.OnceFunc(func() { close(done) })

			signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

			// This endpoint is added for testing purposes
			// It allows us to stop the server from the test suite.
			// This is needed since Windows doesn't support signals.
			if testRun, _ := strconv.ParseBool(os.Getenv("SOFT_SERVE_TESTRUN")); testRun {
				h := s.HTTPServer.Server.Handler
				s.HTTPServer.Server.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.URL.Path == "/__stop" && r.Method == http.MethodHead {
						doneOnce()
						return
					}
					h.ServeHTTP(w, r)
				})
			}

			go func() {
				lch <- s.Start()
				doneOnce()
			}()

			select {
			case err := <-lch:
				if err != nil {
					return fmt.Errorf("server error: %w", err)
				}
			case <-done:
			}

			ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel()
			if err := s.Shutdown(ctx); err != nil {
				return err
			}

			return nil
		},
	}
)

func init() {
	Command.Flags().BoolVarP(&syncHooks, "sync-hooks", "", false, "synchronize hooks for all repositories before running the server")
}

const updateHookExample = `#!/bin/sh
#
# An example hook script to echo information about the push
# and send it to the client.
#
# To enable this hook, rename this file to "update" and make it executable.

refname="$1"
oldrev="$2"
newrev="$3"

# Safety check
if [ -z "$GIT_DIR" ]; then
        echo "Don't run this script from the command line." >&2
        echo " (if you want, you could supply GIT_DIR then run" >&2
        echo "  $0 <ref> <oldrev> <newrev>)" >&2
        exit 1
fi

if [ -z "$refname" -o -z "$oldrev" -o -z "$newrev" ]; then
        echo "usage: $0 <ref> <oldrev> <newrev>" >&2
        exit 1
fi

# Check types
# if $newrev is 0000...0000, it's a commit to delete a ref.
zero=$(git hash-object --stdin </dev/null | tr '[0-9a-f]' '0')
if [ "$newrev" = "$zero" ]; then
        newrev_type=delete
else
        newrev_type=$(git cat-file -t $newrev)
fi

echo "Hi from Soft Serve update hook!"
echo
echo "Repository: $SOFT_SERVE_REPO_NAME"
echo "RefName: $refname"
echo "Change Type: $newrev_type"
echo "Old SHA1: $oldrev"
echo "New SHA1: $newrev"

exit 0
`
