package common

import (
	"github.com/charmbracelet/soft-serve/ui/keymap"
	"github.com/charmbracelet/soft-serve/ui/styles"
)

// Common is a struct all components should embed.
type Common struct {
	Styles *styles.Styles
	Keymap *keymap.KeyMap
	Width  int
	Height int
}

// SetSize sets the width and height of the common struct.
func (c *Common) SetSize(width, height int) {
	c.Width = width
	c.Height = height
}
