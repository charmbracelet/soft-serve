package server

import (
	"fmt"
	"log"
	"path/filepath"
	"smoothie/server/middleware"
	"strings"

	"github.com/charmbracelet/charm/keygen"
	"github.com/gliderlabs/ssh"
	gossh "golang.org/x/crypto/ssh"
)

type Server struct {
	server *ssh.Server
	key    gossh.PublicKey
}

func NewServer(port int, keyPath string, mw ...middleware.Middleware) (*Server, error) {
	s := &Server{server: &ssh.Server{}}
	s.server.Version = "OpenSSH_7.6p1"
	s.server.Addr = fmt.Sprintf(":%d", port)
	s.server.PasswordHandler = s.passHandler
	s.server.PublicKeyHandler = s.authHandler
	kps := strings.Split(keyPath, string(filepath.Separator))
	kp := strings.Join(kps[:len(kps)-1], string(filepath.Separator))
	n := strings.TrimRight(kps[len(kps)-1], "_ed25519")
	_, err := keygen.NewSSHKeyPair(kp, n, nil, "ed25519")
	if err != nil {
		return nil, err
	}
	k := ssh.HostKeyFile(keyPath)
	err = s.server.SetOption(k)
	if err != nil {
		return nil, err
	}
	h := func(s ssh.Session) {}
	for _, m := range mw {
		h = m(h)
	}
	s.server.Handler = h
	return s, nil
}

func (srv *Server) sessionHandler(s ssh.Session) {
}

func (srv *Server) authHandler(ctx ssh.Context, key ssh.PublicKey) bool {
	return true
}

func (srv *Server) passHandler(ctx ssh.Context, pass string) bool {
	return true
}

func (srv *Server) Start() error {
	log.Printf("Starting SSH server on %s\n", srv.server.Addr)
	return srv.server.ListenAndServe()
}
