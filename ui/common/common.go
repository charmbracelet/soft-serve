package common

import (
	"github.com/charmbracelet/soft-serve/ui/keymap"
	"github.com/charmbracelet/soft-serve/ui/styles"
)

type Common struct {
	Styles *styles.Styles
	Keymap *keymap.KeyMap
	Width  int
	Height int
}

func (c *Common) SetSize(width, height int) {
	c.Width = width
	c.Height = height
}
