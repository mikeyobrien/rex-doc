package doc

import (
	"strings"

	"github.com/mattn/go-runewidth"
)

// textWidth is the number of terminal cells s takes up.
func textWidth(s string) int { return runewidth.StringWidth(s) }

func spansWidth(spans []Span) int {
	w := 0
	for _, s := range spans {
		w += textWidth(s.Text)
	}
	return w
}

// word is a run of non-space text, possibly in several styles ("**bold**ly").
type word struct {
	pieces []Span
	width  int
	// spaceBefore is true when the word followed whitespace in the source.
	spaceBefore bool
	// spaceStyle is the style of that whitespace, so underlined links keep
	// their underline between words.
	spaceStyle Style
	// breakBefore forces a new line before the word (a hard line break).
	breakBefore bool
}

// splitWords breaks inline spans into words. "\n" in a span is a hard break.
func splitWords(spans []Span) []word {
	var words []word
	var cur *word
	pendingSpace := false
	pendingBreak := false
	var spaceStyle Style
	flush := func() {
		if cur != nil {
			words = append(words, *cur)
			cur = nil
		}
	}
	for _, sp := range spans {
		text := sp.Text
		off := 0
		for len(text) > 0 {
			switch text[0] {
			case ' ', '\t':
				flush()
				pendingSpace = true
				spaceStyle = sp.Style
				text = text[1:]
				off++
				continue
			case '\n':
				flush()
				pendingBreak = true
				pendingSpace = false
				text = text[1:]
				off++
				continue
			}
			n := strings.IndexAny(text, " \t\n")
			if n < 0 {
				n = len(text)
			}
			piece := Span{Text: text[:n], Style: sp.Style, Src: -1}
			if sp.Src >= 0 {
				piece.Src = sp.Src + off
			}
			if cur == nil {
				cur = &word{spaceBefore: pendingSpace, spaceStyle: spaceStyle, breakBefore: pendingBreak}
				pendingSpace, pendingBreak = false, false
			}
			cur.pieces = append(cur.pieces, piece)
			cur.width += textWidth(piece.Text)
			text = text[n:]
			off += n
		}
	}
	flush()
	return words
}

// wrap lays inline spans out in lines no wider than width, including the
// prefix. first is the prefix of the first line and rest the prefix of the
// others (for list bullets, quote bars and indentation).
func wrap(spans []Span, width int, first, rest []Span, blockSrc int) []Line {
	words := splitWords(spans)
	var lines []Line
	prefix := first
	var cur []Span
	curW := 0
	lineSrc := -1
	start := func() {
		cur = append([]Span(nil), prefix...)
		curW = spansWidth(prefix)
		lineSrc = -1
	}
	end := func() {
		if lineSrc < 0 {
			lineSrc = blockSrc
		}
		lines = append(lines, Line{Spans: cur, Src: lineSrc})
		prefix = rest
	}
	start()
	empty := true
	for _, w := range words {
		space := 0
		if !empty && w.spaceBefore {
			space = 1
		}
		if w.breakBefore && !empty {
			end()
			start()
			empty = true
			space = 0
		}
		if !empty && curW+space+w.width > width {
			end()
			start()
			empty = true
			space = 0
		}
		if space == 1 {
			st := w.spaceStyle
			if st.Link < 0 || st.Link != w.pieces[0].Style.Link {
				st = Plain
			}
			cur = append(cur, Span{Text: " ", Style: st, Src: -1})
			curW++
		}
		// A word wider than the whole line is broken by character.
		if w.width > width-spansWidth(prefix) {
			for _, p := range w.pieces {
				for _, r := range p.Text {
					rw := runewidth.RuneWidth(r)
					if curW+rw > width && !empty {
						end()
						start()
					}
					cur = append(cur, Span{Text: string(r), Style: p.Style, Src: p.Src})
					curW += rw
					empty = false
					if lineSrc < 0 && p.Src >= 0 {
						lineSrc = p.Src
					}
				}
			}
			continue
		}
		for _, p := range w.pieces {
			if lineSrc < 0 && p.Src >= 0 {
				lineSrc = p.Src
			}
			cur = append(cur, p)
		}
		curW += w.width
		empty = false
	}
	if !empty || len(lines) == 0 {
		end()
	}
	return lines
}
