package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/charmbracelet/soft-serve/config"
	"github.com/charmbracelet/soft-serve/server"
	"github.com/spf13/cobra"
)

var (
	serveCmd = &cobra.Command{
		Use:   "serve",
		Short: "Start the server",
		Long:  "Start the server",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
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
			return s.Shutdown(ctx)
		},
	}
)
