package main

import (
	"log"

	"github.com/charmbracelet/soft"
)

func main() {
	cfg := soft.DefaultConfig()
	s := soft.NewServer(cfg)
	log.Printf("Starting SSH server on %s:%d\n", cfg.Host, cfg.Port)
	err := s.ListenAndServe()
	if err != nil {
		log.Fatalln(err)
	}
}
