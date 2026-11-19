package mermaid

import (
	"fmt"
	"regexp"
	"strings"
)

// Sequence diagrams: participants across the top, lifelines down, and
// messages as arrows between them. Notes, loop/alt/opt/par/critical/break
// frames and autonumber are drawn; activations are left out.

type seqEvent struct {
	kind   int // evMsg, evNote, evStart, evElse, evEnd
	from   int
	to     int
	text   string
	dashed bool
	head   string // ">>", ">", "x", ")" or "" (no head)
	both   bool   // <<->>
	pos    string // notes: "left of", "right of", "over"
	word   string // frames: loop, alt, …
}

const (
	evMsg = iota
	evNote
	evStart
	evElse
	evEnd
)

var (
	seqParticipant = regexp.MustCompile(`^(?i)(participant|actor)\s+(.+?)(?:\s+as\s+(.+))?$`)
	seqNote        = regexp.MustCompile(`^(?i)note\s+(left of|right of|over)\s+([^:]+?)\s*:\s*(.*)$`)
	seqStart       = regexp.MustCompile(`^(?i)(loop|alt|opt|par|critical|break|rect)\b\s*(.*)$`)
	seqElse        = regexp.MustCompile(`^(?i)(else|and|option)\b\s*(.*)$`)
	seqMsg         = regexp.MustCompile(`^(.+?)\s*(<<-->>|<<->>|-->>|->>|--x|-x|--\)|-\)|-->|->)\s*[+-]?\s*(.+?)\s*:\s*(.*)$`)
)

type seqDiagram struct {
	names  []string // ids in order
	labels map[string]string
	index  map[string]int
	events []seqEvent
	title  string
	number bool
}

func (d *seqDiagram) participant(id string) int {
	id = strings.TrimSpace(id)
	if i, ok := d.index[id]; ok {
		return i
	}
	d.index[id] = len(d.names)
	d.names = append(d.names, id)
	if _, ok := d.labels[id]; !ok {
		d.labels[id] = id
	}
	return d.index[id]
}

func parseSequence(src string) (*seqDiagram, error) {
	d := &seqDiagram{labels: map[string]string{}, index: map[string]int{}}
	depth := 0
	for _, raw := range strings.Split(src, "\n") {
		l := strings.TrimSpace(raw)
		if i := strings.Index(l, "%%"); i >= 0 {
			l = strings.TrimSpace(l[:i])
		}
		low := strings.ToLower(l)
		switch {
		case l == "" || low == "sequencediagram":
		case strings.HasPrefix(low, "title"):
			d.title = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(l[5:]), ":"))
		case low == "autonumber" || strings.HasPrefix(low, "autonumber "):
			d.number = true
		case strings.HasPrefix(low, "activate") || strings.HasPrefix(low, "deactivate") ||
			strings.HasPrefix(low, "destroy") || strings.HasPrefix(low, "create ") ||
			strings.HasPrefix(low, "box") || strings.HasPrefix(low, "link"):
		case seqParticipant.MatchString(l):
			m := seqParticipant.FindStringSubmatch(l)
			id := strings.TrimSpace(m[2])
			if m[3] != "" {
				d.labels[id] = strings.Join(cleanText(m[3]), " ")
			}
			d.participant(id)
		case seqNote.MatchString(l):
			m := seqNote.FindStringSubmatch(l)
			who := strings.Split(m[2], ",")
			e := seqEvent{kind: evNote, pos: strings.ToLower(m[1]), text: strings.Join(cleanText(m[3]), " ")}
			e.from = d.participant(who[0])
			e.to = e.from
			if len(who) > 1 {
				e.to = d.participant(who[1])
			}
			d.events = append(d.events, e)
		case low == "end":
			if depth > 0 {
				depth--
				d.events = append(d.events, seqEvent{kind: evEnd})
			}
		case seqStart.MatchString(l):
			m := seqStart.FindStringSubmatch(l)
			text := m[2]
			if strings.EqualFold(m[1], "rect") {
				text = ""
			}
			depth++
			d.events = append(d.events, seqEvent{kind: evStart, word: strings.ToLower(m[1]), text: text})
		case seqElse.MatchString(l) && depth > 0:
			m := seqElse.FindStringSubmatch(l)
			d.events = append(d.events, seqEvent{kind: evElse, word: strings.ToLower(m[1]), text: m[2]})
		case seqMsg.MatchString(l):
			m := seqMsg.FindStringSubmatch(l)
			op := m[2]
			e := seqEvent{kind: evMsg, from: d.participant(m[1]), to: d.participant(m[3]),
				text: strings.Join(cleanText(m[4]), " ")}
			e.dashed = strings.HasPrefix(op, "--") || strings.HasPrefix(op, "<<--")
			switch {
			case strings.HasPrefix(op, "<<"):
				e.head, e.both = ">>", true
			case strings.HasSuffix(op, ">>"):
				e.head = ">>"
			case strings.HasSuffix(op, "x"):
				e.head = "x"
			case strings.HasSuffix(op, ")"):
				e.head = ")"
			}
			d.events = append(d.events, e)
		}
	}
	for ; depth > 0; depth-- {
		d.events = append(d.events, seqEvent{kind: evEnd})
	}
	if len(d.names) == 0 {
		return nil, ErrEmpty
	}
	return d, nil
}

func sequence(src string) ([]Line, error) {
	d, err := parseSequence(src)
	if err != nil {
		return nil, err
	}
	n := len(d.names)
	// Box widths and the deepest frame nesting.
	bw := make([]int, n)
	for i, id := range d.names {
		bw[i] = max(textWidth(d.labels[id])+4, 7)
	}
	maxDepth, depth := 0, 0
	num := 0
	labels := make([]string, len(d.events))
	for i, e := range d.events {
		switch e.kind {
		case evStart:
			depth++
			maxDepth = max(maxDepth, depth)
		case evEnd:
			depth--
		case evMsg:
			labels[i] = e.text
			if d.number {
				num++
				labels[i] = fmt.Sprintf("%d. %s", num, e.text)
			}
		}
	}
	// Space between neighbouring participants (gap[i] is from i to i+1).
	gap := make([]int, max(n-1, 0))
	pad := 4
	if compact {
		pad = 1
	}
	for i := range gap {
		gap[i] = (bw[i]+bw[i+1])/2 + pad
	}
	rightExtra, leftExtra := 0, 0
	need := func(lo, hi, w int) {
		if lo == hi {
			return
		}
		sum := 0
		for k := lo; k < hi; k++ {
			sum += gap[k]
		}
		if sum < w {
			gap[hi-1] += w - sum
		}
	}
	for i, e := range d.events {
		switch e.kind {
		case evMsg:
			w := textWidth(labels[i]) + 2 + pad/2
			if e.from == e.to {
				if e.from < n-1 {
					need(e.from, e.from+1, w+2)
				} else {
					rightExtra = max(rightExtra, w+2)
				}
				continue
			}
			need(min(e.from, e.to), max(e.from, e.to), w)
		case evNote:
			w := textWidth(e.text) + 4
			switch {
			case e.pos == "right of" && e.from < n-1:
				need(e.from, e.from+1, w+3)
			case e.pos == "right of":
				rightExtra = max(rightExtra, w+2)
			case e.pos == "left of" && e.from > 0:
				need(e.from-1, e.from, w+3)
			case e.pos == "left of":
				leftExtra = max(leftExtra, w+2)
			}
		case evStart, evElse:
			// Frame labels need room across the whole diagram.
			w := textWidth(e.word+" "+e.text) + 8
			if n > 1 {
				need(0, n-1, w)
			} else {
				rightExtra = max(rightExtra, w)
			}
		}
	}
	frameRoom := maxDepth * 2
	margin := max(bw[0]/2+frameRoom+1, leftExtra+1)
	cx := make([]int, n)
	cx[0] = margin
	for i := 1; i < n; i++ {
		cx[i] = cx[i-1] + gap[i-1]
	}
	right := cx[n-1] + max(bw[n-1]/2+frameRoom+1, rightExtra)
	width := right + 1

	c := &canvas{}
	y := 0
	if d.title != "" {
		c.text(max((width-textWidth(d.title))/2, 0), y, d.title, Name)
		y += 2
	}
	drawBoxes := func(y int) {
		for i, id := range d.names {
			x := cx[i] - bw[i]/2
			c.box(x, y, bw[i], 3, "╭╮╰╯", Border)
			label := d.labels[id]
			c.text(cx[i]-textWidth(label)/2, y+1, label, Name)
		}
	}
	drawBoxes(y)
	top := y + 3
	y = top
	type open struct{ y, depth int }
	var frames []open
	depth = 0
	for i, e := range d.events {
		switch e.kind {
		case evMsg:
			label := labels[i]
			if e.from == e.to {
				x := cx[e.from]
				c.text(x+2, y, label, Label)
				role := Edge
				if e.dashed {
					role = Dashed
				}
				c.text(x+1, y+1, "───┐", role)
				c.text(x+4, y+2, "│", role)
				c.text(x+1, y+3, "◀──┘", role)
				y += 5
				continue
			}
			a, b := cx[e.from], cx[e.to]
			lo, hi := min(a, b), max(a, b)
			c.text(lo+(hi-lo-textWidth(label))/2, y, label, Label)
			ch, role := "─", Edge
			if e.dashed {
				ch, role = "╌", Dashed
			}
			c.text(lo+1, y+1, strings.Repeat(ch, hi-lo-1), role)
			head := map[string]string{">>": "▶", "x": "×", ")": ">", "": ch}[e.head]
			back := map[string]string{">>": "◀", "x": "×", ")": "<", "": ch}[e.head]
			if b > a {
				c.text(b-1, y+1, head, Arrow)
				if e.both {
					c.text(a+1, y+1, back, Arrow)
				}
			} else {
				c.text(b+1, y+1, back, Arrow)
				if e.both {
					c.text(a-1, y+1, head, Arrow)
				}
			}
			y += 3
		case evNote:
			w := textWidth(e.text) + 4
			var x int
			switch e.pos {
			case "right of":
				x = cx[e.from] + 2
			case "left of":
				x = cx[e.from] - 2 - w
			default:
				lo, hi := min(cx[e.from], cx[e.to]), max(cx[e.from], cx[e.to])
				if hi-lo+4 > w {
					w = hi - lo + 4
				}
				x = (lo+hi)/2 - w/2
			}
			c.box(x, y, w, 3, "┌┐└┘", Note)
			c.text(x+1, y+1, strings.Repeat(" ", w-2), Note)
			c.text(x+2, y+1, e.text, Note)
			y += 4
		case evStart:
			l, r := depth*2, width-1-depth*2
			c.text(l, y, "┌"+strings.Repeat("─", r-l-1)+"┐", Frame)
			c.text(l+2, y, " "+e.word+" ", FrameLabel)
			if e.text != "" {
				c.text(l+3+textWidth(e.word)+1, y, e.text+" ", Label)
			}
			frames = append(frames, open{y, depth})
			depth++
			y += 2
		case evElse:
			if depth == 0 {
				continue
			}
			l, r := (depth-1)*2, width-1-(depth-1)*2
			c.text(l, y, "├"+strings.Repeat("┄", r-l-1)+"┤", Frame)
			c.text(l+2, y, " "+e.word+" ", FrameLabel)
			if e.text != "" {
				c.text(l+3+textWidth(e.word)+1, y, e.text+" ", Label)
			}
			y += 2
		case evEnd:
			if len(frames) == 0 {
				continue
			}
			f := frames[len(frames)-1]
			frames = frames[:len(frames)-1]
			depth--
			l, r := f.depth*2, width-1-f.depth*2
			for yy := f.y + 1; yy < y; yy++ {
				if p := c.at(l, yy); p != nil && p.r == 0 {
					c.text(l, yy, "│", Frame)
				}
				if p := c.at(r, yy); p != nil && p.r == 0 {
					c.text(r, yy, "│", Frame)
				}
			}
			c.text(l, y, "└"+strings.Repeat("─", r-l-1)+"┘", Frame)
			y += 2
		}
	}
	// Lifelines under everything else.
	for i := range d.names {
		for yy := top; yy < y; yy++ {
			if p := c.at(cx[i], yy); p != nil && p.r == 0 && p.mask == 0 {
				c.text(cx[i], yy, "┊", Dim)
			}
		}
	}
	// Repeat the participants at the bottom of a long diagram.
	if y-top > 14 {
		drawBoxes(y)
	}
	lines := c.lines()
	for len(lines) > 0 && len(lines[len(lines)-1]) == 0 {
		lines = lines[:len(lines)-1]
	}
	return lines, nil
}
