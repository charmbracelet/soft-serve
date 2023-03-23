package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/charmbracelet/log"
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
			s, err := server.NewServer(cfg)
			if err != nil {
				return err
			}

			if cfg.Debug {
				log.SetLevel(log.DebugLevel)
			}

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
