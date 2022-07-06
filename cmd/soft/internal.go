package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/soft-serve/config"
	"github.com/spf13/cobra"
	gossh "golang.org/x/crypto/ssh"
)

var (
	internalCmd = &cobra.Command{
		Use:   "internal",
		Short: "Internal Soft Serve API",
		Long: `Soft Serve internal API.
This command is used to communicate with the Soft Serve SSH server.`,
		Hidden: true,
	}

	hookCmd = &cobra.Command{
		Use:   "hook",
		Short: "Run git server hooks",
		Long:  "Handles git server hooks. This includes pre-receive, update, and post-receive.",
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
			b, err := s.Output("internal hook pre-receive")
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
			b, err := s.Output(fmt.Sprintf("internal hook update %s %s %s", refName, oldSha, newSha))
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
			b, err := s.Output("internal hook post-receive")
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
	)
	internalCmd.AddCommand(
		hookCmd,
	)
}

func commonInit() (c *gossh.Client, s *gossh.Session, err error) {
	cfg := config.DefaultConfig()
	// Git runs the hook within the repository's directory.
	// Get the working directory to determine the repository name.
	wd, err := os.Getwd()
	if err != nil {
		return
	}
	if !strings.HasPrefix(wd, cfg.RepoPath) {
		err = fmt.Errorf("hook must be run from within repository directory")
		return
	}
	repoName := strings.TrimPrefix(wd, cfg.RepoPath)
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
	pubKey, err := os.ReadFile(cfg.KeyPath)
	if err != nil {
		return nil, err
	}
	hostKey, err := gossh.ParsePrivateKey(pubKey)
	if err != nil {
		return nil, err
	}
	pemKey, err := os.ReadFile(cfg.InternalKeyPath)
	if err != nil {
		return nil, err
	}
	k, err := gossh.ParsePrivateKey(pemKey)
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
	addr := fmt.Sprintf("%s:%d", cfg.BindAddr, cfg.Port)
	c, err := gossh.Dial("tcp", addr, cc)
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
