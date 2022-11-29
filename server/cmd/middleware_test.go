package cmd

import (
	"os"
	"testing"

	"github.com/charmbracelet/soft-serve/server/config"
	"github.com/charmbracelet/wish/testsession"
	"github.com/gliderlabs/ssh"
)

var ()

func TestMiddleware(t *testing.T) {
	t.Cleanup(func() {
		os.RemoveAll("testmiddleware")
	})
	cfg := &config.Config{
		Host: "localhost",
		SSH: config.SSHConfig{
			Port: 22223,
		},
		DataPath: "testmiddleware",
	}
	_ = testsession.New(t, &ssh.Server{
		Handler: Middleware(cfg)(func(s ssh.Session) {
			t.Run("TestCatConfig", func(t *testing.T) {
				_, err := s.Write([]byte("cat config/config.json"))
				if err == nil {
					t.Errorf("Expected error, got nil")
				}
			})
		}),
	}, nil)
}
