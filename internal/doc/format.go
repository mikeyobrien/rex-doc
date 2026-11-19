package doc

import (
	"sort"
	"strings"
)

// Overlay changes how part of a line is drawn, without changing the
// document: a search match, the selected link, or a link-hint label.
// Start and End are byte offsets into the line's text (see Line.Text).
type Overlay struct {
	Start, End int
	// Restyle changes the style of the covered text, if set.
	Restyle func(Style) Style
	// Label replaces the start of the covered text, if set (link hints).
	Label      string
	LabelStyle Style
}

// Text returns the line's plain text, without the margin.
func (l Line) Text() string {
	var b strings.Builder
	for _, s := range l.Spans {
		b.WriteString(s.Text)
	}
	return b.String()
}

// Format returns line i as text with escape sequences, including the left
// margin. color false leaves out all styling.
func (d *Doc) Format(i int, color bool) string { return d.FormatWith(i, color, nil) }

// FormatWith is Format with overlays drawn on top.
func (d *Doc) FormatWith(i int, color bool, ovs []Overlay) string {
	var b strings.Builder
	b.WriteString(spaces(d.Margin))
	if i < 0 || i >= len(d.Lines) {
		return b.String()
	}
	sort.SliceStable(ovs, func(a, c int) bool { return ovs[a].Start < ovs[c].Start })
	put := func(text string, st Style) {
		if !color {
			b.WriteString(text)
			return
		}
		sgr := st.SGR()
		b.WriteString(sgr)
		b.WriteString(text)
		if sgr != "" {
			b.WriteString("\x1b[0m")
		}
	}
	// skip counts the cells of underlying text still hidden by a label.
	skip := 0
	off := 0
	for _, s := range d.Lines[i].Spans {
		text := s.Text
		pos := 0
		for pos < len(text) {
			abs := off + pos
			// Find the overlay covering abs, and where the next boundary is.
			end := len(text)
			var cur *Overlay
			for k := range ovs {
				o := &ovs[k]
				if o.Start <= abs && abs < o.End {
					cur = o
					end = min(end, o.End-off)
					break
				}
				if o.Start > abs {
					end = min(end, o.Start-off)
				}
			}
			seg := text[pos:end]
			st := s.Style
			if cur != nil && cur.Restyle != nil {
				st = cur.Restyle(st)
			}
			if cur != nil && cur.Label != "" && abs == cur.Start {
				put(cur.Label, cur.LabelStyle)
				skip = textWidth(cur.Label)
			}
			if skip > 0 {
				// Drop as many cells as the label took.
				for _, r := range seg {
					if skip <= 0 {
						break
					}
					w := textWidth(string(r))
					skip -= w
					seg = seg[len(string(r)):]
				}
			}
			if seg != "" {
				put(seg, st)
			}
			pos = end
		}
		off += len(text)
	}
	return b.String()
}

// LinkRun is where part of a link sits on screen: a line and a byte range in
// that line's text. A link that wraps has one run per line.
type LinkRun struct {
	Line       int
	Start, End int
	Link       int
}

// LinkRuns lists every link run in the document, in reading order.
func (d *Doc) LinkRuns() []LinkRun {
	var runs []LinkRun
	for i, l := range d.Lines {
		off := 0
		for _, s := range l.Spans {
			if s.Style.Link >= 0 {
				n := len(runs)
				if n > 0 && runs[n-1].Line == i && runs[n-1].Link == s.Style.Link && runs[n-1].End == off {
					runs[n-1].End += len(s.Text)
				} else {
					runs = append(runs, LinkRun{Line: i, Start: off, End: off + len(s.Text), Link: s.Style.Link})
				}
			}
			off += len(s.Text)
		}
	}
	return runs
}

// LinkAt returns the link under display column col (0-based, after the
// margin) on line i, or -1.
func (d *Doc) LinkAt(i, col int) int {
	if i < 0 || i >= len(d.Lines) {
		return -1
	}
	x := 0
	for _, s := range d.Lines[i].Spans {
		w := textWidth(s.Text)
		if col >= x && col < x+w {
			return s.Style.Link
		}
		x += w
	}
	return -1
}

// HeadingBySlug returns the line of the heading with the given anchor, or -1.
func (d *Doc) HeadingBySlug(slug string) int {
	for _, h := range d.Headings {
		if strings.EqualFold(h.Slug, slug) {
			return h.Line
		}
	}
	return -1
}
