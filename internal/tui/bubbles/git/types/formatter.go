package types

import (
	"github.com/charmbracelet/glamour"
	gansi "github.com/charmbracelet/glamour/ansi"
	"github.com/muesli/termenv"
)

var (
	RenderCtx = DefaultRenderCtx()
	Styles    = DefaultStyles()
)

func DefaultStyles() gansi.StyleConfig {
	noColor := ""
	s := glamour.DarkStyleConfig
	s.Document.StylePrimitive.Color = &noColor
	s.CodeBlock.Chroma.Text.Color = &noColor
	s.CodeBlock.Chroma.Name.Color = &noColor
	return s
}

func DefaultRenderCtx() gansi.RenderContext {
	return gansi.NewRenderContext(gansi.Options{
		ColorProfile: termenv.TrueColor,
		Styles:       DefaultStyles(),
	})
}

func NewRenderCtx(worldwrap int) gansi.RenderContext {
	return gansi.NewRenderContext(gansi.Options{
		ColorProfile: termenv.TrueColor,
		Styles:       DefaultStyles(),
		WordWrap:     worldwrap,
	})
}
