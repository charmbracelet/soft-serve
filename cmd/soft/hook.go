package main

import (
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
		Long:   "Handles Soft Serve git server hooks.",
		Hidden: true,
		RunE: func(_ *cobra.Command, args []string) error {
			c, s, err := commonInit()
			if err != nil {
				return err
			}
			defer c.Close() //nolint:errcheck
			defer s.Close() //nolint:errcheck
			s.Stdin = os.Stdin
			s.Stdout = os.Stdout
			s.Stderr = os.Stderr
			cmd := fmt.Sprintf("hook %s", strings.Join(args, " "))
			if err := s.Run(cmd); err != nil {
				return err
			}
			return nil
		},
	}
)

func init() {
	hookCmd.PersistentFlags().StringVarP(&configPath, "config", "c", "", "path to config file")
}

// TODO: use ssh controlmaster
func commonInit() (c *gossh.Client, s *gossh.Session, err error) {
	cfg, err := config.ParseConfig(configPath)
	if err != nil {
		return
	}

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
	repoName = strings.TrimPrefix(repoName, string(os.PathSeparator))
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
	pk, err := keygen.New(cfg.Internal.KeyPath, keygen.WithKeyType(keygen.Ed25519))
	if err != nil {
		return nil, err
	}
	ik, err := keygen.New(cfg.Internal.InternalKeyPath, keygen.WithKeyType(keygen.Ed25519))
	if err != nil {
		return nil, err
	}
	cc := &gossh.ClientConfig{
		User: "internal",
		Auth: []gossh.AuthMethod{
			gossh.PublicKeys(ik.Signer()),
		},
		HostKeyCallback: gossh.FixedHostKey(pk.PublicKey()),
	}
	c, err := gossh.Dial("tcp", cfg.Internal.ListenAddr, cc)
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
