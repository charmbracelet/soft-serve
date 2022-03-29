package server

import (
	"testing"

	"github.com/charmbracelet/soft-serve/internal/config"
	"github.com/charmbracelet/wish/testsession"
	"github.com/gliderlabs/ssh"
	"github.com/matryer/is"
)

var ()

func TestMiddleware(t *testing.T) {
	is := is.New(t)
	appCfg, err := config.NewConfig(cfg)
	is.NoErr(err)
	_ = testsession.New(t, &ssh.Server{
		Handler: softServeMiddleware(appCfg)(func(s ssh.Session) {
			t.Run("TestCatConfig", func(t *testing.T) {
				_, err := s.Write([]byte("config/config.json"))
				if err == nil {
					t.Errorf("Expected error, got nil")
				}
			})
		}),
	}, nil)
}
