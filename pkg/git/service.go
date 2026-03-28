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
	"time"

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
//
// Deadline invariant: the ctx passed here must be cancellation-only — it must
// NOT carry a context.WithDeadline or context.WithTimeout. All three transport
// callers (SSH via gliderlabs/ssh, git daemon via pkg/daemon, HTTP via
// net/http) enforce timeouts by calling conn.SetDeadline on the underlying
// net.Conn, NOT by attaching a deadline to the context. If a caller were ever
// changed to pass a deadline-carrying context, exec.CommandContext would kill
// the git subprocess when the deadline expires, potentially corrupting an
// in-progress push. In that case, replace ctx with a context.WithCancelCause
// derived from context.Background() that mirrors parent cancellation but not
// DeadlineExceeded.
func gitServiceHandler(ctx context.Context, svc Service, scmd ServiceCommand) error {
	// NOTE: ctx is cancellation-only (derived from context.Background via
	// context.WithCancel). SSH and git-daemon timeouts fire as net.Conn
	// deadline errors, not context deadlines — so no deadline can reach here
	// and kill a long-running push or clone mid-transfer.
	cmd := exec.CommandContext(ctx, "git")
	// WaitDelay bounds how long we wait for stdin/stdout pipe goroutines to
	// finish after the git process has already been killed (e.g. by context
	// cancellation). It does NOT impose a timeout on running git operations.
	cmd.WaitDelay = 30 * time.Second
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
			if _, err := io.Copy(stdin, scmd.Stdin); err != nil && ctx.Err() == nil {
				// Broken-pipe here is normal: git read all it needed and exited,
				// cmd.Wait() closed the write end of the pipe before the client
				// finished sending. Log at Debug to avoid spurious error noise.
				log.FromContext(ctx).Debug("gitServiceHandler: stdin copy ended", "err", err)
			}
		}()
	}

	// stdout
	if scmd.Stdout != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := io.Copy(scmd.Stdout, stdout); err != nil && ctx.Err() == nil {
				log.Errorf("gitServiceHandler: failed to copy stdout: %v", err)
			}
		}()
	}

	// stderr
	if scmd.Stderr != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, erro := io.Copy(scmd.Stderr, stderr); erro != nil && ctx.Err() == nil {
				log.Errorf("gitServiceHandler: failed to copy stderr: %v", erro)
			}
		}()
	}

	// Ensure all the output is written before waiting for the command to
	// finish.
	// Stdin is handled by the client side.
	wg.Wait()

	err = cmd.Wait()
	if err != nil {
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
			// When WaitDelay fires alongside a non-zero exit, cmd.Wait returns
			// errors.Join(exitErr, exec.ErrWaitDelay). Strip the WaitDelay noise
			// so callers see a clean exit-status error, not an internal timeout.
			retErr := error(exitErr)
			if len(exitErr.Stderr) > 0 {
				retErr = fmt.Errorf("%w: %s", exitErr, exitErr.Stderr)
			}
			return retErr
		} else if errors.Is(err, exec.ErrWaitDelay) {
			// WaitDelay (30s) expired before pipe goroutines drained. This branch
			// is only reached when there is no accompanying ExitError (i.e. git
			// exited 0 but the client read the pipe slowly). The git command
			// succeeded; the drain timeout is an internal bookkeeping detail, not
			// a caller-visible error.
			log.FromContext(ctx).Debug("git pipe drain timed out", "service", svc.Name())
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
