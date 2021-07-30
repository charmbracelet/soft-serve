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
			log.Printf("%s connect %v %v\n", s.RemoteAddr().String(), hpk, s.Command())
			sh(s)
			log.Printf("%s disconnect %v %v\n", s.RemoteAddr().String(), hpk, s.Command())
		}
	}
}
