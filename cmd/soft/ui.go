package main

import (
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/log"
	"github.com/charmbracelet/soft-serve/server/access"
	"github.com/charmbracelet/soft-serve/server/backend"
	"github.com/charmbracelet/soft-serve/server/config"
	"github.com/charmbracelet/soft-serve/server/errors"
	"github.com/charmbracelet/soft-serve/server/ui"
	"github.com/charmbracelet/soft-serve/server/ui/common"
	"github.com/mattn/go-tty"
	"github.com/muesli/termenv"
	"github.com/spf13/cobra"
)

var uiCmd = &cobra.Command{
	Use: "ui",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		be := backend.FromContext(ctx)
		cfg := config.FromContext(ctx)

		logPath := filepath.Join(cfg.DataPath, "log", "ui.log")
		f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return err
		}

		defer f.Close() // nolint:errcheck

		logger := log.FromContext(ctx).WithPrefix("ui")
		logger.SetOutput(f)
		ctx = log.WithContext(ctx, logger)

		lipgloss.SetColorProfile(termenv.Ascii)

		logger.Infof("SSH_TTY %s", os.Getenv("SSH_TTY"))
		tty, err := tty.OpenDevice(os.Getenv("SSH_TTY"))
		// tty, err := tty.Open()
		if err != nil {
			return err
		}

		// stdin := tty.Input()
		// stdout := cmd.OutOrStdout()
		stdin := tty.Input()
		stdout := tty.Output()
		// if tty := os.Getenv("SSH_TTY"); tty != "" {
		// 	logger.Infof("SSH_TTY %s", tty)
		// 	f, err := os.OpenFile(tty, os.O_RDWR, 0)
		// 	if err != nil {
		// 		return err
		// 	}
		// 	defer f.Close() // nolint:errcheck
		//
		// 	logger.Infof("using %s", tty)
		// 	stdin = f
		// 	stdout = f
		// }

		// if !isatty.IsTerminal(stdout.(interface {
		// 	Fd() uintptr
		// }).Fd()) {
		// 	return fmt.Errorf("stdout is not a terminal")
		// }

		var initialRepo string
		if len(args) == 1 {
			user, _ := be.Authenticate(ctx, nil)
			initialRepo = args[0]
			auth, _ := be.AccessLevel(ctx, initialRepo, user)
			if auth < access.ReadOnlyAccess {
				return errors.ErrUnauthorized
			}
		}

		re := lipgloss.NewRenderer(stdout, termenv.WithColorCache(true), termenv.WithUnsafe(), termenv.WithTTY(true))
		// FIXME: detect color profile and dark background
		re.SetColorProfile(termenv.Ascii)
		re.SetHasDarkBackground(false)
		termenv.SetDefaultOutput(re.Output())
		c := common.NewCommon(ctx, re, 0, 0)
		m := ui.New(c, initialRepo)
		p := tea.NewProgram(m,
			tea.WithInput(stdin),
			tea.WithOutput(re.Output()),
			tea.WithAltScreen(),
			tea.WithoutCatchPanics(),
			tea.WithMouseCellMotion(),
		)

		if _, err := p.Run(); err != nil {
			return err
		}

		return nil
	},
}
