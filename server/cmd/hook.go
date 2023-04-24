package cmd

import (
	"bufio"
	"fmt"
	"strings"

	"github.com/charmbracelet/keygen"
	"github.com/charmbracelet/soft-serve/server/hooks"
	"github.com/charmbracelet/ssh"
	"github.com/spf13/cobra"
)

// hookCommand handles Soft Serve internal API git hook requests.
func hookCommand() *cobra.Command {
	preReceiveCmd := &cobra.Command{
		Use:               "pre-receive",
		Short:             "Run git pre-receive hook",
		PersistentPreRunE: checkIfInternal,
		RunE: func(cmd *cobra.Command, args []string) error {
			_, s := fromContext(cmd)
			hks := cmd.Context().Value(HooksCtxKey).(hooks.Hooks)
			repoName := getRepoName(s)
			opts := make([]hooks.HookArg, 0)
			scanner := bufio.NewScanner(s)
			for scanner.Scan() {
				fields := strings.Fields(scanner.Text())
				if len(fields) != 3 {
					return fmt.Errorf("invalid pre-receive hook input: %s", scanner.Text())
				}
				opts = append(opts, hooks.HookArg{
					OldSha:  fields[0],
					NewSha:  fields[1],
					RefName: fields[2],
				})
			}
			hks.PreReceive(s, s.Stderr(), repoName, opts)
			return nil
		},
	}

	updateCmd := &cobra.Command{
		Use:               "update",
		Short:             "Run git update hook",
		Args:              cobra.ExactArgs(3),
		PersistentPreRunE: checkIfInternal,
		RunE: func(cmd *cobra.Command, args []string) error {
			_, s := fromContext(cmd)
			hks := cmd.Context().Value(HooksCtxKey).(hooks.Hooks)
			repoName := getRepoName(s)
			hks.Update(s, s.Stderr(), repoName, hooks.HookArg{
				RefName: args[0],
				OldSha:  args[1],
				NewSha:  args[2],
			})
			return nil
		},
	}

	postReceiveCmd := &cobra.Command{
		Use:               "post-receive",
		Short:             "Run git post-receive hook",
		PersistentPreRunE: checkIfInternal,
		RunE: func(cmd *cobra.Command, _ []string) error {
			_, s := fromContext(cmd)
			hks := cmd.Context().Value(HooksCtxKey).(hooks.Hooks)
			repoName := getRepoName(s)
			opts := make([]hooks.HookArg, 0)
			scanner := bufio.NewScanner(s)
			for scanner.Scan() {
				fields := strings.Fields(scanner.Text())
				if len(fields) != 3 {
					return fmt.Errorf("invalid post-receive hook input: %s", scanner.Text())
				}
				opts = append(opts, hooks.HookArg{
					OldSha:  fields[0],
					NewSha:  fields[1],
					RefName: fields[2],
				})
			}
			hks.PostReceive(s, s.Stderr(), repoName, opts)
			return nil
		},
	}

	postUpdateCmd := &cobra.Command{
		Use:               "post-update",
		Short:             "Run git post-update hook",
		PersistentPreRunE: checkIfInternal,
		RunE: func(cmd *cobra.Command, args []string) error {
			_, s := fromContext(cmd)
			hks := cmd.Context().Value(HooksCtxKey).(hooks.Hooks)
			repoName := getRepoName(s)
			hks.PostUpdate(s, s.Stderr(), repoName, args...)
			return nil
		},
	}

	hookCmd := &cobra.Command{
		Use:          "hook",
		Short:        "Run git server hooks",
		Hidden:       true,
		SilenceUsage: true,
	}

	hookCmd.AddCommand(
		preReceiveCmd,
		updateCmd,
		postReceiveCmd,
		postUpdateCmd,
	)

	return hookCmd
}

// Check if the session's public key matches the internal API key.
func checkIfInternal(cmd *cobra.Command, _ []string) error {
	cfg, s := fromContext(cmd)
	pk := s.PublicKey()
	kp, err := keygen.New(cfg.SSH.InternalKeyPath, keygen.WithKeyType(keygen.Ed25519))
	if err != nil {
		logger.Errorf("failed to read internal key: %v", err)
		return err
	}
	if !ssh.KeysEqual(pk, kp.PublicKey()) {
		return ErrUnauthorized
	}
	return nil
}

func getRepoName(s ssh.Session) string {
	var repoName string
	for _, env := range s.Environ() {
		if strings.HasPrefix(env, "SOFT_SERVE_REPO_NAME=") {
			return strings.TrimPrefix(env, "SOFT_SERVE_REPO_NAME=")
		}
	}
	return repoName
}
