package common

import (
	"context"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/log"
	"github.com/charmbracelet/soft-serve/git"
	"github.com/charmbracelet/soft-serve/server/auth"
	"github.com/charmbracelet/soft-serve/server/backend"
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
	Renderer      *lipgloss.Renderer
	Logger        *log.Logger
}

// NewCommon returns a new Common struct.
func NewCommon(ctx context.Context, re *lipgloss.Renderer, width, height int) Common {
	if ctx == nil {
		ctx = context.TODO()
	}
	return Common{
		ctx:      ctx,
		Width:    width,
		Height:   height,
		Styles:   styles.DefaultStyles(re),
		KeyMap:   keymap.DefaultKeyMap(),
		Zone:     zone.New(),
		Logger:   log.FromContext(ctx).WithPrefix("ui"),
		Renderer: re,
	}
}

// Output returns the termenv output.
func (c *Common) Output() *termenv.Output {
	return c.Renderer.Output()
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
	return config.FromContext(c.ctx)
}

func (c *Common) Context() context.Context {
	return c.ctx
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

// Backend returns the server backend.
func (c *Common) Backend() *backend.Backend {
	return backend.FromContext(c.ctx)
}

// User returns the current user from context.
func (c *Common) User() auth.User {
	return auth.UserFromContext(c.ctx)
}
