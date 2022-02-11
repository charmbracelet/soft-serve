package types

import (
	"github.com/charmbracelet/glamour"
	gansi "github.com/charmbracelet/glamour/ansi"
	"github.com/muesli/reflow/wrap"
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

func Glamourize(w int, md string) (string, error) {
	if w > GlamourMaxWidth {
		w = GlamourMaxWidth
	}
	tr, err := glamour.NewTermRenderer(
		glamour.WithStyles(DefaultStyles()),
		glamour.WithWordWrap(w),
	)

	if err != nil {
		return "", err
	}
	mdt, err := tr.Render(md)
	if err != nil {
		return "", err
	}
	// For now, hard-wrap long lines in Glamour that would otherwise break the
	// layout when wrapping. This may be due to #43 in Reflow, which has to do
	// with a bug in the way lines longer than the given width are wrapped.
	//
	//     https://github.com/muesli/reflow/issues/43
	//
	// TODO: solve this upstream in Glamour/Reflow.
	return wrap.String(mdt, w), nil
}
