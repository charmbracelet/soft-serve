package ssh

import (
	"fmt"
	"io"

	"github.com/charmbracelet/log"
	"github.com/charmbracelet/soft-serve/pkg/shell"
	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
)

// ShellMiddleware is a middleware for the SSH shell.
func ShellMiddleware(sh ssh.Handler) ssh.Handler {
	return func(s ssh.Session) {
		ctx := s.Context()
		logger := log.FromContext(ctx).WithPrefix("ssh")
		envs := &sessionEnv{s}

		ppty, _, isInteractive := s.Pty()

		var (
			in  io.Reader = s
			out io.Writer = s
			er  io.Writer = s.Stderr()
			err error
		)

		if isInteractive {
			in, out, er, err = ptyNew(ppty.Pty)
			if err != nil {
				logger.Errorf("could not create pty: %v", err)
				// TODO: replace this err with a declared error
				wish.Fatalln(s, fmt.Errorf("internal server error"))
				return
			}
		}

		args := s.Command()
		if len(args) == 0 {
			// XXX: args cannot be nil, otherwise cobra will use os.Args[1:]
			args = []string{}
		}

		cmd := shell.Command(ctx, envs, isInteractive)
		cmd.SetArgs(args)
		cmd.SetIn(in)
		cmd.SetOut(out)
		cmd.SetErr(er)
		cmd.SetContext(ctx)

		if err := cmd.ExecuteContext(ctx); err != nil {
			wish.Fatalln(s, err)
			return
		}

		sh(s)
	}
}
