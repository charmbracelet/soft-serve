package shell

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"

	"github.com/anmitsu/go-shlex"
	"github.com/charmbracelet/soft-serve/cmd"
	"github.com/charmbracelet/soft-serve/pkg/shell"
	sshcmd "github.com/charmbracelet/soft-serve/pkg/ssh/cmd"
	"github.com/charmbracelet/soft-serve/pkg/sshutils"
	"github.com/charmbracelet/ssh"
	"github.com/mattn/go-tty"
	"github.com/muesli/termenv"
	"github.com/spf13/cobra"
)

var (
	commandString string

	// Command is a login shell command.
	Command = &cobra.Command{
		Use:                "shell",
		SilenceUsage:       true,
		PersistentPreRunE:  cmd.InitBackendContext,
		PersistentPostRunE: cmd.CloseDBContext,
		Args:               cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			args, err := shlex.Split(commandString, true)
			if err != nil {
				return err
			}

			return runShell(cmd, args)
		},
	}
)

func init() {
	Command.CompletionOptions.DisableDefaultCmd = true
	Command.SetUsageTemplate(sshcmd.UsageTemplate)
	Command.SetUsageFunc(sshcmd.UsageFunc)
	Command.Flags().StringVarP(&commandString, "", "c", "", "Command to run")
}

func runShell(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	sshTty, isInteractive := os.LookupEnv("SSH_TTY")
	sshUserAuth := os.Getenv("SSH_USER_AUTH")

	var ak string
	if sshUserAuth != "" {
		f, err := os.Open(sshUserAuth)
		if err != nil {
			return fmt.Errorf("could not open SSH_USER_AUTH file: %w", err)
		}

		ak = parseSSHUserAuth(f)
		f.Close() // nolint: errcheck
	}

	if err := os.Setenv("SOFT_SERVE_PUBLIC_KEY", ak); err != nil {
		return fmt.Errorf("could not set SOFT_SERVE_PUBLIC_KEY: %w", err)
	}

	var pk ssh.PublicKey
	var err error
	if ak != "" {
		pk, _, err = sshutils.ParseAuthorizedKey(ak)
		if err != nil {
			return fmt.Errorf("could not parse authorized key: %w", err)
		}
	}

	// We need a public key in the context even if it's nil
	ctx = context.WithValue(ctx, ssh.ContextKeyPublicKey, pk)

	in, out, er := os.Stdin, os.Stdout, os.Stderr
	if isInteractive {
		switch runtime.GOOS {
		case "windows":
			tty, err := tty.Open()
			if err != nil {
				return fmt.Errorf("could not open tty: %w", err)
			}

			in = tty.Input()
			out = tty.Output()
			er = tty.Output()
		default:
			var err error
			in, err = os.Open(sshTty)
			if err != nil {
				return fmt.Errorf("could not open input tty: %w", err)
			}

			out, err = os.OpenFile(sshTty, os.O_WRONLY, 0)
			if err != nil {
				return fmt.Errorf("could not open output tty: %w", err)
			}
			er = out
		}
	}

	c := shell.Command(ctx, osEnv, isInteractive)

	c.SetArgs(args)
	c.SetIn(in)
	c.SetOut(out)
	c.SetErr(er)
	c.SetContext(ctx)

	return c.ExecuteContext(ctx)
}

func parseSSHUserAuth(r io.Reader) string {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "publickey ") {
			return strings.TrimPrefix(line, "publickey ")
		}
	}

	return ""
}

var osEnv = &osEnviron{}

type osEnviron struct{}

var _ termenv.Environ = &osEnviron{}

// Environ implements termenv.Environ.
func (*osEnviron) Environ() []string {
	return os.Environ()
}

// Getenv implements termenv.Environ.
func (*osEnviron) Getenv(key string) string {
	return os.Getenv(key)
}
