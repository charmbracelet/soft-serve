package ssh

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/charmbracelet/log"
	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
)

// ShellMiddleware is a middleware for the SSH shell.
func ShellMiddleware(sh ssh.Handler) ssh.Handler {
	softBin, err := os.Executable()
	if err != nil {
		// TODO: handle this better
		panic(err)
	}

	return func(s ssh.Session) {
		ctx := s.Context()
		logger := log.FromContext(ctx).WithPrefix("ssh")

		args := s.Command()
		ppty, winch, isInteractive := s.Pty()

		envs := s.Environ()
		if len(args) > 0 {
			envs = append(envs, "SSH_ORIGINAL_COMMAND="+strings.Join(args, " "))
		}

		var cmd interface {
			Run() error
		}
		cmdArgs := []string{"shell", "-c", fmt.Sprintf("'%s'", strings.Join(args, " "))}
		if isInteractive && ppty.Pty != nil {
			ppty.Pty.Resize(ppty.Window.Width, ppty.Window.Height)
			go func() {
				for win := range winch {
					log.Printf("resizing to %d x %d", win.Width, win.Height)
					ppty.Pty.Resize(win.Width, win.Height)
				}
			}()

			c := ppty.Pty.CommandContext(ctx, softBin, cmdArgs...)
			c.Env = append(envs, "PATH="+os.Getenv("PATH"))
			cmd = c
		} else {
			c := exec.CommandContext(ctx, softBin, cmdArgs...)
			c.Env = append(envs, "PATH="+os.Getenv("PATH"))
			cmd = c
		}

		if err := cmd.Run(); err != nil {
			logger.Errorf("error running command: %s", err)
			wish.Fatal(s, "internal server error")
			return
		}

		sh(s)
	}
}
