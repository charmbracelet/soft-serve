//go:build darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris
// +build darwin dragonfly freebsd linux netbsd openbsd solaris

// This is an example of binding soft-serve ssh port to a restricted port (<1024) and
// then droping root privileges to a different user to run the server.
// Make sure you run this as root.

package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/charmbracelet/log"

	"github.com/charmbracelet/soft-serve/server"
	"github.com/charmbracelet/soft-serve/server/config"
)

var (
	port = flag.Int("port", 22, "port to listen on")
	gid  = flag.Int("gid", 1000, "group id to run as")
	uid  = flag.Int("uid", 1000, "user id to run as")
)

func main() {
	flag.Parse()
	addr := fmt.Sprintf(":%d", *port)
	// To listen on port 22 we need root privileges
	ls, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatal("Can't listen", "err", err)
	}
	// We don't need root privileges any more
	if err := syscall.Setgid(*gid); err != nil {
		log.Fatal("Setgid error", "err", err)
	}
	if err := syscall.Setuid(*uid); err != nil {
		log.Fatal("Setuid error", "err", err)
	}
	ctx := context.Background()
	cfg := config.DefaultConfig()
	cfg.SSH.ListenAddr = fmt.Sprintf(":%d", *port)
	s, err := server.NewServer(ctx, cfg)
	if err != nil {
		log.Fatal(err)
	}

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	log.Print("Starting SSH server", "addr", cfg.SSH.ListenAddr)
	go func() {
		if err := s.SSHServer.Serve(ls); err != nil {
			log.Fatal(err)
		}
	}()

	<-done

	log.Print("Stopping SSH server", "addr", cfg.SSH.ListenAddr)
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer func() { cancel() }()
	if err := s.Shutdown(ctx); err != nil {
		log.Fatal(err)
	}
}
