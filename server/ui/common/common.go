package common

import (
	"context"

	"github.com/charmbracelet/log"
	"github.com/charmbracelet/soft-serve/git"
	"github.com/charmbracelet/soft-serve/server/config"
	"github.com/charmbracelet/soft-serve/server/ui/keymap"
	"github.com/charmbracelet/soft-serve/server/ui/styles"
	"github.com/charmbracelet/ssh"
	zone "github.com/lrstanley/bubblezone"
	"github.com/muesli/termenv"
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
	Zone          *zone.Manager
	Output        *termenv.Output
	Logger        *log.Logger
}

// NewCommon returns a new Common struct.
func NewCommon(ctx context.Context, out *termenv.Output, width, height int) Common {
	if ctx == nil {
		ctx = context.TODO()
	}
	return Common{
		ctx:    ctx,
		Width:  width,
		Height: height,
		Output: out,
		Styles: styles.DefaultStyles(),
		KeyMap: keymap.DefaultKeyMap(),
		Zone:   zone.New(),
		Logger: log.FromContext(ctx).WithPrefix("ui"),
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
