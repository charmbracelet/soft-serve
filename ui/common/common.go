package common

import (
	"context"

	"github.com/aymanbagabas/go-osc52"
	"github.com/charmbracelet/soft-serve/git"
	"github.com/charmbracelet/soft-serve/server/config"
	"github.com/charmbracelet/soft-serve/ui/keymap"
	"github.com/charmbracelet/soft-serve/ui/styles"
	"github.com/charmbracelet/ssh"
	zone "github.com/lrstanley/bubblezone"
)

type contextKey struct {
	name string
}

// Keys to use for context.Context.
var (
	ConfigKey = &contextKey{"config"}
	RepoKey   = &contextKey{"repo"}
)

// Common is a struct all components should embed.
type Common struct {
	ctx           context.Context
	Width, Height int
	Styles        *styles.Styles
	KeyMap        *keymap.KeyMap
	Copy          *osc52.Output
	Zone          *zone.Manager
}

// NewCommon returns a new Common struct.
func NewCommon(ctx context.Context, copy *osc52.Output, width, height int) Common {
	if ctx == nil {
		ctx = context.TODO()
	}
	return Common{
		ctx:    ctx,
		Width:  width,
		Height: height,
		Copy:   copy,
		Styles: styles.DefaultStyles(),
		KeyMap: keymap.DefaultKeyMap(),
		Zone:   zone.New(),
	}
}

// SetValue sets a value in the context.
func (c *Common) SetValue(key, value interface{}) {
	c.ctx = context.WithValue(c.ctx, key, value)
}

// SetSize sets the width and height of the common struct.
func (c *Common) SetSize(width, height int) {
	c.Width = width
	c.Height = height
}

// Config returns the server config.
func (c *Common) Config() *config.Config {
	v := c.ctx.Value(ConfigKey)
	if cfg, ok := v.(*config.Config); ok {
		return cfg
	}
	return nil
}

// Repo returns the repository.
func (c *Common) Repo() *git.Repository {
	v := c.ctx.Value(RepoKey)
	if r, ok := v.(*git.Repository); ok {
		return r
	}
	return nil
}

// PublicKey returns the public key.
func (c *Common) PublicKey() ssh.PublicKey {
	v := c.ctx.Value(ssh.ContextKeyPublicKey)
	if p, ok := v.(ssh.PublicKey); ok {
		return p
	}
	return nil
}
