package processor

import (
	"io"

	"github.com/Lekuruu/bbgo/context"
	"github.com/Lekuruu/bbgo/node"
)

// URL processes [url] bbcode.
func URL(ctx *context.Context, tag node.Tag, w io.Writer) {
	switch t := tag.(type) {
	case *node.OpeningTag:
		io.WriteString(w, `<a href="`)
		if v := t.Value(); v != "" {
			io.WriteString(w, v)
			io.WriteString(w, `">`)
		}
	case *node.ClosingTag:
		if ot, ok := ctx.Prev2Node().(*node.OpeningTag); ok && ot.TagName() == "url" && ot.Value() == "" {
			io.WriteString(w, `">`)
			io.WriteString(w, ctx.PrevNode().String())
		}
		io.WriteString(w, `</a>`)
	}
}
