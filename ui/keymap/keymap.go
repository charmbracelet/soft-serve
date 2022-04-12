package keymap

import "github.com/charmbracelet/bubbles/key"

// KeyMap is a map of key bindings for the UI.
type KeyMap struct {
	Quit key.Binding
}

// DefaultKeyMap returns the default key map.
func DefaultKeyMap() *KeyMap {
	km := new(KeyMap)

	km.Quit = key.NewBinding(
		key.WithKeys(
			"ctrl-c",
			"q",
		),
		key.WithHelp(
			"q",
			"quit",
		),
	)

	return km
}
