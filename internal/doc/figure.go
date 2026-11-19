package doc

import (
	"fmt"
	"hash/fnv"
	"image"
	_ "image/gif" // registered for DecodeConfig
	_ "image/jpeg"
	_ "image/png"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/yuin/goldmark/ast"
	_ "golang.org/x/image/webp"
)

// Figure is an image shown as a figure. Its placeholder cells sit on the
// lines from Line to Line+Rows-1; the terminal draws the image there once
// rdxt has sent it with ID.
type Figure struct {
	ID         uint32
	Path       string
	Cols, Rows int
	Line       int
	Width      int // the image's own size in pixels
	Height     int
}

// Placeholder is kitty's Unicode placeholder character. Each image row
// starts with it plus a row and a column diacritic; the rest of the row
// is the bare character.
const Placeholder = "\U0010EEEE"

// rowDiacritics are the first entries of kitty's row/column diacritics, so
// figures can be at most this many rows tall.
var rowDiacritics = []rune{0x0305, 0x030D, 0x030E, 0x0310, 0x0312, 0x033D, 0x033E, 0x033F,
	0x0346, 0x034A, 0x034B, 0x034C, 0x0350, 0x0351, 0x0352, 0x0357, 0x035B, 0x0363, 0x0364,
	0x0365, 0x0366, 0x0367, 0x0368, 0x0369, 0x036A, 0x036B, 0x036C, 0x036D, 0x036E, 0x036F}

// MaxFigureRows is the tallest a figure can be.
var MaxFigureRows = len(rowDiacritics)

// PlaceholderRow is one row of placeholder cells for an image cols wide.
func PlaceholderRow(r, cols int) string {
	return Placeholder + string(rowDiacritics[r]) + string(rowDiacritics[0]) + strings.Repeat(Placeholder, cols-1)
}

// soloImage returns the image when a paragraph holds nothing but one image.
func soloImage(p ast.Node, src []byte) *ast.Image {
	var img *ast.Image
	for c := p.FirstChild(); c != nil; c = c.NextSibling() {
		switch c := c.(type) {
		case *ast.Image:
			if img != nil {
				return nil
			}
			img = c
		case *ast.Text:
			if strings.TrimSpace(string(c.Segment.Value(src))) != "" {
				return nil
			}
		default:
			return nil
		}
	}
	return img
}

// figure lays out an image on its own line: a frame, the image's cells and
// a numbered caption. Remote, missing and unreadable images get a card.
func (l *layout) figure(img *ast.Image, first, rest []Span) {
	l.figures++
	dest := strings.TrimSpace(string(img.Destination))
	alt := plainText(l.inlines(img, Plain, -1))
	label := alt
	if t := strings.TrimSpace(string(img.Title)); t != "" {
		label = t
	}
	if label == "" {
		label = filepath.Base(dest)
	}
	caption := fmt.Sprintf("Figure %d · %s", l.figures, label)
	src := l.srcOf(img)
	link := l.addLink(dest, string(img.Title))
	l.doc.Links[link].Image = true

	if u, err := url.Parse(dest); err == nil && (u.Scheme == "http" || u.Scheme == "https") {
		l.card(first, rest, label, "on "+u.Host+" · tap to open in the browser", caption, link, src)
		return
	}
	path := dest
	if p, err := url.PathUnescape(path); err == nil {
		path = p
	}
	if !filepath.IsAbs(path) && l.opts.BaseDir != "" {
		path = filepath.Join(l.opts.BaseDir, path)
	}
	f, err := os.Open(path)
	if err != nil {
		l.card(first, rest, "image not found", dest, caption, -1, src)
		return
	}
	cfg, _, err := image.DecodeConfig(f)
	f.Close()
	if err != nil || cfg.Width == 0 || cfg.Height == 0 {
		l.card(first, rest, label, dest+" · can't show this kind of image yet", caption, link, src)
		return
	}
	if !l.opts.Images {
		l.card(first, rest, label, dest, caption, link, src)
		return
	}

	cw, ch := max(l.opts.CellW, 1), max(l.opts.CellH, 1)
	avail := l.avail(rest) - 4
	cols := min(avail, (cfg.Width+cw-1)/cw) // never scaled up
	rows := max(int(float64(cols*cw)*float64(cfg.Height)/float64(cfg.Width)/float64(ch)+0.5), 1)
	maxRows := min(max(l.opts.ScreenRows/3, 4), MaxFigureRows)
	if rows > maxRows {
		rows = maxRows
		cols = max(int(float64(rows*ch)*float64(cfg.Width)/float64(cfg.Height)/float64(cw)+0.5), 1)
	}
	id := figureID(path, cols, rows)
	indent := Span{Text: spaces((avail - cols) / 2), Style: Plain, Src: -1}
	frame := Plain.With(BrightBlack)

	top := append(clone(first), indent, Span{Text: "╭" + strings.Repeat("─", cols+2) + "╮", Style: frame, Src: src})
	l.emit(Line{Spans: top, Src: src})
	start := len(l.doc.Lines)
	for r := 0; r < rows; r++ {
		cells := PlaceholderRow(r, cols)
		st := Style{FG: Default, BG: Default, Link: link, Image: id}
		spans := append(clone(rest), indent,
			Span{Text: "│ ", Style: frame, Src: -1},
			Span{Text: cells, Style: st, Src: -1},
			Span{Text: " │", Style: frame, Src: -1})
		l.emit(Line{Spans: spans, Src: src})
	}
	bottom := append(clone(rest), indent, Span{Text: "╰" + strings.Repeat("─", cols+2) + "╯", Style: frame, Src: -1})
	l.emit(Line{Spans: bottom, Src: -1})
	l.caption(rest, indent, caption, src)
	l.doc.Figures = append(l.doc.Figures, Figure{ID: id, Path: path, Cols: cols, Rows: rows, Line: start,
		Width: cfg.Width, Height: cfg.Height})
}

// card stands in for an image that isn't drawn: a framed box with a title
// and a detail line, then the caption.
func (l *layout) card(first, rest []Span, title, detail, caption string, link, src int) {
	frame := Plain.With(BrightBlack)
	inner := min(max(textWidth(title), textWidth(detail))+2, l.avail(rest)-2)
	ts := Style{FG: Default, BG: Default, Bold: true, Link: link}
	ds := Style{FG: BrightBlack, BG: Default, Link: link}
	row := func(text string, st Style, p []Span) {
		text = truncate(text, inner-1)
		l.emit(Line{Spans: append(clone(p), Span{Text: "│ ", Style: frame, Src: -1},
			Span{Text: text, Style: st, Src: src},
			Span{Text: spaces(inner-1-textWidth(text)) + "│", Style: frame, Src: -1}), Src: src})
	}
	l.emit(Line{Spans: append(clone(first), Span{Text: "╭" + strings.Repeat("─", inner) + "╮", Style: frame, Src: src}), Src: src})
	row(title, ts, rest)
	row(detail, ds, rest)
	l.emit(Line{Spans: append(clone(rest), Span{Text: "╰" + strings.Repeat("─", inner) + "╯", Style: frame, Src: -1}), Src: -1})
	l.caption(rest, Span{Style: Plain, Src: -1}, caption, src)
}

func (l *layout) caption(rest []Span, indent Span, text string, src int) {
	st := Style{FG: BrightBlack, BG: Default, Italic: true, Link: -1}
	p := append(clone(rest), indent, Span{Text: "  ", Style: Plain, Src: -1})
	l.emit(wrap([]Span{{Text: text, Style: st, Src: src}}, l.width, p, p, src)...)
}

// figureID gives a figure a stable 24-bit image ID (never 0) from its file
// and size, since the ID is also the placeholder cells' color.
func figureID(path string, cols, rows int) uint32 {
	h := fnv.New32a()
	fmt.Fprintf(h, "%s|%d|%d", path, cols, rows)
	id := h.Sum32() & 0xFFFFFF
	if id == 0 {
		id = 1
	}
	return id
}
