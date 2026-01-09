package common

import (
	gansi "charm.land/glamour/v2/ansi"
	"charm.land/glamour/v2/styles"
	"github.com/charmbracelet/colorprofile"
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
	// This fixes an issue with the default style config. For example
	// highlighting empty spaces with red in Dockerfile type.
	s.CodeBlock.Chroma.Error.BackgroundColor = noColor
	return s
}

// StyleRenderer returns a new Glamour renderer.
func StyleRenderer() gansi.RenderContext {
	return StyleRendererWithStyles(StyleConfig())
}

// StyleRendererWithStyles returns a new Glamour renderer.
func StyleRendererWithStyles(styles gansi.StyleConfig) gansi.RenderContext {
	return gansi.NewRenderContext(gansi.Options{
		Styles: styles,
	})
}
