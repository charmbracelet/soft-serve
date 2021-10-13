package main

import (
	"log"

	"github.com/charmbracelet/soft/config"
	"github.com/charmbracelet/soft/server"
)

func main() {
	cfg := config.DefaultConfig()
	s := server.NewServer(cfg)
	log.Printf("Starting SSH server on %s:%d\n", cfg.Host, cfg.Port)
	err := s.Start()
	if err != nil {
		log.Fatalln(err)
	}
}
