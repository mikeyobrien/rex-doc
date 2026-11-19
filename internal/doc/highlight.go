package doc

import (
	"strings"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/lexers"
)

// Highlight splits source into lines of styled spans, for Source mode.
func Highlight(code, lang string) [][]Span { return highlight(code, lang) }

// highlight splits code into lines of styled spans. Colors come only from the
// terminal's 16-color palette, so code follows the terminal's theme.
func highlight(code, lang string) [][]Span {
	var lexer chroma.Lexer
	if lang != "" {
		lexer = lexers.Get(lang)
	}
	if lexer == nil {
		var lines [][]Span
		for _, line := range strings.Split(code, "\n") {
			lines = append(lines, []Span{{Text: line, Style: Plain, Src: -1}})
		}
		return lines
	}
	it, err := chroma.Coalesce(lexer).Tokenise(nil, code)
	if err != nil {
		return highlight(code, "")
	}
	lines := [][]Span{nil}
	for tok := it(); tok != chroma.EOF; tok = it() {
		st := tokenStyle(tok.Type)
		parts := strings.Split(tok.Value, "\n")
		for i, p := range parts {
			if i > 0 {
				lines = append(lines, nil)
			}
			if p != "" {
				n := len(lines) - 1
				lines[n] = append(lines[n], Span{Text: p, Style: st, Src: -1})
			}
		}
	}
	// Chroma ends code with a newline; drop the empty last line it makes.
	if len(lines) > 1 && len(lines[len(lines)-1]) == 0 && !strings.HasSuffix(code, "\n") {
		lines = lines[:len(lines)-1]
	}
	return lines
}

func tokenStyle(t chroma.TokenType) Style {
	st := Plain
	switch {
	case t == chroma.KeywordType || t == chroma.NameClass || t == chroma.NameBuiltin:
		st.FG = Cyan
	case t.InCategory(chroma.Keyword) || t == chroma.NameTag:
		st.FG = Magenta
	case t == chroma.NameFunction || t == chroma.NameFunctionMagic:
		st.FG = Blue
	case t == chroma.NameAttribute || t.InCategory(chroma.LiteralNumber):
		st.FG = Yellow
	case t.InCategory(chroma.LiteralString):
		st.FG = Green
	case t.InCategory(chroma.Comment):
		st.FG = BrightBlack
		st.Italic = true
	case t == chroma.GenericInserted:
		st.FG = Green
	case t == chroma.GenericDeleted || t == chroma.Error:
		st.FG = Red
	case t == chroma.GenericHeading:
		st.FG, st.Bold = Magenta, true
	case t == chroma.GenericSubheading:
		st.FG, st.Bold = Blue, true
	case t == chroma.GenericStrong:
		st.Bold = true
	case t == chroma.GenericEmph:
		st.Italic = true
	}
	return st
}
