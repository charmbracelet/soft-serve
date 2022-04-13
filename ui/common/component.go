package common

import (
	"github.com/charmbracelet/bubbles/help"
	tea "github.com/charmbracelet/bubbletea"
)

type Component interface {
	tea.Model
	SetSize(width, height int)
}

type Page interface {
	Component
	help.KeyMap
}
