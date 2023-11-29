//go:build windows
// +build windows

package ssh

import (
	"os"

	"github.com/aymanbagabas/go-pty"
)

func ptyNew(p pty.Pty) (in *os.File, out *os.File, er *os.File, err error) { // nolint
	tty := p.(pty.ConPty)
	in = tty.InputPipe()
	out = tty.OutputPipe()
	er = tty.OutputPipe()
	return
}
