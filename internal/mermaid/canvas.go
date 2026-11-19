// Package mermaid draws Mermaid diagrams as terminal text: sequence
// diagrams, flowcharts and pie charts. Other kinds return ErrUnsupported so
// the caller can fall back to an image or the source.
package mermaid

import (
	"errors"
	"strings"

	"github.com/mattn/go-runewidth"
)

// Role says what part of a diagram a run of text is, so the caller can
// style it from the terminal's palette.
type Role uint8

const (
	Plain      Role = iota
	Border          // participant and node boxes
	Name            // text inside boxes
	Edge            // edges, arrows, lifelines
	Arrow           // arrowheads
	Label           // text on edges and messages
	Dashed          // dashed arrows (replies)
	Note            // note boxes
	Frame           // loop / alt / opt frames
	FrameLabel      // their keywords and labels
	Decision        // decision (diamond) nodes
	Dim             // lifelines, quiet decoration
	Bar1            // pie chart bars, in order
	Bar2
	Bar3
	Bar4
	Bar5
	Bar6
)

// Run is text in one role.
type Run struct {
	Text string
	Role Role
}

// Line is one row of a diagram.
type Line []Run

var (
	// ErrUnsupported means the diagram kind isn't drawn as text.
	ErrUnsupported = errors.New("mermaid: this kind of diagram isn't drawn as text")
	// ErrTooWide means the diagram doesn't fit the width given.
	ErrTooWide = errors.New("mermaid: the diagram is too wide for this width")
	// ErrEmpty means nothing could be read from the source.
	ErrEmpty = errors.New("mermaid: nothing to draw")
)

// Kind returns the diagram's kind from its first line, like
// "sequenceDiagram" or "flowchart".
func Kind(src string) string {
	for _, l := range strings.Split(src, "\n") {
		l = strings.TrimSpace(l)
		if l == "" || strings.HasPrefix(l, "%%") || l == "---" {
			continue
		}
		f := strings.Fields(l)
		return f[0]
	}
	return ""
}

// KindName is a diagram kind's everyday name, like "Sequence diagram".
func KindName(kind string) string {
	switch kind {
	case "sequenceDiagram":
		return "Sequence diagram"
	case "flowchart", "graph":
		return "Flowchart"
	case "classDiagram":
		return "Class diagram"
	case "stateDiagram", "stateDiagram-v2":
		return "State diagram"
	case "erDiagram":
		return "Entity relationship diagram"
	case "gantt":
		return "Gantt chart"
	case "pie":
		return "Pie chart"
	case "journey":
		return "User journey"
	case "gitGraph":
		return "Git graph"
	case "mindmap":
		return "Mind map"
	case "timeline":
		return "Timeline"
	}
	return "Mermaid diagram"
}

// compact is set while drawing a second, tighter time for narrow panes.
var compact bool

// Render draws src in at most width cells. When the diagram is too wide, it
// tries once more with tighter spacing.
func Render(src string, width int) ([]Line, error) {
	lines, err := render(src, width)
	if errors.Is(err, ErrTooWide) {
		compact = true
		lines, err = render(src, width)
		compact = false
	}
	return lines, err
}

func render(src string, width int) ([]Line, error) {
	var lines []Line
	var err error
	switch Kind(src) {
	case "sequenceDiagram":
		lines, err = sequence(src)
	case "flowchart", "graph":
		lines, err = flowchart(src)
	case "pie":
		lines, err = pie(src, width)
	case "":
		return nil, ErrEmpty
	default:
		return nil, ErrUnsupported
	}
	if err != nil {
		return nil, err
	}
	for _, l := range lines {
		if lineWidth(l) > width {
			return nil, ErrTooWide
		}
	}
	return lines, nil
}

func lineWidth(l Line) int {
	w := 0
	for _, r := range l {
		w += runewidth.StringWidth(r.Text)
	}
	return w
}

func textWidth(s string) int { return runewidth.StringWidth(s) }

// ---- The canvas ----

// Line directions, combined into box-drawing characters.
const (
	up uint8 = 1 << iota
	down
	left
	right
)

type weight uint8

const (
	light weight = iota
	heavy
	dotted
)

type cell struct {
	r    rune // text, or 0
	cont bool // the second half of a wide character
	role Role
	mask uint8
	w    weight
}

type canvas struct {
	cells [][]cell
	w     int
}

func (c *canvas) at(x, y int) *cell {
	if x < 0 || y < 0 {
		return nil
	}
	for len(c.cells) <= y {
		c.cells = append(c.cells, nil)
	}
	row := c.cells[y]
	for len(row) <= x {
		row = append(row, cell{})
	}
	c.cells[y] = row
	if x+1 > c.w {
		c.w = x + 1
	}
	return &row[x]
}

// text writes s at x, y. Wide characters take two cells.
func (c *canvas) text(x, y int, s string, role Role) {
	for _, r := range s {
		w := runewidth.RuneWidth(r)
		if w == 0 {
			continue
		}
		if p := c.at(x, y); p != nil {
			*p = cell{r: r, role: role}
		}
		if w == 2 {
			if p := c.at(x+1, y); p != nil {
				*p = cell{cont: true, role: role}
			}
		}
		x += w
	}
}

// hline and vline add line directions to cells, so crossings and joins
// become the right box-drawing characters.
func (c *canvas) hline(x0, x1, y int, w weight, role Role) {
	if x0 > x1 {
		x0, x1 = x1, x0
	}
	for x := x0; x <= x1; x++ {
		p := c.at(x, y)
		if p == nil || p.r != 0 {
			continue
		}
		fresh := p.mask == 0
		if x > x0 {
			p.mask |= left
		}
		if x < x1 {
			p.mask |= right
		}
		if x0 == x1 {
			p.mask |= left | right
		}
		if fresh {
			p.w = w
		} else {
			p.w = maxWeight(p.w, w)
		}
		p.role = role
	}
}

func (c *canvas) vline(x, y0, y1 int, w weight, role Role) {
	if y0 > y1 {
		y0, y1 = y1, y0
	}
	for y := y0; y <= y1; y++ {
		p := c.at(x, y)
		if p == nil || p.r != 0 {
			continue
		}
		fresh := p.mask == 0
		if y > y0 {
			p.mask |= up
		}
		if y < y1 {
			p.mask |= down
		}
		if y0 == y1 {
			p.mask |= up | down
		}
		if fresh {
			p.w = w
		} else {
			p.w = maxWeight(p.w, w)
		}
		p.role = role
	}
}

func maxWeight(a, b weight) weight {
	if a == dotted || b == dotted {
		if a == heavy || b == heavy {
			return heavy
		}
		if a == light && b == dotted || a == dotted && b == light {
			return light
		}
		return dotted
	}
	if a > b {
		return a
	}
	return b
}

// box draws a rectangle with the given corners.
func (c *canvas) box(x, y, w, h int, corners string, role Role) {
	cs := []rune(corners) // top-left, top-right, bottom-left, bottom-right
	for i := 1; i < w-1; i++ {
		c.text(x+i, y, "─", role)
		c.text(x+i, y+h-1, "─", role)
	}
	for j := 1; j < h-1; j++ {
		c.text(x, y+j, "│", role)
		c.text(x+w-1, y+j, "│", role)
	}
	c.text(x, y, string(cs[0]), role)
	c.text(x+w-1, y, string(cs[1]), role)
	c.text(x, y+h-1, string(cs[2]), role)
	c.text(x+w-1, y+h-1, string(cs[3]), role)
}

var lightChars = map[uint8]rune{
	up: '│', down: '│', up | down: '│', left: '─', right: '─', left | right: '─',
	down | right: '┌', down | left: '┐', up | right: '└', up | left: '┘',
	up | down | right: '├', up | down | left: '┤', down | left | right: '┬', up | left | right: '┴',
	up | down | left | right: '┼',
}

var heavyChars = map[uint8]rune{
	up: '┃', down: '┃', up | down: '┃', left: '━', right: '━', left | right: '━',
	down | right: '┏', down | left: '┓', up | right: '┗', up | left: '┛',
	up | down | right: '┣', up | down | left: '┫', down | left | right: '┳', up | left | right: '┻',
	up | down | left | right: '╋',
}

func lineChar(m uint8, w weight) rune {
	switch w {
	case heavy:
		return heavyChars[m]
	case dotted:
		switch m {
		case up, down, up | down:
			return '┆'
		case left, right, left | right:
			return '┄'
		}
	}
	return lightChars[m]
}

// lines turns the canvas into rows of runs, without trailing spaces.
func (c *canvas) lines() []Line {
	var out []Line
	for _, row := range c.cells {
		var l Line
		var b strings.Builder
		role := Plain
		flush := func() {
			if b.Len() > 0 {
				l = append(l, Run{Text: b.String(), Role: role})
				b.Reset()
			}
		}
		for _, p := range row {
			if p.cont {
				continue
			}
			r, rl := ' ', Plain
			switch {
			case p.r != 0:
				r, rl = p.r, p.role
			case p.mask != 0:
				r, rl = lineChar(p.mask, p.w), p.role
			}
			if rl != role {
				flush()
				role = rl
			}
			b.WriteRune(r)
		}
		flush()
		// Trim trailing spaces.
		for len(l) > 0 {
			last := &l[len(l)-1]
			t := strings.TrimRight(last.Text, " ")
			if t != "" {
				last.Text = t
				break
			}
			l = l[:len(l)-1]
		}
		out = append(out, l)
	}
	return out
}

// cleanText removes Mermaid's markup from a label: quotes, <br> and
// Markdown-string backticks.
func cleanText(s string) []string {
	s = strings.TrimSpace(s)
	s = strings.Trim(s, "\"`")
	for _, br := range []string{"<br/>", "<br />", "<br>", "\\n"} {
		s = strings.ReplaceAll(s, br, "\n")
	}
	s = strings.NewReplacer("**", "", "#quot;", "\"", "#amp;", "&", "&nbsp;", " ").Replace(s)
	var out []string
	for _, l := range strings.Split(s, "\n") {
		out = append(out, strings.TrimSpace(l))
	}
	return out
}
