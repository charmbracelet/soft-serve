package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/charmbracelet/soft/config"
	"github.com/charmbracelet/soft/server"
)

var (
	Version   = ""
	CommitSHA = ""

	version = flag.Bool("version", false, "display version")
)

func main() {
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
	log.Printf("Starting SSH server on %s:%d\n", cfg.Host, cfg.Port)
	err := s.Start()
	if err != nil {
		log.Fatalln(err)
	}
}
