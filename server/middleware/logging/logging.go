package logging

import (
	"log"
	"smoothie/server/middleware"

	"github.com/gliderlabs/ssh"
)

func Middleware() middleware.Middleware {
	return func(sh ssh.Handler) ssh.Handler {
		return func(s ssh.Session) {
			hpk := s.PublicKey() != nil
			pty, _, _ := s.Pty()
			log.Printf("%s connect %v %v %s %v %v\n", s.RemoteAddr().String(), hpk, s.Command(), pty.Term, pty.Window.Width, pty.Window.Height)
			sh(s)
			log.Printf("%s disconnect\n", s.RemoteAddr().String())
		}
	}
}
