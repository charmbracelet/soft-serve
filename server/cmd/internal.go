package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/soft-serve/internal/config"
	"github.com/gliderlabs/ssh"
	"github.com/spf13/cobra"
	gossh "golang.org/x/crypto/ssh"
)

// InternalCommand handles Soft Serve internal API requests.
func InternalCommand() *cobra.Command {
	preReceiveCmd := &cobra.Command{
		Use:   "pre-receive",
		Short: "Run git pre-receive hook",
		RunE: func(cmd *cobra.Command, args []string) error {
			ac, s := fromContext(cmd)
			repoName := getRepoName(s)
			opts := make([]config.HookOption, 0)
			scanner := bufio.NewScanner(s)
			for scanner.Scan() {
				fields := strings.Fields(scanner.Text())
				if len(fields) != 3 {
					return fmt.Errorf("invalid pre-receive hook input: %s", scanner.Text())
				}
				opts = append(opts, config.HookOption{
					OldSha:  fields[0],
					NewSha:  fields[1],
					RefName: fields[2],
				})
			}
			ac.PreReceive(repoName, opts)
			return nil
		},
	}

	updateCmd := &cobra.Command{
		Use:   "update",
		Short: "Run git update hook",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			ac, s := fromContext(cmd)
			repoName := getRepoName(s)
			ac.Update(repoName, config.HookOption{
				RefName: args[0],
				OldSha:  args[1],
				NewSha:  args[2],
			})
			return nil
		},
	}

	postReceiveCmd := &cobra.Command{
		Use:   "post-receive",
		Short: "Run git post-receive hook",
		RunE: func(cmd *cobra.Command, args []string) error {
			ac, s := fromContext(cmd)
			repoName := getRepoName(s)
			opts := make([]config.HookOption, 0)
			scanner := bufio.NewScanner(s)
			for scanner.Scan() {
				fields := strings.Fields(scanner.Text())
				if len(fields) != 3 {
					return fmt.Errorf("invalid post-receive hook input: %s", scanner.Text())
				}
				opts = append(opts, config.HookOption{
					OldSha:  fields[0],
					NewSha:  fields[1],
					RefName: fields[2],
				})
			}
			ac.PostReceive(repoName, opts)
			return nil
		},
	}

	hookCmd := &cobra.Command{
		Use:   "hook",
		Short: "Run git server hooks",
	}

	hookCmd.AddCommand(
		preReceiveCmd,
		updateCmd,
		postReceiveCmd,
	)

	// Check if the session's public key matches the internal API key.
	authorized := func(cmd *cobra.Command) (bool, error) {
		ac, s := fromContext(cmd)
		pk := s.PublicKey()
		kp := ac.Cfg.InternalKeyPath
		pemKey, err := os.ReadFile(kp)
		if err != nil {
			return false, err
		}
		priv, err := gossh.ParsePrivateKey(pemKey)
		if err != nil {
			return false, err
		}
		if !ssh.KeysEqual(pk, priv.PublicKey()) {
			return false, ErrUnauthorized
		}
		return true, nil
	}
	internalCmd := &cobra.Command{
		Use:          "internal",
		Short:        "Internal Soft Serve API",
		Hidden:       true,
		SilenceUsage: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			cmd.SilenceUsage = false
			authd, err := authorized(cmd)
			if err != nil {
				cmd.SilenceUsage = true
				return err
			}
			if !authd {
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			authd, err := authorized(cmd)
			if err != nil {
				return err
			}
			if !authd {
				return ErrUnauthorized
			}
			return cmd.Help()
		},
	}

	internalCmd.AddCommand(
		hookCmd,
	)

	return internalCmd
}

func getRepoName(s ssh.Session) string {
	var repoName string
	for _, env := range s.Environ() {
		if strings.HasPrefix(env, "SOFT_SERVE_REPO_NAME=") {
			repoName = strings.TrimPrefix(env, "SOFT_SERVE_REPO_NAME=")
			break
		}
	}
	return repoName
}
