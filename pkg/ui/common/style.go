package common

import (
	"github.com/charmbracelet/colorprofile"
	gansi "github.com/charmbracelet/glamour/ansi"
	"github.com/charmbracelet/glamour/styles"
	"github.com/muesli/termenv"
)

// DefaultColorProfile is the default color profile used by the SSH server.
var DefaultColorProfile = colorprofile.ANSI256

func strptr(s string) *string {
	return &s
}

// StyleConfig returns the default Glamour style configuration.
func StyleConfig() gansi.StyleConfig {
	noColor := strptr("")
	s := styles.DarkStyleConfig
	s.H1.BackgroundColor = noColor
	s.H1.Prefix = "# "
	s.H1.Suffix = ""
	s.H1.Color = strptr("39")
	s.Document.StylePrimitive.Color = noColor
	s.CodeBlock.Chroma.Text.Color = noColor
	s.CodeBlock.Chroma.Name.Color = noColor
	// This fixes an issue with the default style config. For example
	// highlighting empty spaces with red in Dockerfile type.
	s.CodeBlock.Chroma.Error.BackgroundColor = noColor
	return s
}

// StyleRenderer returns a new Glamour renderer with the DefaultColorProfile.
func StyleRenderer() gansi.RenderContext {
	return StyleRendererWithStyles(StyleConfig())
}

// StyleRendererWithStyles returns a new Glamour renderer with the
// DefaultColorProfile and styles.
func StyleRendererWithStyles(styles gansi.StyleConfig) gansi.RenderContext {
	return gansi.NewRenderContext(gansi.Options{
		ColorProfile: termenv.ANSI256,
		Styles:       styles,
	})
}
