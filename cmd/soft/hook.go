package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/keygen"
	"github.com/charmbracelet/soft-serve/server/config"
	"github.com/spf13/cobra"
	gossh "golang.org/x/crypto/ssh"
)

var (
	configPath string

	hookCmd = &cobra.Command{
		Use:    "hook",
		Short:  "Run git server hooks",
		Long:   "Handles git server hooks. This includes pre-receive, update, and post-receive.",
		Hidden: true,
	}

	preReceiveCmd = &cobra.Command{
		Use:   "pre-receive",
		Short: "Run git pre-receive hook",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, s, err := commonInit()
			if err != nil {
				return err
			}
			defer c.Close() //nolint:errcheck
			defer s.Close() //nolint:errcheck
			in, err := s.StdinPipe()
			if err != nil {
				return err
			}
			scanner := bufio.NewScanner(os.Stdin)
			for scanner.Scan() {
				in.Write([]byte(scanner.Text()))
				in.Write([]byte("\n"))
			}
			in.Close() //nolint:errcheck
			b, err := s.Output("hook pre-receive")
			if err != nil {
				return err
			}
			cmd.Print(string(b))
			return nil
		},
	}

	updateCmd = &cobra.Command{
		Use:   "update",
		Short: "Run git update hook",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			refName := args[0]
			oldSha := args[1]
			newSha := args[2]
			c, s, err := commonInit()
			if err != nil {
				return err
			}
			defer c.Close() //nolint:errcheck
			defer s.Close() //nolint:errcheck
			b, err := s.Output(fmt.Sprintf("hook update %s %s %s", refName, oldSha, newSha))
			if err != nil {
				return err
			}
			cmd.Print(string(b))
			return nil
		},
	}

	postReceiveCmd = &cobra.Command{
		Use:   "post-receive",
		Short: "Run git post-receive hook",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, s, err := commonInit()
			if err != nil {
				return err
			}
			defer c.Close() //nolint:errcheck
			defer s.Close() //nolint:errcheck
			in, err := s.StdinPipe()
			if err != nil {
				return err
			}
			scanner := bufio.NewScanner(os.Stdin)
			for scanner.Scan() {
				in.Write([]byte(scanner.Text()))
				in.Write([]byte("\n"))
			}
			in.Close() //nolint:errcheck
			b, err := s.Output("hook post-receive")
			if err != nil {
				return err
			}
			cmd.Print(string(b))
			return nil
		},
	}

	postUpdateCmd = &cobra.Command{
		Use:   "post-update",
		Short: "Run git post-update hook",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, s, err := commonInit()
			if err != nil {
				return err
			}
			defer c.Close() //nolint:errcheck
			defer s.Close() //nolint:errcheck
			b, err := s.Output(fmt.Sprintf("hook post-update %s", strings.Join(args, " ")))
			if err != nil {
				return err
			}
			cmd.Print(string(b))
			return nil
		},
	}
)

func init() {
	hookCmd.AddCommand(
		preReceiveCmd,
		updateCmd,
		postReceiveCmd,
		postUpdateCmd,
	)

	hookCmd.PersistentFlags().StringVarP(&configPath, "config", "c", "", "path to config file")
}

func commonInit() (c *gossh.Client, s *gossh.Session, err error) {
	cfg, err := config.ParseConfig(configPath)
	if err != nil {
		return
	}

	// Use absolute path.
	cfg.DataPath = filepath.Dir(configPath)

	// Git runs the hook within the repository's directory.
	// Get the working directory to determine the repository name.
	wd, err := os.Getwd()
	if err != nil {
		return
	}

	rs, err := filepath.Abs(filepath.Join(cfg.DataPath, "repos"))
	if err != nil {
		return
	}

	if !strings.HasPrefix(wd, rs) {
		err = fmt.Errorf("hook must be run from within repository directory")
		return
	}
	repoName := strings.TrimPrefix(wd, rs)
	repoName = strings.TrimPrefix(repoName, fmt.Sprintf("%c", os.PathSeparator))
	c, err = newClient(cfg)
	if err != nil {
		return
	}
	s, err = newSession(c)
	if err != nil {
		return
	}
	s.Setenv("SOFT_SERVE_REPO_NAME", repoName)
	return
}

func newClient(cfg *config.Config) (*gossh.Client, error) {
	// Only accept the server's host key.
	pk, err := keygen.New(cfg.SSH.KeyPath, nil, keygen.Ed25519)
	if err != nil {
		return nil, err
	}
	hostKey, err := gossh.ParsePrivateKey(pk.PrivateKeyPEM())
	if err != nil {
		return nil, err
	}
	ik, err := keygen.New(cfg.SSH.InternalKeyPath, nil, keygen.Ed25519)
	if err != nil {
		return nil, err
	}
	k, err := gossh.ParsePrivateKey(ik.PrivateKeyPEM())
	if err != nil {
		return nil, err
	}
	cc := &gossh.ClientConfig{
		User: "internal",
		Auth: []gossh.AuthMethod{
			gossh.PublicKeys(k),
		},
		HostKeyCallback: gossh.FixedHostKey(hostKey.PublicKey()),
	}
	c, err := gossh.Dial("tcp", cfg.SSH.ListenAddr, cc)
	if err != nil {
		return nil, err
	}
	return c, nil
}

func newSession(c *gossh.Client) (*gossh.Session, error) {
	s, err := c.NewSession()
	if err != nil {
		return nil, err
	}
	return s, nil
}
