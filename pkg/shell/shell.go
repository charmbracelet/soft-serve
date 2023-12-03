package shell

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/log"
	"github.com/charmbracelet/soft-serve/pkg/config"
	"github.com/charmbracelet/soft-serve/pkg/ssh/cmd"
	"github.com/charmbracelet/soft-serve/pkg/sshutils"
	"github.com/charmbracelet/soft-serve/pkg/ui/common"
	"github.com/charmbracelet/ssh"
	"github.com/muesli/termenv"
	"github.com/spf13/cobra"
)

// Command returns a new shell command.
func Command(ctx context.Context, env termenv.Environ, isInteractive bool) *cobra.Command {
	cfg := config.FromContext(ctx)
	c := &cobra.Command{
		Short:            "Soft Serve is a self-hostable Git server for the command line.",
		SilenceUsage:     true,
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			in := cmd.InOrStdin()
			out := cmd.OutOrStdout()
			if isInteractive && len(args) == 0 {
				// Run UI
				output := termenv.NewOutput(out, termenv.WithColorCache(true), termenv.WithEnvironment(env))
				c := common.NewCommon(ctx, output, 0, 0)
				c.SetValue(common.ConfigKey, cfg)
				m := NewUI(c, "")
				p := tea.NewProgram(m,
					tea.WithInput(in),
					tea.WithOutput(out),
					tea.WithAltScreen(),
					tea.WithoutCatchPanics(),
					tea.WithMouseCellMotion(),
					tea.WithContext(ctx),
				)

				return startProgram(cmd.Context(), p)
			} else if len(args) > 0 {
				// Run custom command
				return startCommand(cmd, args)
			}

			return fmt.Errorf("invalid command %v", args)
		},
	}
	c.CompletionOptions.DisableDefaultCmd = true

	c.SetUsageTemplate(cmd.UsageTemplate)
	c.SetUsageFunc(cmd.UsageFunc)
	c.AddCommand(
		cmd.GitUploadPackCommand(),
		cmd.GitUploadArchiveCommand(),
		cmd.GitReceivePackCommand(),
		// TODO: write shell commands for these
		// cmd.RepoCommand(),
		// cmd.SettingsCommand(),
		// cmd.UserCommand(),
		// cmd.InfoCommand(),
		// cmd.PubkeyCommand(),
		// cmd.SetUsernameCommand(),
		// cmd.JWTCommand(),
		// cmd.TokenCommand(),
	)

	if cfg.LFS.Enabled {
		c.AddCommand(
			cmd.GitLFSAuthenticateCommand(),
		)

		if cfg.LFS.SSHEnabled {
			c.AddCommand(
				cmd.GitLFSTransfer(),
			)
		}
	}

	c.SetContext(ctx)

	return c
}

func startProgram(ctx context.Context, p *tea.Program) (err error) {
	var windowChanges <-chan ssh.Window
	if s := sshutils.SessionFromContext(ctx); s != nil {
		_, windowChanges, _ = s.Pty()
	}
	ctx, cancel := context.WithCancel(ctx)
	go func() {
		for {
			select {
			case <-ctx.Done():
				if p != nil {
					p.Quit()
					return
				}
			case w := <-windowChanges:
				if p != nil {
					p.Send(tea.WindowSizeMsg{Width: w.Width, Height: w.Height})
				}
			}
		}
	}()

	_, err = p.Run()

	// p.Kill() will force kill the program if it's still running,
	// and restore the terminal to its original state in case of a
	// tui crash
	p.Kill()
	cancel()

	return
}

func startCommand(co *cobra.Command, args []string) error {
	ctx := co.Context()
	cfg := config.FromContext(ctx)
	if len(args) == 0 {
		return fmt.Errorf("no command specified")
	}

	cmdsDir := filepath.Join(cfg.DataPath, "commands")

	var cmdArgs []string
	if len(args) > 1 {
		cmdArgs = args[1:]
	}

	cmdPath := filepath.Join(cmdsDir, args[0])

	// if stat, err := os.Stat(cmdPath); errors.Is(err, fs.ErrNotExist) || stat.Mode()&0111 == 0 {
	// 	log.Printf("command mode %s", stat.Mode().String())
	// 	return fmt.Errorf("command not found: %s", args[0])
	// }

	cmdPath, err := filepath.Abs(cmdPath)
	if err != nil {
		return fmt.Errorf("could not get absolute path for command: %w", err)
	}

	cmd := exec.CommandContext(ctx, cmdPath, cmdArgs...)

	cmd.Dir = cmdsDir
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("could not get stdin pipe: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("could not get stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("could not get stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("could not start command: %w", err)
	}

	go io.Copy(stdin, co.InOrStdin())    // nolint: errcheck
	go io.Copy(co.OutOrStdout(), stdout) // nolint: errcheck
	go io.Copy(co.ErrOrStderr(), stderr) // nolint: errcheck

	log.Infof("waiting for command to finish: %s", cmdPath)
	if err := cmd.Wait(); err != nil {
		return err
	}

	return nil
}
