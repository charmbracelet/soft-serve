package server

import (
	"io"

	"github.com/charmbracelet/log"
	"github.com/charmbracelet/soft-serve/server/hooks"
)

var _ hooks.Hooks = (*Server)(nil)

// PostReceive is called by the git post-receive hook.
//
// It implements Hooks.
func (*Server) PostReceive(stdin io.Reader, stdout io.Writer, stderr io.Writer, repo string, args []hooks.HookArg) {
	io.WriteString(stdout, "Hello, world!\n")
	log.WithPrefix("server.hooks").Debug("post-receive hook called", "repo", repo, "args", args)
}

// PreReceive is called by the git pre-receive hook.
//
// It implements Hooks.
func (*Server) PreReceive(stdin io.Reader, stdout io.Writer, stderr io.Writer, repo string, args []hooks.HookArg) {
	log.WithPrefix("server.hooks").Debug("pre-receive hook called", "repo", repo, "args", args)
}

// Update is called by the git update hook.
//
// It implements Hooks.
func (*Server) Update(stdin io.Reader, stdout io.Writer, stderr io.Writer, repo string, arg hooks.HookArg) {
	log.WithPrefix("server.hooks").Debug("update hook called", "repo", repo, "arg", arg)
}

// PostUpdate is called by the git post-update hook.
//
// It implements Hooks.
func (s *Server) PostUpdate(stdin io.Reader, stdout io.Writer, stderr io.Writer, repo string, args ...string) {
	rr, err := s.Config.Backend.Repository(repo)
	if err != nil {
		log.WithPrefix("server.hooks.post-update").Error("error getting repository", "repo", repo, "err", err)
		return
	}

	r, err := rr.Open()
	if err != nil {
		log.WithPrefix("server.hooks.post-update").Error("error opening repository", "repo", repo, "err", err)
		return
	}

	if err := r.UpdateServerInfo(); err != nil {
		log.WithPrefix("server.hooks.post-update").Error("error updating server info", "repo", repo, "err", err)
		return
	}
}
