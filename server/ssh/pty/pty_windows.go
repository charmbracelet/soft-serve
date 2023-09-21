//go:build windows
// +build windows

package pty

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"syscall"
	"unicode/utf16"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	_PROC_THREAD_ATTRIBUTE_PSEUDOCONSOLE = 0x20016 // nolint:revive
)

var (
	errClosedConPty = errors.New("pseudo console is closed")
	errNotStarted   = errors.New("process not started")
)

// Install this from github.com/Microsoft/go-winio
// go install github.com/Microsoft/go-winio/tools/mkwinsyscall@latest
//go:generate mkwinsyscall -output zsyscall_windows.go ./*.go

// https://github.com/microsoft/hcsshim/blob/main/internal/conpty/conpty.go
type conPty struct {
	handle          windows.Handle
	inPipe, outPipe *os.File
	mtx             sync.RWMutex
}

var _ Pty = &conPty{}

func newPty() (Pty, error) {
	ptyIn, inPipeOurs, err := os.Pipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create pipes for pseudo console: %w", err)
	}

	outPipeOurs, ptyOut, err := os.Pipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create pipes for pseudo console: %w", err)
	}

	var hpc windows.Handle
	coord := windows.Coord{X: 80, Y: 25}
	err = createPseudoConsole(coord, windows.Handle(ptyIn.Fd()), windows.Handle(ptyOut.Fd()), 0, &hpc)
	if err != nil {
		return nil, fmt.Errorf("failed to create pseudo console: %w", err)
	}

	if err := ptyOut.Close(); err != nil {
		return nil, fmt.Errorf("failed to close pseudo console handle: %w", err)
	}
	if err := ptyIn.Close(); err != nil {
		return nil, fmt.Errorf("failed to close pseudo console handle: %w", err)
	}

	return &conPty{
		handle:  hpc,
		inPipe:  inPipeOurs,
		outPipe: outPipeOurs,
	}, nil
}

// Close implements Pty.
func (p *conPty) Close() error {
	p.mtx.Lock()
	defer p.mtx.Unlock()

	closePseudoConsole(p.handle)
	return errors.Join(p.inPipe.Close(), p.outPipe.Close())
}

// Command implements Pty.
func (p *conPty) Command(name string, args ...string) *Cmd {
	return p.CommandContext(nil, name, args...)
}

// CommandContext implements Pty.
func (p *conPty) CommandContext(ctx context.Context, name string, args ...string) *Cmd {
	return &Cmd{
		ctx:  ctx,
		pty:  p,
		Path: name,
		Args: append([]string{name}, args...),
	}
}

// Control implements Pty.
func (*conPty) Control(func(fd uintptr)) error {
	return nil
}

// Name implements Pty.
func (*conPty) Name() string {
	return "windows-pty"
}

// Read implements Pty.
func (p *conPty) Read(b []byte) (n int, err error) {
	return p.outPipe.Read(b)
}

// Resize implements Pty.
func (p *conPty) Resize(rows int, cols int) error {
	if err := resizePseudoConsole(p.handle, windows.Coord{X: int16(cols), Y: int16(rows)}); err != nil {
		return fmt.Errorf("failed to resize pseudo console: %w", err)
	}
	return nil
}

// Write implements Pty.
func (p *conPty) Write(b []byte) (n int, err error) {
	return p.inPipe.Write(b)
}

// updateProcThreadAttribute updates the passed in attribute list to contain the entry necessary for use with
// CreateProcess.
func (p *conPty) updateProcThreadAttribute(attrList *windows.ProcThreadAttributeListContainer) error {
	p.mtx.RLock()
	defer p.mtx.RUnlock()

	if p.handle == 0 {
		return errClosedConPty
	}

	if err := attrList.Update(
		_PROC_THREAD_ATTRIBUTE_PSEUDOCONSOLE,
		unsafe.Pointer(p.handle),
		unsafe.Sizeof(p.handle),
	); err != nil {
		return fmt.Errorf("failed to update proc thread attributes for pseudo console: %w", err)
	}

	return nil
}

// createPseudoConsole creates a windows pseudo console.
func createPseudoConsole(size windows.Coord, hInput windows.Handle, hOutput windows.Handle, dwFlags uint32, hpcon *windows.Handle) error {
	// We need this wrapper as the function takes a COORD struct and not a pointer to one, so we need to cast to something beforehand.
	return _createPseudoConsole(*((*uint32)(unsafe.Pointer(&size))), hInput, hOutput, dwFlags, hpcon)
}

// resizePseudoConsole resizes the internal buffers of the pseudo console to the width and height specified in `size`.
func resizePseudoConsole(hpcon windows.Handle, size windows.Coord) error {
	// We need this wrapper as the function takes a COORD struct and not a pointer to one, so we need to cast to something beforehand.
	return _resizePseudoConsole(hpcon, *((*uint32)(unsafe.Pointer(&size))))
}

//sys _createPseudoConsole(size uint32, hInput windows.Handle, hOutput windows.Handle, dwFlags uint32, hpcon *windows.Handle) (hr error) = kernel32.CreatePseudoConsole
//sys _resizePseudoConsole(hPc windows.Handle, size uint32) (hr error) = kernel32.ResizePseudoConsole
//sys closePseudoConsole(hpc windows.Handle) = kernel32.ClosePseudoConsole

type conPtySys struct {
	attrs  *windows.ProcThreadAttributeListContainer
	done   chan error
	cmdErr error
}

func (c *Cmd) start() error {
	pty, ok := c.pty.(*conPty)
	if !ok {
		return ErrInvalidCommand
	}

	argv0 := c.Path
	if len(c.Dir) != 0 {
		// Windows CreateProcess looks for argv0 relative to the current
		// directory, and, only once the new process is started, it does
		// Chdir(attr.Dir). We are adjusting for that difference here by
		// making argv0 absolute.
		var err error
		argv0, err = joinExeDirAndFName(c.Dir, c.Path)
		if err != nil {
			return err
		}
	}

	argv0p, err := windows.UTF16PtrFromString(argv0)
	if err != nil {
		return err
	}

	var cmdline string
	if c.SysProcAttr.CmdLine != "" {
		cmdline = c.SysProcAttr.CmdLine
	} else {
		cmdline = windows.ComposeCommandLine(c.Args)
	}
	argvp, err := windows.UTF16PtrFromString(cmdline)
	if err != nil {
		return err
	}

	var dirp *uint16
	if len(c.Dir) != 0 {
		dirp, err = windows.UTF16PtrFromString(c.Dir)
		if err != nil {
			return err
		}
	}

	if c.Env == nil {
		c.Env, err = execEnvDefault(c.SysProcAttr)
		if err != nil {
			return err
		}
	}

	siEx := new(windows.StartupInfoEx)
	siEx.Flags = windows.STARTF_USESTDHANDLES
	pi := new(windows.ProcessInformation)

	// Need EXTENDED_STARTUPINFO_PRESENT as we're making use of the attribute list field.
	flags := uint32(windows.CREATE_UNICODE_ENVIRONMENT) | windows.EXTENDED_STARTUPINFO_PRESENT | c.SysProcAttr.CreationFlags

	// Allocate an attribute list that's large enough to do the operations we care about
	// 2. Pseudo console setup if one was requested.
	// Therefore we need a list of size 3.
	attrs, err := windows.NewProcThreadAttributeList(1)
	if err != nil {
		return fmt.Errorf("failed to initialize process thread attribute list: %w", err)
	}

	c.sys = &conPtySys{
		attrs: attrs,
		done:  make(chan error, 1),
	}

	if err := pty.updateProcThreadAttribute(attrs); err != nil {
		return err
	}

	var zeroSec windows.SecurityAttributes
	pSec := &windows.SecurityAttributes{Length: uint32(unsafe.Sizeof(zeroSec)), InheritHandle: 1}
	if c.SysProcAttr.ProcessAttributes != nil {
		pSec = &windows.SecurityAttributes{
			Length:        c.SysProcAttr.ProcessAttributes.Length,
			InheritHandle: c.SysProcAttr.ProcessAttributes.InheritHandle,
		}
	}
	tSec := &windows.SecurityAttributes{Length: uint32(unsafe.Sizeof(zeroSec)), InheritHandle: 1}
	if c.SysProcAttr.ThreadAttributes != nil {
		tSec = &windows.SecurityAttributes{
			Length:        c.SysProcAttr.ThreadAttributes.Length,
			InheritHandle: c.SysProcAttr.ThreadAttributes.InheritHandle,
		}
	}

	siEx.ProcThreadAttributeList = attrs.List() //nolint:govet // unusedwrite: ProcThreadAttributeList will be read in syscall
	siEx.Cb = uint32(unsafe.Sizeof(*siEx))
	if c.SysProcAttr.Token != 0 {
		err = windows.CreateProcessAsUser(
			windows.Token(c.SysProcAttr.Token),
			argv0p,
			argvp,
			pSec,
			tSec,
			false,
			flags,
			createEnvBlock(addCriticalEnv(dedupEnvCase(true, c.Env))),
			dirp,
			&siEx.StartupInfo,
			pi,
		)
	} else {
		err = windows.CreateProcess(
			argv0p,
			argvp,
			pSec,
			tSec,
			false,
			flags,
			createEnvBlock(addCriticalEnv(dedupEnvCase(true, c.Env))),
			dirp,
			&siEx.StartupInfo,
			pi,
		)
	}
	if err != nil {
		return fmt.Errorf("failed to create process: %w", err)
	}
	// Don't need the thread handle for anything.
	defer func() {
		_ = windows.CloseHandle(pi.Thread)
	}()

	// Grab an *os.Process to avoid reinventing the wheel here. The stdlib has great logic around waiting, exit code status/cleanup after a
	// process has been launched.
	c.Process, err = os.FindProcess(int(pi.ProcessId))
	if err != nil {
		// If we can't find the process via os.FindProcess, terminate the process as that's what we rely on for all further operations on the
		// object.
		if tErr := windows.TerminateProcess(pi.Process, 1); tErr != nil {
			return fmt.Errorf("failed to terminate process after process not found: %w", tErr)
		}
		return fmt.Errorf("failed to find process after starting: %w", err)
	}

	if c.ctx != nil {
		if c.Cancel == nil {
			c.Cancel = func() error {
				if c.Process == nil {
					return errNotStarted
				}
				return c.Process.Kill()
			}
		}
		go c.waitOnContext()
	}

	return nil
}

func (c *Cmd) waitOnContext() {
	sys := c.sys.(*conPtySys)
	select {
	case <-c.ctx.Done():
		_ = c.Cancel()
		sys.cmdErr = c.ctx.Err()
	case err := <-sys.done:
		sys.cmdErr = err
	}
}

func (c *Cmd) wait() (retErr error) {
	if c.Process == nil {
		return errNotStarted
	}
	if c.ProcessState != nil {
		return errors.New("process already waited on")
	}
	defer func() {
		sys := c.sys.(*conPtySys)
		sys.attrs.Delete()
		sys.done <- nil
		if retErr == nil {
			retErr = sys.cmdErr
		}
	}()
	c.ProcessState, retErr = c.Process.Wait()
	if retErr != nil {
		return retErr
	}
	return
}

//
// Below are a bunch of helpers for working with Windows' CreateProcess family of functions. These are mostly exact copies of the same utilities
// found in the go stdlib.
//

func execEnvDefault(sys *syscall.SysProcAttr) (env []string, err error) {
	if sys == nil || sys.Token == 0 {
		return syscall.Environ(), nil
	}

	var block *uint16
	err = windows.CreateEnvironmentBlock(&block, windows.Token(sys.Token), false)
	if err != nil {
		return nil, err
	}

	defer windows.DestroyEnvironmentBlock(block)
	blockp := uintptr(unsafe.Pointer(block))

	for {
		// find NUL terminator
		end := unsafe.Pointer(blockp)
		for *(*uint16)(end) != 0 {
			end = unsafe.Pointer(uintptr(end) + 2)
		}

		n := (uintptr(end) - uintptr(unsafe.Pointer(blockp))) / 2
		if n == 0 {
			// environment block ends with empty string
			break
		}

		entry := (*[(1 << 30) - 1]uint16)(unsafe.Pointer(blockp))[:n:n]
		env = append(env, string(utf16.Decode(entry)))
		blockp += 2 * (uintptr(len(entry)) + 1)
	}
	return
}

func isSlash(c uint8) bool {
	return c == '\\' || c == '/'
}

func normalizeDir(dir string) (name string, err error) {
	ndir, err := syscall.FullPath(dir)
	if err != nil {
		return "", err
	}
	if len(ndir) > 2 && isSlash(ndir[0]) && isSlash(ndir[1]) {
		// dir cannot have \\server\share\path form
		return "", syscall.EINVAL
	}
	return ndir, nil
}

func volToUpper(ch int) int {
	if 'a' <= ch && ch <= 'z' {
		ch += 'A' - 'a'
	}
	return ch
}

func joinExeDirAndFName(dir, p string) (name string, err error) {
	if len(p) == 0 {
		return "", syscall.EINVAL
	}
	if len(p) > 2 && isSlash(p[0]) && isSlash(p[1]) {
		// \\server\share\path form
		return p, nil
	}
	if len(p) > 1 && p[1] == ':' {
		// has drive letter
		if len(p) == 2 {
			return "", syscall.EINVAL
		}
		if isSlash(p[2]) {
			return p, nil
		} else {
			d, err := normalizeDir(dir)
			if err != nil {
				return "", err
			}
			if volToUpper(int(p[0])) == volToUpper(int(d[0])) {
				return syscall.FullPath(d + "\\" + p[2:])
			} else {
				return syscall.FullPath(p)
			}
		}
	} else {
		// no drive letter
		d, err := normalizeDir(dir)
		if err != nil {
			return "", err
		}
		if isSlash(p[0]) {
			return windows.FullPath(d[:2] + p)
		} else {
			return windows.FullPath(d + "\\" + p)
		}
	}
}

// createEnvBlock converts an array of environment strings into
// the representation required by CreateProcess: a sequence of NUL
// terminated strings followed by a nil.
// Last bytes are two UCS-2 NULs, or four NUL bytes.
func createEnvBlock(envv []string) *uint16 {
	if len(envv) == 0 {
		return &utf16.Encode([]rune("\x00\x00"))[0]
	}
	length := 0
	for _, s := range envv {
		length += len(s) + 1
	}
	length++

	b := make([]byte, length)
	i := 0
	for _, s := range envv {
		l := len(s)
		copy(b[i:i+l], []byte(s))
		copy(b[i+l:i+l+1], []byte{0})
		i = i + l + 1
	}
	copy(b[i:i+1], []byte{0})

	return &utf16.Encode([]rune(string(b)))[0]
}

// dedupEnvCase is dedupEnv with a case option for testing.
// If caseInsensitive is true, the case of keys is ignored.
func dedupEnvCase(caseInsensitive bool, env []string) []string {
	out := make([]string, 0, len(env))
	saw := make(map[string]int, len(env)) // key => index into out
	for _, kv := range env {
		eq := strings.Index(kv, "=")
		if eq < 0 {
			out = append(out, kv)
			continue
		}
		k := kv[:eq]
		if caseInsensitive {
			k = strings.ToLower(k)
		}
		if dupIdx, isDup := saw[k]; isDup {
			out[dupIdx] = kv
			continue
		}
		saw[k] = len(out)
		out = append(out, kv)
	}
	return out
}

// addCriticalEnv adds any critical environment variables that are required
// (or at least almost always required) on the operating system.
// Currently this is only used for Windows.
func addCriticalEnv(env []string) []string {
	for _, kv := range env {
		eq := strings.Index(kv, "=")
		if eq < 0 {
			continue
		}
		k := kv[:eq]
		if strings.EqualFold(k, "SYSTEMROOT") {
			// We already have it.
			return env
		}
	}
	return append(env, "SYSTEMROOT="+os.Getenv("SYSTEMROOT"))
}
