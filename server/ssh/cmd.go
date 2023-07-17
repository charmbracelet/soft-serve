package ssh

import (
	"github.com/charmbracelet/log"
	"github.com/charmbracelet/soft-serve/server/ssh/cmd"
	"github.com/charmbracelet/ssh"
)

func handleCli(s ssh.Session) {
	ctx := s.Context()
	logger := log.FromContext(ctx)
	rootCmd := cmd.RootCommand(s)
	if err := rootCmd.ExecuteContext(ctx); err != nil {
		logger.Error("error executing command", "err", err)
		_ = s.Exit(1)
	}
}
