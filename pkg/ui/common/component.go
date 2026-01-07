package common

import (
	"charm.land/bubbles/v2/help"
	tea "charm.land/bubbletea/v2"
)

// Model represents a simple UI model.
type Model interface {
	Init() tea.Cmd
	Update(tea.Msg) (Model, tea.Cmd)
	View() string
}

// Component represents a Bubble Tea model that implements a SetSize function.
type Component interface {
	Model
	help.KeyMap
	SetSize(width, height int)
}

// TabComponenet represents a model that is mounted to a tab.
// TODO: find a better name
type TabComponent interface {
	Component

	// StatusBarValue returns the status bar value component.
	StatusBarValue() string

	// StatusBarInfo returns the status bar info component.
	StatusBarInfo() string

	// SpinnerID returns the ID of the spinner.
	SpinnerID() int

	// TabName returns the name of the tab.
	TabName() string

	// Path returns the hierarchical path of the tab.
	Path() string
}
