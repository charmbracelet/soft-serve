//go:build !linux && !darwin && !freebsd && !dragonfly && !netbsd && !openbsd && !solaris && !windows
// +build !linux,!darwin,!freebsd,!dragonfly,!netbsd,!openbsd,!solaris,!windows

package ssh

import (
	"os"

	"github.com/aymanbagabas/go-pty"
)

func ptyNew(p pty.Pty) (in *os.File, out *os.File, er *os.File, err error) { // nolint
	return nil, nil, nil, pty.ErrUnsupported
}
