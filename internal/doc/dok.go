package doc

import (
	"regexp"
	"strings"
)

// Dok (.dok) is a plain-text format: ALL CAPS section headers alone on a
// line, content indented two spaces, "--" comment lines, dash list items and
// backticks for literal values. See https://github.com/travelingflwr/dok.
//
// Dok is meant to read well as plain text, so the renderer keeps its line
// structure and aligned columns. It only styles headers, list markers,
// two-column terms and backticks, and wraps lines that are too long.

var (
	dokHeader = regexp.MustCompile(`^[A-Z][A-Z0-9 &/_'-]*:?\s*$`)
	// A term, two or more spaces, then its description ("Simple   The …").
	dokColumns = regexp.MustCompile(`^(\S(?:.*?\S)?)( {2,})(\S.*)$`)
)

func renderDok(src []byte, d *Doc) {
	l := &layout{src: src, width: d.Width, doc: d, slugs: map[string]int{}}
	lines := strings.SplitAfter(string(src), "\n")
	off := 0
	sections := 0
	blank := false
	for _, raw := range lines {
		start := off
		off += len(raw)
		line := strings.TrimRight(raw, "\r\n")
		trimmed := strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(line, "--"):
			continue // a comment
		case trimmed == "":
			blank = true
			continue
		case !strings.HasPrefix(line, " ") && dokHeader.MatchString(line) && len(trimmed) > 1:
			if sections > 0 {
				l.emit(Line{Src: -1}, Line{Src: -1})
			}
			sections++
			l.dokHeader(strings.TrimSuffix(trimmed, ":"), start)
			l.emit(Line{Src: -1})
			blank = false
			continue
		}
		if blank && len(d.Lines) > 0 && !isBlank(d.Lines[len(d.Lines)-1]) {
			l.emit(Line{Src: -1})
		}
		blank = false
		// Content is indented two spaces; deeper indents are kept.
		indent := len(line) - len(strings.TrimLeft(line, " "))
		if sections > 0 {
			indent = max(indent-2, 0)
		}
		l.dokLine(trimmed, indent, start+len(line)-len(strings.TrimLeft(line, " ")))
	}
	for len(d.Lines) > 0 && isBlank(d.Lines[len(d.Lines)-1]) {
		d.Lines = d.Lines[:len(d.Lines)-1]
	}
}

func (l *layout) dokHeader(name string, src int) {
	st := headingStyles[2]
	l.doc.Headings = append(l.doc.Headings, Heading{Level: 2, Text: name, Slug: l.slug(name), Line: len(l.doc.Lines)})
	lines := wrap([]Span{{Text: name, Style: st, Src: src}}, l.width, nil, nil, src)
	for i := range lines {
		lines[i].Heading = 2
	}
	l.emit(lines...)
	l.emit(Line{Spans: []Span{{Text: strings.Repeat("─", l.width), Style: Plain.With(BrightBlack), Src: -1}}, Src: -1})
}

// dokLine lays out one content line: a list item, a two-column row, or prose.
func (l *layout) dokLine(text string, indent, src int) {
	pad := Span{Text: spaces(indent), Style: Plain, Src: -1}
	for _, marker := range []string{"-- ", "— ", "- "} {
		if strings.HasPrefix(text, marker) {
			body := strings.TrimSpace(text[len(marker):])
			bullet := Span{Text: "•", Style: Plain.With(BrightBlack), Src: -1}
			first := []Span{pad, bullet, {Text: " ", Style: Plain, Src: -1}}
			rest := []Span{{Text: spaces(indent + 2), Style: Plain, Src: -1}}
			l.emit(wrap(dokInline(body, src+len(marker)), l.width, first, rest, src)...)
			return
		}
	}
	if m := dokColumns.FindStringSubmatchIndex(text); m != nil && indent == 0 {
		term := text[m[2]:m[3]]
		gap := text[m[4]:m[5]]
		desc := text[m[6]:m[7]]
		col := indent + textWidth(term) + len(gap)
		if col <= l.width/2 {
			termStyle := Style{FG: Default, BG: Default, Bold: true, Link: -1}
			first := []Span{pad, {Text: term, Style: termStyle, Src: src}, {Text: gap, Style: Plain, Src: -1}}
			rest := []Span{{Text: spaces(col), Style: Plain, Src: -1}}
			l.emit(wrap(dokInline(desc, src+m[6]), l.width, first, rest, src)...)
			return
		}
	}
	rest := []Span{{Text: spaces(indent), Style: Plain, Src: -1}}
	l.emit(wrap(dokInline(text, src), l.width, []Span{pad}, rest, src)...)
}

// dokInline styles `backticks` as inline code. Nothing else is markup in Dok.
func dokInline(text string, src int) []Span {
	var out []Span
	code := Style{FG: Cyan, BG: Default, Link: -1}
	for {
		i := strings.IndexByte(text, '`')
		if i < 0 {
			break
		}
		j := strings.IndexByte(text[i+1:], '`')
		if j < 0 {
			break
		}
		if i > 0 {
			out = append(out, Span{Text: text[:i], Style: Plain, Src: src})
		}
		lit := text[i+1 : i+1+j]
		out = append(out, Span{Text: strings.ReplaceAll(lit, " ", " "), Style: code, Src: src + i + 1})
		text = text[i+j+2:]
		src += i + j + 2
	}
	if text != "" {
		out = append(out, Span{Text: text, Style: Plain, Src: src})
	}
	return out
}
