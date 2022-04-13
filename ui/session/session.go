package session

import (
	tea "github.com/charmbracelet/bubbletea"
	appCfg "github.com/charmbracelet/soft-serve/config"
	"github.com/gliderlabs/ssh"
)

// Session is a interface representing a UI session.
type Session interface {
	Send(tea.Msg)
	Config() *appCfg.Config
	PublicKey() ssh.PublicKey
}
