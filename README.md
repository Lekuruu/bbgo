# bbgo

This module provides a bbcode renderer, based on bbgo. It changes the renderer to be value based: tag renderers receive the rendered body of a tag instead of separate opening and closing events.

## Usage

Create a parser and call `Parse()` to render BBCode as HTML.

```go
parser := bbgo.New()

html := parser.Parse("[b]Hello World[/b]")
fmt.Println(html)

// Output:
// <b>Hello World</b>
```

Use `Format()` when you want to pass request or service context into custom renderers.

```go
html := parser.Format("[user=1]alice[/user]", bbgo.Context{
	"base_url": "https://example.com",
})
```

## Custom Tags

Use `AddFormatter` to register a tag. The renderer receives the tag name, rendered body, parsed options, parent tag options, and caller-provided context.

```go
parser := bbgo.New()

parser.AddFormatter("user", func(ctx bbgo.RenderContext) string {
	baseURL, _ := ctx.Context["base_url"].(string)
	userID := ctx.Options.Get("user")

	return fmt.Sprintf(
		`<a href="%s/u/%s">%s</a>`,
		baseURL,
		userID,
		ctx.Value,
	)
}, bbgo.TagOptions{
	RenderEmbedded:    true,
	TransformNewlines: true,
	EscapeHTML:        true,
	ReplaceLinks:      true,
	ReplaceCosmetic:   true,
})
```

The default tags can be replaced by registering the same tag name again.

```go
parser.AddSimpleFormatter("b", "<strong>%s</strong>", bbgo.TagOptions{
	RenderEmbedded:    true,
	TransformNewlines: true,
	EscapeHTML:        true,
	ReplaceLinks:      true,
	ReplaceCosmetic:   true,
})
```

## Parser Options

Use `NewWithOptions` to customize global behavior.

```go
parser := bbgo.NewWithOptions(bbgo.ParserOptions{
	Newline: "<br />",
	LinkRenderer: func(url string, ctx bbgo.Context) string {
		return fmt.Sprintf(`<a rel="nofollow" href="%s">%s</a>`, url, url)
	},
	UnknownLineRenderer: func(tagText string, ctx bbgo.Context) (string, bool) {
		return "<h2>" + tagText + "</h2>", true
	},
})
```

- `Newline` controls how rendered newlines are written, defaulting to `<br />`
- `DropUnrecognized` drops unknown tags instead of keeping them as text
- `MaxTagDepth` limits recursive rendering depth
- `LinkRenderer` controls automatic plain URL rendering
- `UnknownLineRenderer` handles unknown tags that appear alone on a line

## Supported Syntax

```text
[tag]basic tag[/tag]
[tag1][tag2]nested tags[/tag2][/tag1]

[tag=value]tag with value[/tag]
[tag arg=value]tag with named argument[/tag]
[tag="quote value"]tag with quoted value[/tag]
[tag flag]tag with flag option[/tag]

[tag=value foo="hello world" bar=baz]multiple tag arguments[/tag]
```

Tag and option names are case-insensitive. Quoted option values may contain spaces or `]`.

## Default Tags

- `[b]text[/b]` renders as `<b>text</b>`
- `[i]text[/i]` renders as `<i>text</i>`
- `[u]text[/u]` renders as `<u>text</u>`
- `[s]text[/s]` renders as `<s>text</s>`
- `[url]link[/url]` renders as a link to `link`
- `[url=link]text[/url]` renders as a link to `link`
- `[img]link[/img]` renders as an image tag
- `[color=red]text[/color]` renders as a colored span
- `[quote]text[/quote]` renders as a blockquote
- `[quote=Somebody]text[/quote]` renders as a blockquote with a citation
- `[quote name=Somebody]text[/quote]` renders as a blockquote with a citation
- `[code][b]anything[/b][/code]` renders the body without rendering embedded BBCode
- `[list][*] item 1[*] item 2[/list]` renders as an unordered list
- `[list=1][*] item 1[/list]` renders as an ordered list

## Text Transforms

Plain text is escaped by default. The renderer also converts plain URLs into links and performs small cosmetic replacements such as `---`, `--`, `...`, `(c)`, `(reg)`, and `(tm)`.

Per-tag `TagOptions` can disable embedded rendering, link replacement, cosmetic replacement, HTML escaping, or newline transformation. This is useful for tags like `[code]`, raw content, or service-specific embeds.

## Stripping Tags

Use `Strip()` to remove recognized tags while keeping the original text content.

```go
text := parser.Strip("[b]hello[/b]\n[i]world[/i]", false)
fmt.Println(text)

// Output:
// hello
// world
```

Pass `true` as the second argument to remove newlines as well.
