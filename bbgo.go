package bbgo

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

const newlineSentinel = "\r"

var (
	autoLinkRE = regexp.MustCompile(`(?im)\b((?:https?://|www\d{0,3}[.]|[a-z0-9.\-]+[.][a-z]{2,}/)(?:[^\s()<>]+|\([^\s()<>]+\))+(?:\([^\s()<>]+\)|[^\s` + "`" + `!()\[\]{};:'".,<>?]))`)

	htmlReplacements = []replacement{
		{"&", "&amp;"},
		{"<", "&lt;"},
		{">", "&gt;"},
		{`"`, "&quot;"},
		{"'", "&#39;"},
	}

	cosmeticReplacements = []replacement{
		{"---", "&mdash;"},
		{"--", "&ndash;"},
		{"...", "&#8230;"},
		{"(c)", "&copy;"},
		{"(reg)", "&reg;"},
		{"(tm)", "&trade;"},
	}
)

type replacement struct {
	find string
	with string
}

// Options contains parsed tag arguments. Keys are lowercase.
type Options map[string]string

// Context is caller-defined data passed through formatting.
type Context map[string]any

// RenderFunc renders one tag after its body has been formatted.
type RenderFunc func(ctx RenderContext) string

// RenderContext contains all data available to a tag renderer.
type RenderContext struct {
	TagName string
	Value   string
	Options Options
	Parent  *TagOptions
	Context Context
}

// TagOptions controls how a tag body is found and transformed.
type TagOptions struct {
	TagName                string
	NewlineCloses          bool
	SameTagCloses          bool
	Standalone             bool
	RenderEmbedded         bool
	TransformNewlines      bool
	EscapeHTML             bool
	ReplaceLinks           bool
	ReplaceCosmetic        bool
	Strip                  bool
	SwallowTrailingNewline bool
}

// ParserOptions controls global formatting behavior.
type ParserOptions struct {
	Newline             string
	EscapeHTML          bool
	ReplaceLinks        bool
	ReplaceCosmetic     bool
	DropUnrecognized    bool
	MaxTagDepth         int
	LinkRenderer        func(url string, context Context) string
	UnknownLineRenderer func(tagText string, context Context) (html string, ok bool)
}

// BBGO is a reusable BBCode parser and formatter.
type BBGO struct {
	options    ParserOptions
	formatters map[string]formatter
}

type formatter struct {
	render  RenderFunc
	options TagOptions
}

type tokenKind int

const (
	tokenStart tokenKind = iota
	tokenEnd
	tokenNewline
	tokenText
	tokenStandalone
)

type parseToken struct {
	kind    tokenKind
	tagName string
	options Options
	text    string
}

// New creates a parser with generic default BBCode tags.
func New() *BBGO {
	return NewWithOptions(ParserOptions{})
}

// NewWithOptions creates a parser with caller-provided global options.
func NewWithOptions(options ParserOptions) *BBGO {
	b := &BBGO{
		options:    normalizeParserOptions(options),
		formatters: make(map[string]formatter),
	}
	b.registerDefaultFormatters()
	return b
}

// Parse formats input with no caller context.
func (b *BBGO) Parse(input string) string {
	return b.Format(input, nil)
}

// AddFormatter registers or replaces a tag formatter.
func (b *BBGO) AddFormatter(tagName string, render RenderFunc, options TagOptions) {
	name := normalizeName(tagName)
	options.TagName = name
	b.formatters[name] = formatter{render: render, options: options}
}

// AddSimpleFormatter registers a formatter using fmt.Sprintf with the rendered
// tag body as its only argument.
func (b *BBGO) AddSimpleFormatter(tagName string, format string, options TagOptions) {
	b.AddFormatter(tagName, func(ctx RenderContext) string {
		return fmt.Sprintf(format, ctx.Value)
	}, options)
}

// Format renders BBCode input into HTML.
func (b *BBGO) Format(input string, context Context) string {
	tokens := b.tokenize(input, context)
	rendered := b.formatTokens(tokens, nil, nil, 1, context)
	return strings.ReplaceAll(rendered, newlineSentinel, b.options.Newline)
}

// Strip removes recognized tags from input using the formatter tokenizer.
func (b *BBGO) Strip(input string, stripNewlines bool) string {
	var out strings.Builder
	for _, token := range b.tokenize(input, nil) {
		if token.kind == tokenText {
			out.WriteString(token.text)
		}
		if token.kind == tokenNewline && !stripNewlines {
			out.WriteString(token.text)
		}
	}
	return out.String()
}

func normalizeParserOptions(options ParserOptions) ParserOptions {
	if options.Newline == "" {
		options.Newline = "<br />"
	}
	if options.MaxTagDepth <= 0 {
		options.MaxTagDepth = 1000
	}
	options.EscapeHTML = true
	options.ReplaceLinks = true
	options.ReplaceCosmetic = true
	return options
}

func normalizeName(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

func defaultTagOptions() TagOptions {
	return TagOptions{
		RenderEmbedded:    true,
		TransformNewlines: true,
		EscapeHTML:        true,
		ReplaceLinks:      true,
		ReplaceCosmetic:   true,
	}
}

func rawTagOptions() TagOptions {
	options := defaultTagOptions()
	options.SameTagCloses = true
	options.RenderEmbedded = false
	options.ReplaceLinks = false
	options.ReplaceCosmetic = false
	return options
}

func (b *BBGO) registerDefaultFormatters() {
	for _, tag := range []string{"b", "i", "u", "s"} {
		b.AddFormatter(tag, simpleTagRenderer(tag), defaultTagOptions())
	}

	b.AddFormatter("color", renderColor, defaultTagOptions())
	b.AddFormatter("quote", renderQuote, defaultTagOptions())
	b.AddFormatter("url", renderURL, urlTagOptions())
	b.AddFormatter("img", renderImage, rawMediaOptions())
	b.AddFormatter("code", renderCode, rawTagOptions())
	b.AddFormatter("list", renderList, defaultTagOptions())

	itemOptions := defaultTagOptions()
	itemOptions.SameTagCloses = true
	b.AddFormatter("*", renderListItem, itemOptions)
}

func rawMediaOptions() TagOptions {
	options := rawTagOptions()
	options.SameTagCloses = false
	return options
}

func urlTagOptions() TagOptions {
	options := defaultTagOptions()
	options.ReplaceLinks = false
	return options
}

func simpleTagRenderer(tag string) RenderFunc {
	return func(ctx RenderContext) string {
		return "<" + tag + ">" + ctx.Value + "</" + tag + ">"
	}
}

func renderColor(ctx RenderContext) string {
	color := strings.ReplaceAll(ctx.Options.Get("color"), ";", "")
	return `<span style="color: ` + escapeAttribute(color) + `;">` + ctx.Value + `</span>`
}

func renderQuote(ctx RenderContext) string {
	cite := firstOption(ctx.Options, "quote", "name")
	if cite == "" {
		return "<blockquote>" + ctx.Value + "</blockquote>"
	}
	return "<blockquote><cite>" + escapeHTML(cite) + " said:</cite>" + ctx.Value + "</blockquote>"
}

func renderURL(ctx RenderContext) string {
	href := firstOption(ctx.Options, "url")
	text := ctx.Value
	if href == "" {
		href = text
	}
	return `<a href="` + escapeAttribute(ensureScheme(href)) + `">` + text + `</a>`
}

func renderImage(ctx RenderContext) string {
	src := strings.TrimSpace(ctx.Value)
	if src == "" {
		return `<img src="">`
	}
	return `<img src="` + escapeAttribute(src) + `">`
}

func renderCode(ctx RenderContext) string {
	return "<pre>" + ctx.Value + "</pre>"
}

func renderList(ctx RenderContext) string {
	if _, ordered := ctx.Options["list"]; ordered {
		return "<ol>" + ctx.Value + "</ol>"
	}
	return "<ul>" + ctx.Value + "</ul>"
}

func renderListItem(ctx RenderContext) string {
	return "<li>" + ctx.Value + "</li>"
}

func firstOption(options Options, names ...string) string {
	for _, name := range names {
		if value, ok := options[name]; ok {
			return value
		}
	}
	return ""
}

func (o Options) Get(name string) string {
	if o == nil {
		return ""
	}
	return o[normalizeName(name)]
}

func normalizeInput(input string) string {
	input = strings.ReplaceAll(input, "\r\n", "\n")
	return strings.ReplaceAll(input, "\r", "\n")
}

func (b *BBGO) tokenize(input string, context Context) []parseToken {
	input = normalizeInput(input)
	tokens := make([]parseToken, 0)
	pos := 0

	for pos < len(input) {
		start := strings.Index(input[pos:], "[")
		if start < 0 {
			tokens = append(tokens, newlineTokens(input[pos:])...)
			break
		}

		start += pos
		if start > pos {
			tokens = append(tokens, newlineTokens(input[pos:start])...)
		}

		end, ok := tagExtent(input, start)
		if !ok {
			tokens = append(tokens, newlineTokens(input[start:end])...)
			pos = end
			continue
		}

		tagText := input[start:end]
		tokens = append(tokens, b.tokenizeTag(input, tagText, start, end, context)...)
		pos = end
	}

	return tokens
}

func tagExtent(input string, start int) (int, bool) {
	inQuote := rune(0)
	quotable := false

	for i := start + 1; i < len(input); i++ {
		ch := rune(input[i])
		if ch == '\n' {
			return i, false
		}
		if ch == '=' {
			quotable = true
		}
		if quoteChanged(ch, quotable, &inQuote) {
			continue
		}
		if inQuote == 0 && ch == '[' {
			return i, false
		}
		if inQuote == 0 && ch == ']' {
			return i + 1, true
		}
	}

	return len(input), false
}

func quoteChanged(ch rune, quotable bool, inQuote *rune) bool {
	if ch != '"' && ch != '\'' {
		return false
	}
	if !quotable && *inQuote == 0 {
		return false
	}
	if *inQuote == 0 {
		*inQuote = ch
		return true
	}
	if *inQuote == ch {
		*inQuote = 0
		return true
	}
	return false
}

func (b *BBGO) tokenizeTag(input string, tagText string, start int, end int, context Context) []parseToken {
	tag, ok := parseTag(tagText)
	if !ok {
		return newlineTokens(tagText)
	}
	if b.isRecognized(tag.tagName) {
		return []parseToken{tag}
	}
	if lineOnlyTag(input, start, end) {
		return b.unknownLineTokens(tagText, context)
	}
	if b.options.DropUnrecognized {
		return nil
	}
	return newlineTokens(tagText)
}

func (b *BBGO) unknownLineTokens(tagText string, context Context) []parseToken {
	if b.options.UnknownLineRenderer == nil {
		return newlineTokens(tagText)
	}
	html, ok := b.options.UnknownLineRenderer(stripBrackets(tagText), context)
	if !ok {
		return newlineTokens(tagText)
	}
	return []parseToken{{kind: tokenStandalone, text: html}}
}

func stripBrackets(tagText string) string {
	return strings.TrimSuffix(strings.TrimPrefix(tagText, "["), "]")
}

func lineOnlyTag(input string, start int, end int) bool {
	before := start == 0 || input[start-1] == '\n'
	after := end >= len(input) || input[end] == '\n'
	return before && after
}

func (b *BBGO) isRecognized(tagName string) bool {
	_, ok := b.formatters[tagName]
	return ok
}

func newlineTokens(input string) []parseToken {
	parts := strings.Split(input, "\n")
	tokens := make([]parseToken, 0, len(parts)*2-1)

	for i, part := range parts {
		if part != "" {
			tokens = append(tokens, parseToken{kind: tokenText, text: part})
		}
		if i < len(parts)-1 {
			tokens = append(tokens, parseToken{kind: tokenNewline, text: "\n"})
		}
	}

	return tokens
}

func parseTag(tagText string) (parseToken, bool) {
	if !strings.HasPrefix(tagText, "[") || !strings.HasSuffix(tagText, "]") {
		return parseToken{}, false
	}

	content := strings.TrimSpace(tagText[1 : len(tagText)-1])
	if content == "" {
		return parseToken{}, false
	}

	if strings.HasPrefix(content, "/") {
		return parseClosingTag(content, tagText)
	}

	name, options := parseOptions(content)
	if name == "" {
		return parseToken{}, false
	}

	return parseToken{
		kind:    tokenStart,
		tagName: normalizeName(name),
		options: options,
		text:    tagText,
	}, true
}

func parseClosingTag(content string, tagText string) (parseToken, bool) {
	name := normalizeName(strings.TrimSpace(content[1:]))
	if name == "" || strings.ContainsAny(name, " =") {
		return parseToken{}, false
	}
	return parseToken{kind: tokenEnd, tagName: name, text: tagText}, true
}

func parseOptions(content string) (string, Options) {
	name, rest := readName(content)
	options := Options{}
	if name == "" {
		return "", options
	}
	if strings.HasPrefix(rest, "=") {
		value, remaining := readValue(rest[1:])
		options[normalizeName(name)] = value
		rest = remaining
	}

	for strings.TrimSpace(rest) != "" {
		key, value, remaining := readOption(rest)
		if key != "" {
			options[normalizeName(key)] = value
		}
		rest = remaining
	}

	return name, options
}

func readName(input string) (string, string) {
	input = strings.TrimLeft(input, " \t")
	for i, r := range input {
		if r == '=' || r == ' ' || r == '\t' {
			return input[:i], input[i:]
		}
	}
	return input, ""
}

func readOption(input string) (string, string, string) {
	key, rest := readName(input)
	if key == "" {
		return "", "", ""
	}
	if !strings.HasPrefix(rest, "=") {
		return key, "", rest
	}
	value, remaining := readValue(rest[1:])
	return key, value, remaining
}

func readValue(input string) (string, string) {
	input = strings.TrimLeft(input, " \t")
	if input == "" {
		return "", ""
	}
	if input[0] == '"' || input[0] == '\'' {
		return readQuotedValue(input)
	}
	return readBareValue(input)
}

func readQuotedValue(input string) (string, string) {
	quote := input[0]
	var out strings.Builder

	for i := 1; i < len(input); i++ {
		if input[i] == '\\' && i+1 < len(input) && escapableQuote(input[i+1]) {
			out.WriteByte(input[i+1])
			i++
			continue
		}
		if input[i] == quote {
			return out.String(), input[i+1:]
		}
		out.WriteByte(input[i])
	}

	return out.String(), ""
}

func escapableQuote(ch byte) bool {
	return ch == '\\' || ch == '"' || ch == '\''
}

func readBareValue(input string) (string, string) {
	for i, r := range input {
		if r == ' ' || r == '\t' {
			return input[:i], input[i:]
		}
	}
	return input, ""
}

func (b *BBGO) formatTokens(tokens []parseToken, parent *TagOptions, state *formatState, depth int, context Context) string {
	if state == nil {
		state = &formatState{}
	}

	var out strings.Builder
	for state.index < len(tokens) {
		token := tokens[state.index]
		out.WriteString(b.formatToken(tokens, token, parent, state, depth, context))
		state.index++
	}
	return out.String()
}

type formatState struct {
	index int
}

func (b *BBGO) formatToken(tokens []parseToken, token parseToken, parent *TagOptions, state *formatState, depth int, context Context) string {
	switch token.kind {
	case tokenStart:
		return b.formatStartTag(tokens, token, parent, state, depth, context)
	case tokenNewline:
		if parent == nil || parent.TransformNewlines {
			return newlineSentinel
		}
		return token.text
	case tokenText:
		return b.transformText(token.text, transformOptionsFor(parent), context)
	case tokenStandalone:
		return token.text
	default:
		return ""
	}
}

func (b *BBGO) formatStartTag(tokens []parseToken, token parseToken, parent *TagOptions, state *formatState, depth int, context Context) string {
	entry, ok := b.formatters[token.tagName]
	if !ok {
		return b.transformText(token.text, transformOptionsFor(parent), context)
	}

	if entry.options.Standalone {
		return entry.render(newRenderContext(token, "", parent, context))
	}

	end, consume := b.findClosingToken(entry.options, tokens, state.index+1)
	inner := b.formatInner(tokens[state.index+1:end], &entry.options, depth, context)
	if entry.options.Strip {
		inner = strings.TrimSpace(inner)
	}

	state.index = nextTokenIndex(end, consume)
	state.index = b.swallowTrailingNewline(tokens, state.index, entry.options)
	return entry.render(newRenderContext(token, inner, parent, context))
}

func newRenderContext(token parseToken, value string, parent *TagOptions, context Context) RenderContext {
	return RenderContext{
		TagName: token.tagName,
		Value:   value,
		Options: token.options,
		Parent:  parent,
		Context: context,
	}
}

func nextTokenIndex(end int, consume bool) int {
	if consume {
		return end
	}
	return end - 1
}

func (b *BBGO) swallowTrailingNewline(tokens []parseToken, index int, options TagOptions) int {
	if !options.SwallowTrailingNewline {
		return index
	}
	next := index + 1
	if next < len(tokens) && tokens[next].kind == tokenNewline {
		return next
	}
	return index
}

func (b *BBGO) findClosingToken(tag TagOptions, tokens []parseToken, start int) (end int, consume bool) {
	embedded := 0

	for i := start; i < len(tokens); i++ {
		token := tokens[i]
		if tag.NewlineCloses && token.kind == tokenNewline {
			return i, true
		}
		if token.kind == tokenStart && token.tagName == tag.TagName {
			if tag.SameTagCloses {
				return i, false
			}
			if tag.RenderEmbedded {
				embedded++
			}
			continue
		}
		if token.kind == tokenEnd && token.tagName == tag.TagName {
			if embedded == 0 {
				return i, true
			}
			embedded--
		}
	}

	return len(tokens), true
}

func (b *BBGO) formatInner(tokens []parseToken, options *TagOptions, depth int, context Context) string {
	if !options.RenderEmbedded || depth >= b.options.MaxTagDepth {
		return b.transformText(rawText(tokens), *options, context)
	}
	state := &formatState{}
	return b.formatTokens(tokens, options, state, depth+1, context)
}

func rawText(tokens []parseToken) string {
	var out strings.Builder
	for _, token := range tokens {
		out.WriteString(token.text)
	}
	return out.String()
}

func transformOptionsFor(parent *TagOptions) TagOptions {
	if parent == nil {
		return defaultTagOptions()
	}
	return *parent
}

func (b *BBGO) transformText(input string, options TagOptions, context Context) string {
	links := map[string]string{}
	if b.options.ReplaceLinks && options.ReplaceLinks {
		input, links = b.replaceLinks(input, context)
	}
	if b.options.EscapeHTML && options.EscapeHTML {
		input = escapeHTML(input)
	}
	if b.options.ReplaceCosmetic && options.ReplaceCosmetic {
		input = replaceCosmetic(input)
	}
	for token, html := range links {
		input = strings.ReplaceAll(input, token, html)
	}
	if options.TransformNewlines {
		input = strings.ReplaceAll(input, "\n", newlineSentinel)
	}
	return input
}

func (b *BBGO) replaceLinks(input string, context Context) (string, map[string]string) {
	matches := autoLinkRE.FindAllStringIndex(input, -1)
	if len(matches) == 0 {
		return input, nil
	}

	replacements := make(map[string]string, len(matches))
	var out strings.Builder
	last := 0

	for i, match := range matches {
		token := "{{ bbcode-link-" + strconv.Itoa(i) + " }}"
		urlText := input[match[0]:match[1]]
		replacements[token] = b.renderLink(urlText, context)
		out.WriteString(input[last:match[0]])
		out.WriteString(token)
		last = match[1]
	}

	out.WriteString(input[last:])
	return out.String(), replacements
}

func (b *BBGO) renderLink(urlText string, context Context) string {
	if b.options.LinkRenderer != nil {
		return b.options.LinkRenderer(urlText, context)
	}
	href := strings.ReplaceAll(ensureScheme(urlText), `"`, "%22")
	return `<a rel="nofollow" href="` + href + `">` + urlText + `</a>`
}

func ensureScheme(input string) string {
	if strings.Contains(input, "://") {
		return input
	}
	return "http://" + input
}

func escapeHTML(input string) string {
	return replaceAll(input, htmlReplacements)
}

func replaceCosmetic(input string) string {
	return replaceAll(input, cosmeticReplacements)
}

func replaceAll(input string, replacements []replacement) string {
	for _, replacement := range replacements {
		input = strings.ReplaceAll(input, replacement.find, replacement.with)
	}
	return input
}

func escapeAttribute(input string) string {
	return strings.ReplaceAll(escapeHTML(input), newlineSentinel, "")
}
