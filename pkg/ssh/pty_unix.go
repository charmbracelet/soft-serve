//go:build darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris
// +build darwin dragonfly freebsd linux netbsd openbsd solaris

package ssh

import (
	"os"

	"github.com/aymanbagabas/go-pty"
)

func ptyNew(p pty.Pty) (in *os.File, out *os.File, er *os.File, err error) { // nolint
	tty := p.(pty.UnixPty)
	in = tty.Slave()
	out = tty.Slave()
	er = tty.Slave()
	return
}
