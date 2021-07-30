package main

import (
	"fmt"
	"log"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/charm/keygen"
	"github.com/gliderlabs/ssh"
	gossh "golang.org/x/crypto/ssh"
)

type Middleware func(ssh.Handler) ssh.Handler

func LoggingMiddleware() Middleware {
	return func(sh ssh.Handler) ssh.Handler {
		return func(s ssh.Session) {
			hpk := s.PublicKey() != nil
			log.Printf("%s connect %v %v\n", s.RemoteAddr().String(), hpk, s.Command())
			sh(s)
			log.Printf("%s disconnect %v %v\n", s.RemoteAddr().String(), hpk, s.Command())
		}
	}
}

func logError(s ssh.Session, err error) {
	log.Printf("%s error %v: %s\n", s.RemoteAddr().String(), s.Command(), err)
}

func BubbleTeaMiddleware(bth func(ssh.Session) tea.Model, opts ...tea.ProgramOption) Middleware {
	return func(sh ssh.Handler) ssh.Handler {
		return func(s ssh.Session) {
			m := bth(s)
			if m != nil {
				opts = append(opts, tea.WithInput(s), tea.WithOutput(s))
				p := tea.NewProgram(m, opts...)
				err := p.Start()
				if err != nil {
					logError(s, err)
				}
			}
			sh(s)
		}
	}
}

type Server struct {
	server *ssh.Server
	key    gossh.PublicKey
}

func NewServer(port int, keyPath string, mw ...Middleware) (*Server, error) {
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
