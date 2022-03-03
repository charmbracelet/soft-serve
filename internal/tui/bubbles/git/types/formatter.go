package types

import (
	"strings"

	"github.com/alecthomas/chroma/lexers"
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
	return mdt, nil
}

func RenderFile(path, content string, width int) (string, error) {
	lexer := lexers.Fallback
	if path == "" {
		lexer = lexers.Analyse(content)
	} else {
		lexer = lexers.Match(path)
	}
	lang := ""
	if lexer != nil && lexer.Config() != nil {
		lang = lexer.Config().Name
	}
	formatter := &gansi.CodeBlockElement{
		Code:     content,
		Language: lang,
	}
	if lang == "markdown" {
		md, err := Glamourize(width, content)
		if err != nil {
			return "", err
		}
		return md, nil
	}
	r := strings.Builder{}
	err := formatter.Render(&r, RenderCtx)
	if err != nil {
		return "", err
	}
	return r.String(), nil
}
