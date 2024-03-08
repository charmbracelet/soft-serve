package common

import (
	"context"
	"fmt"

	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/log"
	"github.com/charmbracelet/soft-serve/git"
	"github.com/charmbracelet/soft-serve/pkg/backend"
	"github.com/charmbracelet/soft-serve/pkg/config"
	"github.com/charmbracelet/soft-serve/pkg/ui/keymap"
	"github.com/charmbracelet/soft-serve/pkg/ui/styles"
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
	Output        *termenv.Output
	Logger        *log.Logger
	HideCloneCmd  bool
}

// NewCommon returns a new Common struct.
func NewCommon(ctx context.Context, out *lipgloss.Renderer, width, height int) Common {
	if ctx == nil {
		ctx = context.TODO()
	}
	return Common{
		ctx:      ctx,
		Width:    width,
		Height:   height,
		Renderer: out,
		Output:   out.Output(),
		Styles:   styles.DefaultStyles(out),
		KeyMap:   keymap.DefaultKeyMap(),
		Zone:     zone.New(),
		Logger:   log.FromContext(ctx).WithPrefix("ui"),
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

// Context returns the context.
func (c *Common) Context() context.Context {
	return c.ctx
}

// Config returns the server config.
func (c *Common) Config() *config.Config {
	return config.FromContext(c.ctx)
}

// Backend returns the Soft Serve backend.
func (c *Common) Backend() *backend.Backend {
	return backend.FromContext(c.ctx)
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

// CloneCmd returns the clone command string.
func (c *Common) CloneCmd(publicURL, name string) string {
	if c.HideCloneCmd {
		return ""
	}
	return fmt.Sprintf("git clone %s", RepoURL(publicURL, name))
}

// IsFileMarkdown returns true if the file is markdown.
// It uses chroma lexers to analyze and determine the language.
func IsFileMarkdown(content, ext string) bool {
	var lang string
	lexer := lexers.Match(ext)
	if lexer == nil {
		lexer = lexers.Analyse(content)
	}
	if lexer != nil && lexer.Config() != nil {
		lang = lexer.Config().Name
	}
	return lang == "markdown"
}
