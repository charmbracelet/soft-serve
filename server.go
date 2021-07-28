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

type SessionHandler func(ssh.Session) (tea.Model, error)

type Server struct {
	server  *ssh.Server
	key     gossh.PublicKey
	handler SessionHandler
}

func NewServer(port int, keyPath string, handler SessionHandler) (*Server, error) {
	s := &Server{
		server:  &ssh.Server{},
		handler: handler,
	}
	s.server.Version = "OpenSSH_7.6p1"
	s.server.Addr = fmt.Sprintf(":%d", port)
	s.server.Handler = s.sessionHandler
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
	return s, nil
}

func (srv *Server) sessionHandler(s ssh.Session) {
	hpk := s.PublicKey() != nil
	log.Printf("%s connect %v %v\n", s.RemoteAddr().String(), hpk, s.Command())
	m, err := srv.handler(s)
	if err != nil {
		log.Printf("%s error %v %s\n", s.RemoteAddr().String(), hpk, err)
		s.Exit(1)
		return
	}
	if m != nil {
		p := tea.NewProgram(m, tea.WithInput(s), tea.WithOutput(s))
		err = p.Start()
		if err != nil {
			log.Printf("%s error %v %s\n", s.RemoteAddr().String(), hpk, err)
			s.Exit(1)
			return
		}
	}
	log.Printf("%s disconnect %v %v\n", s.RemoteAddr().String(), hpk, s.Command())
}

func (srv *Server) authHandler(ctx ssh.Context, key ssh.PublicKey) bool {
	return true
}

func (srv *Server) passHandler(ctx ssh.Context, pass string) bool {
	return true
}

func (srv *Server) Start() error {
	return srv.server.ListenAndServe()
}
