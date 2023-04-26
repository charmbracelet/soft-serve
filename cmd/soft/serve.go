package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/charmbracelet/soft-serve/log"
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
			ctx := cmd.Context()
			cfg := config.DefaultConfig()
			s, err := server.NewServer(ctx, cfg)
			if err != nil {
				return fmt.Errorf("start server: %w", err)
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

			ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
			defer cancel()
			if err := s.Shutdown(ctx); err != nil {
				return err
			}

			// wait for serve to finish
			return <-lch
		},
	}
)
