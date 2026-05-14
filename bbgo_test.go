package bbgo_test

import (
	"strings"
	"testing"

	"github.com/Lekuruu/bbgo"
)

func TestBBGO_Parse(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"empty string", "", ""},
		{"new line", "hello\n\nworld", "hello<br /><br />world"},
		{"normalize line endings", "a\r\nb\rc", "a<br />b<br />c"},
		{"escape html", `<script src="'>`, "&lt;script src=&quot;&#39;&gt;"},
		{"unknown tag", `[foo][b]hello[/b][/foo]`, `[foo]<b>hello</b>[/foo]`},
		{"case insensitive", `[B]hello[/B]`, `<b>hello</b>`},
		{"nested tags", `[b][i]hello[/i][/b]`, `<b><i>hello</i></b>`},
		{"implicit eof close", `[b]hello`, `<b>hello</b>`},
		{"color", `[color=#00BFFF]hello[/color]`, `<span style="color: #00BFFF;">hello</span>`},
		{"quote", `[quote]hello[/quote]`, `<blockquote>hello</blockquote>`},
		{"quote with value", `[quote=Somebody]hello[/quote]`, `<blockquote><cite>Somebody said:</cite>hello</blockquote>`},
		{"quote with attr", `[quote name="Somebody Else"]hello[/quote]`, `<blockquote><cite>Somebody Else said:</cite>hello</blockquote>`},
		{"quoted option with bracket", `[quote name="Some]body"]hello[/quote]`, `<blockquote><cite>Some]body said:</cite>hello</blockquote>`},
		{"url", `[url]https://example.com[/url]`, `<a href="https://example.com">https://example.com</a>`},
		{"url with value", `[url=example.com]Example[/url]`, `<a href="http://example.com">Example</a>`},
		{"img", `[img]https://example.com/logo.png[/img]`, `<img src="https://example.com/logo.png">`},
		{"code raw", "[code][b]some[/b]\n[/code][b]more[/b]", "<pre>[b]some[/b]<br /></pre><b>more</b>"},
		{"list", `[list][*] item 1[*] item 2[/list]`, `<ul><li> item 1</li><li> item 2</li></ul>`},
		{"ordered list", `[list=1][*] item 1[/list]`, `<ol><li> item 1</li></ol>`},
		{"new tag start before close", `[b [i]hello[/i]`, `[b <i>hello</i>`},
		{"cosmetic replacements", `a---b -- c... (c) (reg) (tm)`, `a&mdash;b &ndash; c&#8230; &copy; &reg; &trade;`},
		{"auto link", `go to example.com/path`, `go to <a rel="nofollow" href="http://example.com/path">example.com/path</a>`},
		{"auto link protects escaping", `https://example.com/a?b=1&c=2`, `<a rel="nofollow" href="https://example.com/a?b=1&c=2">https://example.com/a?b=1&c=2</a>`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := bbgo.New().Parse(tt.input)
			if got != tt.want {
				t.Fatalf("want %q, got %q", tt.want, got)
			}
		})
	}
}

func TestFlagOption(t *testing.T) {
	parser := bbgo.New()
	parser.AddFormatter("flag", func(ctx bbgo.RenderContext) string {
		if _, ok := ctx.Options["enabled"]; ok {
			return "enabled:" + ctx.Value
		}
		return "disabled:" + ctx.Value
	}, bbgo.TagOptions{
		RenderEmbedded:    true,
		TransformNewlines: true,
		EscapeHTML:        true,
		ReplaceLinks:      true,
		ReplaceCosmetic:   true,
	})

	got := parser.Parse("[flag enabled]body[/flag]")
	want := "enabled:body"
	if got != want {
		t.Fatalf("want %q, got %q", want, got)
	}
}

func TestCustomFormatter(t *testing.T) {
	parser := bbgo.New()
	parser.AddFormatter("user", func(ctx bbgo.RenderContext) string {
		return `<a data-id="` + ctx.Options.Get("user") + `">` + ctx.Value + `</a>`
	}, bbgo.TagOptions{
		RenderEmbedded:    true,
		TransformNewlines: true,
		EscapeHTML:        true,
		ReplaceLinks:      true,
		ReplaceCosmetic:   true,
	})

	got := parser.Parse(`[user=42][b]name[/b][/user]`)
	want := `<a data-id="42"><b>name</b></a>`
	if got != want {
		t.Fatalf("want %q, got %q", want, got)
	}
}

func TestOverrideDefaultFormatter(t *testing.T) {
	parser := bbgo.New()
	parser.AddSimpleFormatter("b", "<strong>%s</strong>", bbgo.TagOptions{
		RenderEmbedded:    true,
		TransformNewlines: true,
		EscapeHTML:        true,
		ReplaceLinks:      true,
		ReplaceCosmetic:   true,
	})

	got := parser.Parse(`[b]hello[/b]`)
	want := `<strong>hello</strong>`
	if got != want {
		t.Fatalf("want %q, got %q", want, got)
	}
}

func TestUnknownLineRenderer(t *testing.T) {
	parser := bbgo.NewWithOptions(bbgo.ParserOptions{
		UnknownLineRenderer: func(tagText string, context bbgo.Context) (string, bool) {
			return `<h2>` + tagText + `</h2>`, true
		},
	})

	got := parser.Parse("[Section]\nbody")
	want := "<h2>Section</h2><br />body"
	if got != want {
		t.Fatalf("want %q, got %q", want, got)
	}
}

func TestLinkRenderer(t *testing.T) {
	parser := bbgo.NewWithOptions(bbgo.ParserOptions{
		LinkRenderer: func(url string, context bbgo.Context) string {
			return `<a data-url="` + url + `">link</a>`
		},
	})

	got := parser.Format("visit https://example.com", bbgo.Context{"source": "test"})
	want := `visit <a data-url="https://example.com">link</a>`
	if got != want {
		t.Fatalf("want %q, got %q", want, got)
	}
}

func TestStrip(t *testing.T) {
	parser := bbgo.New()

	got := parser.Strip("[b]hello[/b]\n[i]world[/i]", false)
	want := "hello\nworld"
	if got != want {
		t.Fatalf("want %q, got %q", want, got)
	}

	got = parser.Strip("[b]hello[/b]\n[i]world[/i]", true)
	want = "helloworld"
	if got != want {
		t.Fatalf("want %q, got %q", want, got)
	}
}

func BenchmarkBBGO_Parse(b *testing.B) {
	var sb strings.Builder
	for i := 0; i < 20000; i++ {
		sb.WriteString("[quote]hello")
	}
	sb.WriteString("middle")
	for i := 0; i < 20000; i++ {
		sb.WriteString("world[/quote]")
	}

	parser := bbgo.New()
	input := sb.String()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		parser.Parse(input)
	}
}
