package session

import (
	tea "github.com/charmbracelet/bubbletea"
	appCfg "github.com/charmbracelet/soft-serve/config"
	"github.com/gliderlabs/ssh"
)

// Session is a interface representing a UI session.
type Session interface {
	// Send sends a message to the parent Bubble Tea program.
	Send(tea.Msg)
	// Config returns the app configuration.
	Config() *appCfg.Config
	// PublicKey returns the public key of the user.
	PublicKey() ssh.PublicKey
}
