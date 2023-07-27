package main

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/log"
	"github.com/charmbracelet/soft-serve/server/backend"
	"github.com/charmbracelet/soft-serve/server/config"
	"github.com/charmbracelet/soft-serve/server/hooks"
	"github.com/spf13/cobra"
)

var (
	// ErrInternalServerError indicates that an internal server error occurred.
	ErrInternalServerError = errors.New("internal server error")

	// Deprecated: this flag is ignored.
	configPath string

	hookCmd = &cobra.Command{
		Use:    "hook",
		Short:  "Run git server hooks",
		Long:   "Handles Soft Serve git server hooks.",
		Hidden: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			logger := log.FromContext(cmd.Context())
			if err := initBackendContext(cmd, args); err != nil {
				logger.Error("failed to initialize backend context", "err", err)
				return ErrInternalServerError
			}

			return nil
		},
		PersistentPostRunE: func(cmd *cobra.Command, args []string) error {
			logger := log.FromContext(cmd.Context())
			if err := closeDBContext(cmd, args); err != nil {
				logger.Error("failed to close backend", "err", err)
				return ErrInternalServerError
			}

			return nil
		},
	}

	// Git hooks read the config from the environment, based on
	// $SOFT_SERVE_DATA_PATH. We already parse the config when the binary
	// starts, so we don't need to do it again.
	// The --config flag is now deprecated.
	hooksRunE = func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		hks := backend.FromContext(ctx)
		cfg := config.FromContext(ctx)

		// This is set in the server before invoking git-receive-pack/git-upload-pack
		repoName := os.Getenv("SOFT_SERVE_REPO_NAME")

		stdin := cmd.InOrStdin()
		stdout := cmd.OutOrStdout()
		stderr := cmd.ErrOrStderr()

		cmdName := cmd.Name()
		customHookPath := filepath.Join(cfg.DataPath, "hooks", cmdName)

		var buf bytes.Buffer
		opts := make([]hooks.HookArg, 0)

		switch cmdName {
		case hooks.PreReceiveHook, hooks.PostReceiveHook:
			scanner := bufio.NewScanner(stdin)
			for scanner.Scan() {
				buf.Write(scanner.Bytes())
				fields := strings.Fields(scanner.Text())
				if len(fields) != 3 {
					return fmt.Errorf("invalid hook input: %s", scanner.Text())
				}
				opts = append(opts, hooks.HookArg{
					OldSha:  fields[0],
					NewSha:  fields[1],
					RefName: fields[2],
				})
			}

			switch cmdName {
			case hooks.PreReceiveHook:
				hks.PreReceive(ctx, stdout, stderr, repoName, opts)
			case hooks.PostReceiveHook:
				hks.PostReceive(ctx, stdout, stderr, repoName, opts)
			}
		case hooks.UpdateHook:
			if len(args) != 3 {
				return fmt.Errorf("invalid update hook input: %s", args)
			}

			hks.Update(ctx, stdout, stderr, repoName, hooks.HookArg{
				OldSha:  args[0],
				NewSha:  args[1],
				RefName: args[2],
			})
		case hooks.PostUpdateHook:
			hks.PostUpdate(ctx, stdout, stderr, repoName, args...)
		}

		// Custom hooks
		if stat, err := os.Stat(customHookPath); err == nil && !stat.IsDir() && stat.Mode()&0o111 != 0 {
			// If the custom hook is executable, run it
			if err := runCommand(ctx, &buf, stdout, stderr, customHookPath, args...); err != nil {
				return fmt.Errorf("failed to run custom hook: %w", err)
			}
		}

		return nil
	}

	preReceiveCmd = &cobra.Command{
		Use:   "pre-receive",
		Short: "Run git pre-receive hook",
		RunE:  hooksRunE,
	}

	updateCmd = &cobra.Command{
		Use:   "update",
		Short: "Run git update hook",
		Args:  cobra.ExactArgs(3),
		RunE:  hooksRunE,
	}

	postReceiveCmd = &cobra.Command{
		Use:   "post-receive",
		Short: "Run git post-receive hook",
		RunE:  hooksRunE,
	}

	postUpdateCmd = &cobra.Command{
		Use:   "post-update",
		Short: "Run git post-update hook",
		RunE:  hooksRunE,
	}
)

func init() {
	hookCmd.PersistentFlags().StringVar(&configPath, "config", "", "path to config file (deprecated)")
	hookCmd.AddCommand(
		preReceiveCmd,
		updateCmd,
		postReceiveCmd,
		postUpdateCmd,
	)
}

func runCommand(ctx context.Context, in io.Reader, out io.Writer, err io.Writer, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdin = in
	cmd.Stdout = out
	cmd.Stderr = err
	return cmd.Run()
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
