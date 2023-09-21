//go:build !windows
// +build !windows

package pty

import (
	"context"
	"errors"
	"os"
	"os/exec"

	"github.com/creack/pty"
	"golang.org/x/sys/unix"
)

type unixPty struct {
	master, slave *os.File
	closed        bool
}

var _ Pty = &unixPty{}

// Close implements Pty.
func (p *unixPty) Close() error {
	if p.closed {
		return nil
	}
	defer func() {
		p.closed = true
	}()
	return errors.Join(p.master.Close(), p.slave.Close())
}

// Command implements Pty.
func (p *unixPty) Command(name string, args ...string) *Cmd {
	return p.CommandContext(nil, name, args...) // nolint:staticcheck
}

// CommandContext implements Pty.
func (p *unixPty) CommandContext(ctx context.Context, name string, args ...string) *Cmd {
	cmd := exec.Command(name, args...)
	if ctx != nil {
		cmd = exec.CommandContext(ctx, name, args...)
	}
	c := &Cmd{
		ctx:  ctx,
		pty:  p,
		sys:  cmd,
		Path: name,
		Args: append([]string{name}, args...),
	}
	return c
}

// Name implements Pty.
func (p *unixPty) Name() string {
	return p.slave.Name()
}

// Read implements Pty.
func (p *unixPty) Read(b []byte) (n int, err error) {
	return p.master.Read(b)
}

func (p *unixPty) Control(f func(fd uintptr)) error {
	conn, err := p.master.SyscallConn()
	if err != nil {
		return err
	}
	return conn.Control(f)
}

// Resize implements Pty.
func (p *unixPty) Resize(rows int, cols int) error {
	var ctrlErr error
	if err := p.Control(func(fd uintptr) {
		ctrlErr = unix.IoctlSetWinsize(int(fd), unix.TIOCSWINSZ, &unix.Winsize{
			Row: uint16(rows),
			Col: uint16(cols),
		})
	}); err != nil {
		return err
	}

	return ctrlErr
}

// Write implements Pty.
func (p *unixPty) Write(b []byte) (n int, err error) {
	return p.master.Write(b)
}

func newPty() (Pty, error) {
	master, slave, err := pty.Open()
	if err != nil {
		return nil, err
	}

	return &unixPty{
		master: master,
		slave:  slave,
	}, nil
}

func (c *Cmd) start() error {
	cmd, ok := c.sys.(*exec.Cmd)
	if !ok {
		return ErrInvalidCommand
	}
	pty, ok := c.pty.(*unixPty)
	if !ok {
		return ErrInvalidCommand
	}

	cmd.Stdin = pty.slave
	cmd.Stdout = pty.slave
	cmd.Stderr = pty.slave
	cmd.SysProcAttr = &unix.SysProcAttr{
		Setsid:  true,
		Setctty: true,
	}
	if err := cmd.Start(); err != nil {
		return err
	}

	c.Process = cmd.Process
	return nil
}

func (c *Cmd) wait() error {
	cmd, ok := c.sys.(*exec.Cmd)
	if !ok {
		return ErrInvalidCommand
	}
	err := cmd.Wait()
	c.ProcessState = cmd.ProcessState
	return err
}
