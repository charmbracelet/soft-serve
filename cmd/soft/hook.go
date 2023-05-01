package main

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/soft-serve/server/backend"
	"github.com/charmbracelet/soft-serve/server/backend/sqlite"
	"github.com/charmbracelet/soft-serve/server/config"
	"github.com/charmbracelet/soft-serve/server/hooks"
	"github.com/spf13/cobra"
)

var (
	confixCtxKey  = "config"
	backendCtxKey = "backend"
)

var (
	configPath string

	hookCmd = &cobra.Command{
		Use:    "hook",
		Short:  "Run git server hooks",
		Long:   "Handles Soft Serve git server hooks.",
		Hidden: true,
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := config.ParseConfig(configPath)
			if err != nil {
				return fmt.Errorf("could not parse config: %w", err)
			}

			// Set up the backend
			// TODO: support other backends
			sb, err := sqlite.NewSqliteBackend(cmd.Context(), cfg)
			if err != nil {
				return fmt.Errorf("failed to create sqlite backend: %w", err)
			}

			cfg = cfg.WithBackend(sb)

			cmd.SetContext(context.WithValue(cmd.Context(), confixCtxKey, cfg))
			cmd.SetContext(context.WithValue(cmd.Context(), backendCtxKey, sb))

			return nil
		},
	}

	hooksRunE = func(cmd *cobra.Command, args []string) error {
		cfg := cmd.Context().Value(confixCtxKey).(*config.Config)
		hks := cfg.Backend.(backend.Hooks)

		// This is set in the server before invoking git-receive-pack/git-upload-pack
		repoName := os.Getenv("SOFT_SERVE_REPO_NAME")

		in := cmd.InOrStdin()
		out := cmd.OutOrStdout()
		err := cmd.ErrOrStderr()

		cmdName := cmd.Name()
		switch cmdName {
		case hooks.PreReceiveHook, hooks.PostReceiveHook:
			var buf bytes.Buffer
			opts := make([]backend.HookArg, 0)
			scanner := bufio.NewScanner(in)
			for scanner.Scan() {
				buf.Write(scanner.Bytes())
				fields := strings.Fields(scanner.Text())
				if len(fields) != 3 {
					return fmt.Errorf("invalid pre-receive hook input: %s", scanner.Text())
				}
				opts = append(opts, backend.HookArg{
					OldSha:  fields[0],
					NewSha:  fields[1],
					RefName: fields[2],
				})
			}

			switch cmdName {
			case hooks.PreReceiveHook:
				hks.PreReceive(out, err, repoName, opts)
			case hooks.PostReceiveHook:
				hks.PostReceive(out, err, repoName, opts)
			}
		case hooks.UpdateHook:
			if len(args) != 3 {
				return fmt.Errorf("invalid update hook input: %s", args)
			}

			hks.Update(out, err, repoName, backend.HookArg{
				OldSha:  args[0],
				NewSha:  args[1],
				RefName: args[2],
			})
		case hooks.PostUpdateHook:
			hks.PostUpdate(out, err, repoName, args...)
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
	hookCmd.PersistentFlags().StringVarP(&configPath, "config", "c", "", "path to config file")
	hookCmd.AddCommand(
		preReceiveCmd,
		updateCmd,
		postReceiveCmd,
		postUpdateCmd,
	)
}
