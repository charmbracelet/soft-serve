package git

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"

	"charm.land/log/v2"
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
	// LFSTransferService is the LFS transfer service.
	LFSTransferService Service = "git-lfs-transfer"
	// LFSAuthenticateService is the LFS authenticate service.
	LFSAuthenticateService = "git-lfs-authenticate"
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
	case LFSTransferService:
		return LFSTransfer(ctx, cmd)
	case LFSAuthenticateService:
		return LFSAuthenticate(ctx, cmd)
	default:
		return fmt.Errorf("unsupported service: %s", s)
	}
}

// ServiceHandler is a git service command handler.
type ServiceHandler func(ctx context.Context, cmd ServiceCommand) error

// gitServiceHandler is the default service handler using the git binary.
func gitServiceHandler(ctx context.Context, svc Service, scmd ServiceCommand) error {
	cmd := exec.CommandContext(ctx, "git")
	cmd.Dir = scmd.Dir
	cmd.Args = append(cmd.Args, []string{
		// Enable partial clones
		"-c", "uploadpack.allowFilter=true",
		// Enable push options
		"-c", "receive.advertisePushOptions=true",
		// Disable LFS filters
		"-c", "filter.lfs.required=", "-c", "filter.lfs.smudge=", "-c", "filter.lfs.clean=",
		svc.Name(),
	}...)
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

	if err := cmd.Start(); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return ErrInvalidRepo
		}
		return err
	}

	wg := &sync.WaitGroup{}

	// stdin
	if scmd.Stdin != nil {
		go func() {
			defer stdin.Close() //nolint: errcheck
			if _, err := io.Copy(stdin, scmd.Stdin); err != nil {
				log.Errorf("gitServiceHandler: failed to copy stdin: %v", err)
			}
		}()
	}

	// stdout
	if scmd.Stdout != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := io.Copy(scmd.Stdout, stdout); err != nil {
				log.Errorf("gitServiceHandler: failed to copy stdout: %v", err)
			}
		}()
	}

	// stderr
	if scmd.Stderr != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, erro := io.Copy(scmd.Stderr, stderr); erro != nil {
				log.Errorf("gitServiceHandler: failed to copy stderr: %v", erro)
			}
		}()
	}

	// Ensure all the output is written before waiting for the command to
	// finish.
	// Stdin is handled by the client side.
	wg.Wait()

	err = cmd.Wait()
	if err != nil && errors.Is(err, os.ErrNotExist) {
		return ErrInvalidRepo
	} else if err != nil {
		var exitErr *exec.ExitError
		// Note: errors.As correctly unwraps through errors.Join, which Go 1.20+
		// uses when cmd.WaitDelay fires (joining the ExitError with a timeout
		// error). Verified: errors.As(errors.Join(exitErr, timeoutErr), &exitErr)
		// returns true. So the suppression path below is safe even when WaitDelay
		// wraps the ExitError via errors.Join.
		if errors.As(err, &exitErr) {
			if errors.Is(ctx.Err(), context.Canceled) {
				// Process exited because context was cancelled — client disconnected
				// or server is shutting down. Expected; not worth surfacing.
				// We do not gate on ExitCode()==-1: on Unix a signal-killed process
				// has no exit code (-1), but on Windows TerminateProcess sets exit
				// code 1. Checking ctx.Err() alone is portable across both.
				log.FromContext(ctx).Debug("git process exited on context cancellation", "service", svc.Name())
				return nil
			}
			if len(exitErr.Stderr) > 0 {
				return fmt.Errorf("%w: %s", err, exitErr.Stderr)
			}
		} else if errors.Is(err, exec.ErrWaitDelay) && errors.Is(ctx.Err(), context.Canceled) {
			// WaitDelay expired after context cancellation — the process was already
			// killed; pipe goroutines did not drain within 30s. Not an error.
			log.FromContext(ctx).Debug("git pipe drain timed out after cancellation", "service", svc.Name())
			return nil
		}

		return err
	}

	return nil
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
	CmdFunc func(*exec.Cmd)
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
