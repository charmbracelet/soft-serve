package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/charmbracelet/soft-serve/server"
	"github.com/charmbracelet/soft-serve/server/config"
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

			log.Printf("Starting SSH server on %s:%d", cfg.BindAddr, cfg.Port)

			done := make(chan os.Signal, 1)
			lch := make(chan error, 1)
			go func() {
				defer close(lch)
				defer close(done)
				lch <- s.Start()
			}()

			signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
			<-done

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			if err := s.Shutdown(ctx); err != nil {
				return err
			}

			// wait for serve to finish
			return <-lch
		},
	}
)
