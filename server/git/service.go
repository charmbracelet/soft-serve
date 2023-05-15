package git

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/charmbracelet/log"
	"golang.org/x/sync/errgroup"
)

// Service is a Git daemon service.
type Service string

const (
	// UploadPackService is the upload-pack service.
	UploadPackService Service = "git-upload-pack"
	// UploadArchiveService is the upload-archive service.
	UploadArchiveService Service = "git-upload-archive"
	// ReceivePackService is the receive-pack service.
	ReceivePackService Service = "git-receive-pack"
)

// String returns the string representation of the service.
func (s Service) String() string {
	return string(s)
}

// Name returns the name of the service.
func (s Service) Name() string {
	return strings.TrimPrefix(s.String(), "git-")
}

// Handler is the service handler.
func (s Service) Handler(ctx context.Context, cmd ServiceCommand) error {
	switch s {
	case UploadPackService, UploadArchiveService, ReceivePackService:
		return gitServiceHandler(ctx, s, cmd)
	default:
		return fmt.Errorf("unsupported service: %s", s)
	}
}

// ServiceHandler is a git service command handler.
type ServiceHandler func(ctx context.Context, cmd ServiceCommand) error

// gitServiceHandler is the default service handler using the git binary.
func gitServiceHandler(ctx context.Context, svc Service, scmd ServiceCommand) error {
	cmd := exec.CommandContext(ctx, "git", "-c", "uploadpack.allowFilter=true", svc.Name()) // nolint: gosec
	cmd.Dir = scmd.Dir
	if len(scmd.Args) > 0 {
		cmd.Args = append(cmd.Args, scmd.Args...)
	}

	cmd.Args = append(cmd.Args, ".")

	cmd.Env = os.Environ()
	if len(scmd.Env) > 0 {
		cmd.Env = append(cmd.Env, scmd.Env...)
	}

	if scmd.CmdFunc != nil {
		scmd.CmdFunc(cmd)
	}

	var (
		err    error
		stdin  io.WriteCloser
		stdout io.ReadCloser
		stderr io.ReadCloser
	)

	if scmd.Stdin != nil {
		stdin, err = cmd.StdinPipe()
		if err != nil {
			return err
		}
	}

	if scmd.Stdout != nil {
		stdout, err = cmd.StdoutPipe()
		if err != nil {
			return err
		}
	}

	if scmd.Stderr != nil {
		stderr, err = cmd.StderrPipe()
		if err != nil {
			return err
		}
	}

	log.Debugf("git service command in %q: %s", cmd.Dir, cmd.String())
	if err := cmd.Start(); err != nil {
		return err
	}

	errg, ctx := errgroup.WithContext(ctx)

	// stdin
	if scmd.Stdin != nil {
		errg.Go(func() error {
			if scmd.StdinHandler != nil {
				return scmd.StdinHandler(scmd.Stdin, stdin)
			} else {
				return defaultStdinHandler(scmd.Stdin, stdin)
			}
		})
	}

	// stdout
	if scmd.Stdout != nil {
		errg.Go(func() error {
			if scmd.StdoutHandler != nil {
				return scmd.StdoutHandler(scmd.Stdout, stdout)
			} else {
				return defaultStdoutHandler(scmd.Stdout, stdout)
			}
		})
	}

	// stderr
	if scmd.Stderr != nil {
		errg.Go(func() error {
			if scmd.StderrHandler != nil {
				return scmd.StderrHandler(scmd.Stderr, stderr)
			} else {
				return defaultStderrHandler(scmd.Stderr, stderr)
			}
		})
	}

	return errors.Join(errg.Wait(), cmd.Wait())
}

// ServiceCommand is used to run a git service command.
type ServiceCommand struct {
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
	Dir    string
	Env    []string
	Args   []string

	// Modifier functions
	CmdFunc       func(*exec.Cmd)
	StdinHandler  func(io.Reader, io.WriteCloser) error
	StdoutHandler func(io.Writer, io.ReadCloser) error
	StderrHandler func(io.Writer, io.ReadCloser) error
}

func defaultStdinHandler(in io.Reader, stdin io.WriteCloser) error {
	defer stdin.Close() // nolint: errcheck
	_, err := io.Copy(stdin, in)
	return err
}

func defaultStdoutHandler(out io.Writer, stdout io.ReadCloser) error {
	_, err := io.Copy(out, stdout)
	return err
}

func defaultStderrHandler(err io.Writer, stderr io.ReadCloser) error {
	_, erro := io.Copy(err, stderr)
	return erro
}

// UploadPack runs the git upload-pack protocol against the provided repo.
func UploadPack(ctx context.Context, cmd ServiceCommand) error {
	return gitServiceHandler(ctx, UploadPackService, cmd)
}

// UploadArchive runs the git upload-archive protocol against the provided repo.
func UploadArchive(ctx context.Context, cmd ServiceCommand) error {
	return gitServiceHandler(ctx, UploadArchiveService, cmd)
}

// ReceivePack runs the git receive-pack protocol against the provided repo.
func ReceivePack(ctx context.Context, cmd ServiceCommand) error {
	return gitServiceHandler(ctx, ReceivePackService, cmd)
}
