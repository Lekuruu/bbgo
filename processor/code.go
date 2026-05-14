package processor

import (
	"io"

	"github.com/Lekuruu/bbgo/context"
	"github.com/Lekuruu/bbgo/node"
)

// Code processes [code] bbcode.
func Code(ctx *context.Context, tag node.Tag, w io.Writer) {
	switch tag.(type) {
	case *node.OpeningTag:
		io.WriteString(w, "<pre>")
		ctx.BeginRawMode(tag)
	case *node.ClosingTag:
		io.WriteString(w, "</pre>")
		ctx.EndRawMode(tag)
	}
}
