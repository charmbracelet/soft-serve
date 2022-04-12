package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/charmbracelet/soft-serve/server"
	"github.com/charmbracelet/soft-serve/server/config"
)

var (
	// Version contains the application version number. It's set via ldflags
	// when building.
	Version = ""

	// CommitSHA contains the SHA of the commit that this application was built
	// against. It's set via ldflags when building.
	CommitSHA = ""

	version = flag.Bool("version", false, "display version")
)

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Soft Serve, a self-hostable Git server for the command line.\n\n")
		flag.PrintDefaults()
	}

	flag.Parse()

	if *version {
		if len(CommitSHA) > 7 {
			CommitSHA = CommitSHA[:7]
		}
		if Version == "" {
			Version = "(built from source)"
		}

		fmt.Printf("Soft Serve %s", Version)
		if len(CommitSHA) > 0 {
			fmt.Printf(" (%s)", CommitSHA)
		}

		fmt.Println()
		os.Exit(0)
	}

	cfg := config.DefaultConfig()
	s := server.NewServer(cfg)

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	log.Printf("Starting SSH server on %s:%d", cfg.BindAddr, cfg.Port)
	go func() {
		if err := s.Start(); err != nil {
			log.Fatalln(err)
		}
	}()

	<-done

	log.Printf("Stopping SSH server on %s:%d", cfg.BindAddr, cfg.Port)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer func() { cancel() }()
	if err := s.Shutdown(ctx); err != nil {
		log.Fatalln(err)
	}
}
