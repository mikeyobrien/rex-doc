package mermaid

import (
	"regexp"
	"sort"
	"strings"
)

// Flowcharts are laid out in layers (a simple Sugiyama layout): each node
// goes one layer below the nodes that point to it, long edges get
// pass-through points in the layers they cross, nodes are ordered to reduce
// crossings, and edges are routed with right angles in the gaps between
// layers. LR and RL are laid out top-down and turned on their side.

type fnode struct {
	id    string
	lines []string
	shape string // "rect", "round", "decision", "circle", "stadium", "sub"
	dummy bool
	// Layout, in top-down space.
	layer, order int
	x, w, h      int // x is the left edge
}

type fedge struct {
	from, to int
	label    string
	w        weight
	head     bool // an arrowhead at the target
	both     bool // and at the source
	reversed bool // a back edge, drawn against the layer direction
}

type flow struct {
	nodes []*fnode
	index map[string]int
	edges []fedge
	dir   string
}

var (
	flowHeader = regexp.MustCompile(`^(?i)(flowchart|graph)\s*(TD|TB|BT|LR|RL)?`)
	nodeID     = regexp.MustCompile(`^[\p{L}\p{N}_][\p{L}\p{N}_\-.]*`)
	edgeText   = regexp.MustCompile(`^\s*(<)?(--|==|-\.)\s+([^\s].*?)\s+(-{2,}|={2,}|\.+-)([>ox]?)`)
	edgePlain  = regexp.MustCompile(`^\s*([<ox])?(-{2,}|={2,}|-\.+-)([>ox])?\s*(?:\|([^|]*)\|)?`)
)

// Shapes: opener, closer, shape.
var shapes = []struct{ open, close, shape string }{
	{"(((", ")))", "circle"}, {"((", "))", "circle"}, {"([", "])", "stadium"}, {"[[", "]]", "sub"},
	{"[(", ")]", "stadium"}, {"{{", "}}", "decision"}, {"[/", "/]", "rect"}, {"[\\", "\\]", "rect"},
	{"[/", "\\]", "rect"}, {"[\\", "/]", "rect"}, {"[", "]", "rect"}, {"(", ")", "round"},
	{"{", "}", "decision"}, {">", "]", "rect"},
}

func (f *flow) node(id string) int {
	if i, ok := f.index[id]; ok {
		return i
	}
	f.index[id] = len(f.nodes)
	f.nodes = append(f.nodes, &fnode{id: id, lines: []string{id}, shape: "rect"})
	return f.index[id]
}

// parseNodes reads "A", "A[Text]" or "A & B{Text}" at the start of s.
func (f *flow) parseNodes(s string) ([]int, string) {
	var ids []int
	for {
		s = strings.TrimLeft(s, " \t")
		id := nodeID.FindString(s)
		if id == "" {
			return ids, s
		}
		s = s[len(id):]
		i := f.node(id)
		for _, sh := range shapes {
			if strings.HasPrefix(s, sh.open) {
				end := strings.Index(s[len(sh.open):], sh.close)
				if end < 0 {
					break
				}
				text := s[len(sh.open) : len(sh.open)+end]
				f.nodes[i].lines = cleanText(text)
				f.nodes[i].shape = sh.shape
				s = s[len(sh.open)+end+len(sh.close):]
				break
			}
		}
		// Class names: A:::important
		if strings.HasPrefix(s, ":::") {
			s = strings.TrimLeft(s[3:], "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_-")
		}
		ids = append(ids, i)
		t := strings.TrimLeft(s, " \t")
		if !strings.HasPrefix(t, "&") {
			return ids, s
		}
		s = t[1:]
	}
}

func parseFlow(src string) (*flow, error) {
	f := &flow{index: map[string]int{}, dir: "TD"}
	for _, raw := range strings.Split(src, "\n") {
		if i := strings.Index(raw, "%%"); i >= 0 {
			raw = raw[:i]
		}
		for _, stmt := range strings.Split(raw, ";") {
			l := strings.TrimSpace(stmt)
			low := strings.ToLower(l)
			if l == "" {
				continue
			}
			if m := flowHeader.FindStringSubmatch(l); m != nil && len(f.nodes) == 0 && len(f.edges) == 0 && strings.EqualFold(l[:len(m[1])], m[1]) && (len(l) == len(m[0]) || l[len(m[0])] == ' ') {
				if m[2] != "" {
					f.dir = strings.ToUpper(m[2])
				}
				if f.dir == "TB" {
					f.dir = "TD"
				}
				continue
			}
			for _, kw := range []string{"classdef ", "class ", "style ", "linkstyle ", "click ", "direction "} {
				if strings.HasPrefix(low, kw) {
					low = "skip"
				}
			}
			if low == "skip" || low == "end" || strings.HasPrefix(low, "subgraph") {
				continue
			}
			f.parseChain(l)
		}
	}
	if len(f.nodes) == 0 {
		return nil, ErrEmpty
	}
	return f, nil
}

// parseChain reads "A --> B -->|yes| C & D".
func (f *flow) parseChain(s string) {
	from, rest := f.parseNodes(s)
	for len(from) > 0 {
		var e fedge
		if m := edgeText.FindStringSubmatch(rest); m != nil {
			e = fedge{label: strings.Join(cleanText(m[3]), " "), head: m[5] == ">", both: m[1] == "<"}
			e.w = edgeWeight(m[2] + m[4])
			rest = rest[len(m[0]):]
		} else if m := edgePlain.FindStringSubmatch(rest); m != nil && m[0] != "" {
			e = fedge{label: strings.Join(cleanText(m[4]), " "), head: m[3] == ">", both: m[1] == "<"}
			e.w = edgeWeight(m[2])
			rest = rest[len(m[0]):]
		} else {
			return
		}
		to, r := f.parseNodes(rest)
		rest = r
		for _, a := range from {
			for _, b := range to {
				e.from, e.to = a, b
				f.edges = append(f.edges, e)
			}
		}
		from = to
	}
}

func edgeWeight(op string) weight {
	switch {
	case strings.Contains(op, "="):
		return heavy
	case strings.Contains(op, "."):
		return dotted
	}
	return light
}

func flowchart(src string) ([]Line, error) {
	f, err := parseFlow(src)
	if err != nil {
		return nil, err
	}
	sideways := f.dir == "LR" || f.dir == "RL"
	// Node sizes in top-down space (sideways layouts swap them).
	for _, n := range f.nodes {
		tw := 0
		for _, l := range n.lines {
			tw = max(tw, textWidth(l))
		}
		w, h := tw+4, len(n.lines)+2
		if sideways {
			w, h = h, w
		}
		n.w, n.h = w, h
	}
	f.layers()
	segs := f.split()
	byLayer := f.order(segs)
	f.place(byLayer, segs)
	c, boxes, labels := f.route(byLayer, segs)
	// The layout's full size, boxes included, in top-down space.
	W, H := c.w, len(c.cells)
	for _, b := range boxes {
		W, H = max(W, b.x+b.w), max(H, b.y+b.h)
	}
	if sideways {
		c = transpose(c)
		W, H = H, W
	}
	if f.dir == "BT" {
		c = flipV(c, H)
	}
	if f.dir == "RL" {
		c = flipH(c, W)
	}
	// Boxes and text go on last, in the final orientation.
	for _, b := range boxes {
		x, y, w, h := b.x, b.y, b.w, b.h
		if sideways {
			x, y, w, h = b.y, b.x, b.h, b.w
		}
		if f.dir == "BT" {
			y = H - y - h
		}
		if f.dir == "RL" {
			x = W - x - w
		}
		drawNode(c, b.n, x, y, w, h)
	}
	for _, lb := range labels {
		x, y := lb.x, lb.y
		if sideways {
			// In a sideways layout the label sits just above its edge.
			x, y = lb.y, lb.x-1
		} else {
			x -= textWidth(lb.text) / 2
		}
		if f.dir == "BT" {
			y = H - 1 - y
		}
		if f.dir == "RL" {
			x = W - x - textWidth(lb.text)
		}
		c.text(x, y, lb.text, Label)
	}
	return c.lines(), nil
}

// layers puts each node one layer below its parents, after turning back
// edges around so the graph has no cycles.
func (f *flow) layers() {
	n := len(f.nodes)
	adj := make([][]int, n)
	for i, e := range f.edges {
		if e.from != e.to {
			adj[e.from] = append(adj[e.from], i)
		}
	}
	state := make([]int, n) // 0 new, 1 on the stack, 2 done
	var visit func(v int)
	visit = func(v int) {
		state[v] = 1
		for _, ei := range adj[v] {
			w := f.edges[ei].to
			if state[w] == 1 {
				f.edges[ei].reversed = true
			} else if state[w] == 0 {
				visit(w)
			}
		}
		state[v] = 2
	}
	for v := 0; v < n; v++ {
		if state[v] == 0 {
			visit(v)
		}
	}
	// Longest path from the roots, in the edges' (fixed) direction.
	for changed := true; changed; {
		changed = false
		for _, e := range f.edges {
			if e.from == e.to {
				continue
			}
			a, b := e.from, e.to
			if e.reversed {
				a, b = b, a
			}
			if f.nodes[b].layer < f.nodes[a].layer+1 {
				f.nodes[b].layer = f.nodes[a].layer + 1
				changed = true
			}
		}
	}
}

// seg is an edge between neighbouring layers, from upper to lower.
type seg struct {
	up, down int // node indexes
	edge     int
	last     bool // the segment that reaches the edge's lower end
	first    bool // the segment that leaves the edge's upper end
}

// split breaks long edges into segments through pass-through nodes.
func (f *flow) split() []seg {
	var segs []seg
	for ei, e := range f.edges {
		if e.from == e.to {
			continue
		}
		a, b := e.from, e.to
		if e.reversed {
			a, b = b, a
		}
		prev := a
		for l := f.nodes[a].layer + 1; l < f.nodes[b].layer; l++ {
			d := &fnode{id: "", dummy: true, layer: l, w: 1, h: 0}
			f.nodes = append(f.nodes, d)
			segs = append(segs, seg{up: prev, down: len(f.nodes) - 1, edge: ei, first: prev == a})
			prev = len(f.nodes) - 1
		}
		segs = append(segs, seg{up: prev, down: b, edge: ei, first: prev == a, last: true})
	}
	return segs
}

// order sorts each layer to reduce crossings, by the average position of
// each node's neighbours, sweeping down and up a few times.
func (f *flow) order(segs []seg) [][]int {
	maxL := 0
	for _, n := range f.nodes {
		maxL = max(maxL, n.layer)
	}
	byLayer := make([][]int, maxL+1)
	for i, n := range f.nodes {
		byLayer[n.layer] = append(byLayer[n.layer], i)
	}
	pos := make([]float64, len(f.nodes))
	set := func() {
		for _, l := range byLayer {
			for k, i := range l {
				pos[i] = float64(k)
				f.nodes[i].order = k
			}
		}
	}
	set()
	ups := make([][]int, len(f.nodes))
	downs := make([][]int, len(f.nodes))
	for _, s := range segs {
		downs[s.up] = append(downs[s.up], s.down)
		ups[s.down] = append(ups[s.down], s.up)
	}
	sortBy := func(l []int, nb [][]int) {
		key := map[int]float64{}
		for _, i := range l {
			if len(nb[i]) == 0 {
				key[i] = pos[i]
				continue
			}
			sum := 0.0
			for _, j := range nb[i] {
				sum += pos[j]
			}
			key[i] = sum / float64(len(nb[i]))
		}
		sort.SliceStable(l, func(a, b int) bool { return key[l[a]] < key[l[b]] })
	}
	for pass := 0; pass < 4; pass++ {
		for li := 1; li < len(byLayer); li++ {
			sortBy(byLayer[li], ups)
			set()
		}
		for li := len(byLayer) - 2; li >= 0; li-- {
			sortBy(byLayer[li], downs)
			set()
		}
	}
	return byLayer
}

func (n *fnode) cx() int { return n.x + n.w/2 }

// gapX is the space between boxes in a layer.
func gapX() int {
	if compact {
		return 1
	}
	return 3
}

// place gives nodes x positions: each layer packed, then nodes pulled
// toward the middle of their neighbours without overlapping.
func (f *flow) place(byLayer [][]int, segs []seg) {
	for _, l := range byLayer {
		x := 0
		for _, i := range l {
			f.nodes[i].x = x
			x += f.nodes[i].w + gapX()
		}
	}
	ups := make([][]int, len(f.nodes))
	downs := make([][]int, len(f.nodes))
	for _, s := range segs {
		downs[s.up] = append(downs[s.up], s.down)
		ups[s.down] = append(ups[s.down], s.up)
	}
	align := func(l []int, nb [][]int) {
		want := make([]int, len(l))
		for k, i := range l {
			n := f.nodes[i]
			want[k] = n.x
			if len(nb[i]) > 0 {
				sum := 0
				for _, j := range nb[i] {
					sum += f.nodes[j].cx()
				}
				want[k] = sum/len(nb[i]) - n.w/2
			}
		}
		// Left to right, keeping the gaps.
		prevRight := -1 << 30
		for k, i := range l {
			n := f.nodes[i]
			n.x = max(want[k], prevRight+gapX())
			prevRight = n.x + n.w
		}
		// Right to left, pulling back toward what each wanted.
		for k := len(l) - 2; k >= 0; k-- {
			n, next := f.nodes[l[k]], f.nodes[l[k+1]]
			if limit := next.x - gapX() - n.w; n.x > limit {
				n.x = limit
			} else if n.x < want[k] {
				n.x = min(want[k], limit)
			}
		}
	}
	for pass := 0; pass < 3; pass++ {
		for li := 1; li < len(byLayer); li++ {
			align(byLayer[li], ups)
		}
		for li := len(byLayer) - 2; li >= 0; li-- {
			align(byLayer[li], downs)
		}
	}
	minX := 1 << 30
	for _, n := range f.nodes {
		minX = min(minX, n.x)
	}
	for _, n := range f.nodes {
		n.x -= minX
	}
}

type placedBox struct {
	n          *fnode
	x, y, w, h int
}

type placedLabel struct {
	text string
	x, y int // top-down space: x is the edge's column, y the label row
}

// route draws the edges in the gaps between layers and returns where the
// boxes and labels go.
func (f *flow) route(byLayer [][]int, segs []seg) (*canvas, []placedBox, []placedLabel) {
	c := &canvas{}
	sideways := f.dir == "LR" || f.dir == "RL"
	// Layer heights, and the gap below each layer.
	bandH := make([]int, len(byLayer))
	for li, l := range byLayer {
		for _, i := range l {
			bandH[li] = max(bandH[li], f.nodes[i].h)
		}
		bandH[li] = max(bandH[li], 1)
	}
	type gapInfo struct {
		tracks map[int]int // upper node → track number
		label  int         // rows for labels
	}
	gaps := make([]gapInfo, len(byLayer))
	for li := range gaps {
		gaps[li].tracks = map[int]int{}
	}
	// A track is shared by the edges leaving one node in one line style.
	trackKey := func(s seg) int { return s.up*4 + int(f.edges[s.edge].w) }
	for _, s := range segs {
		u, d := f.nodes[s.up], f.nodes[s.down]
		g := &gaps[u.layer]
		if u.cx() != d.cx() {
			if _, ok := g.tracks[trackKey(s)]; !ok {
				g.tracks[trackKey(s)] = len(g.tracks)
			}
		}
		if s.last && f.edges[s.edge].label != "" {
			lw := 1
			if sideways {
				lw = textWidth(f.edges[s.edge].label) + 1
			}
			g.label = max(g.label, lw)
		}
	}
	// Order tracks by the upper node's position, so lines cross less.
	for li := range gaps {
		var ups []int
		for u := range gaps[li].tracks {
			ups = append(ups, u)
		}
		sort.Slice(ups, func(a, b int) bool {
			if f.nodes[ups[a]/4].x != f.nodes[ups[b]/4].x {
				return f.nodes[ups[a]/4].x < f.nodes[ups[b]/4].x
			}
			return ups[a] < ups[b]
		})
		for k, u := range ups {
			gaps[li].tracks[u] = k
		}
	}
	top := make([]int, len(byLayer))
	y := 0
	for li := range byLayer {
		top[li] = y
		y += bandH[li]
		if li < len(byLayer)-1 {
			y += 1 + len(gaps[li].tracks) + gaps[li].label + 1
		}
	}
	var labels []placedLabel
	for _, s := range segs {
		u, d := f.nodes[s.up], f.nodes[s.down]
		e := f.edges[s.edge]
		li := u.layer
		g := gaps[li]
		gapTop := top[li] + bandH[li]
		arrowRow := top[li+1] - 1
		ux, dx := u.cx(), d.cx()
		uBottom := top[li] + u.h // first row below the box
		if u.dummy {
			uBottom = top[li]
			c.vline(ux, top[li], top[li]+bandH[li], e.w, Edge)
		}
		role := Edge
		if t, ok := g.tracks[trackKey(s)]; ok {
			ty := gapTop + 1 + t
			c.vline(ux, uBottom, ty, e.w, role)
			c.hline(ux, dx, ty, e.w, role)
			c.vline(dx, ty, arrowRow, e.w, role)
		} else {
			c.vline(ux, uBottom, arrowRow, e.w, role)
		}
		if d.dummy {
			c.vline(dx, arrowRow, top[li+1], e.w, role)
		}
		// Arrowheads: at the lower end for edges going down, at the upper
		// end for back edges.
		down, upHead := e.head, e.both
		if e.reversed {
			down, upHead = e.both, e.head
		}
		if s.last && down {
			c.text(dx, arrowRow, arrowFor("down", sideways), Arrow)
		}
		if s.first && upHead {
			c.text(ux, uBottom, arrowFor("up", sideways), Arrow)
		}
		if s.last && e.label != "" {
			labels = append(labels, placedLabel{text: e.label, x: dx, y: arrowRow - g.label})
			if !sideways {
				// The label sits on the line: clear the cells under it.
				w := textWidth(e.label)
				for x := dx - w/2; x < dx-w/2+w; x++ {
					if p := c.at(x, arrowRow-1); p != nil && p.r == 0 {
						p.mask = 0
					}
				}
				labels[len(labels)-1].y = arrowRow - 1
			}
		}
	}
	var boxes []placedBox
	for li, l := range byLayer {
		for _, i := range l {
			n := f.nodes[i]
			if !n.dummy {
				boxes = append(boxes, placedBox{n: n, x: n.x, y: top[li], w: n.w, h: n.h})
			}
		}
	}
	return c, boxes, labels
}

func arrowFor(dir string, sideways bool) string {
	switch {
	case dir == "down" && !sideways:
		return "▼"
	case dir == "up" && !sideways:
		return "▲"
	case dir == "down":
		return "▶"
	}
	return "◀"
}

// drawNode draws a node's box in its shape, with its text centered.
func drawNode(c *canvas, n *fnode, x, y, w, h int) {
	role, corners := Border, "┌┐└┘"
	switch n.shape {
	case "round", "stadium":
		corners = "╭╮╰╯"
	case "circle":
		corners, role = "╭╮╰╯", Decision
	case "decision":
		corners, role = "◆◆◆◆", Decision
	case "sub":
		corners = "╓╖╙╜"
	}
	c.box(x, y, w, h, corners, role)
	for j := 1; j < h-1; j++ {
		c.text(x+1, y+j, strings.Repeat(" ", w-2), Name)
	}
	for j, l := range n.lines {
		c.text(x+(w-textWidth(l))/2, y+1+j+(h-2-len(n.lines))/2, l, Name)
	}
}

// ---- Turning a canvas ----

func transpose(c *canvas) *canvas {
	out := &canvas{}
	for y, row := range c.cells {
		for x, p := range row {
			if p.r == 0 && p.mask == 0 {
				continue
			}
			q := out.at(y, x)
			*q = p
			q.mask = swapMask(p.mask)
			switch p.r {
			case '▼':
				q.r = '▶'
			case '▲':
				q.r = '◀'
			}
		}
	}
	return out
}

func swapMask(m uint8) uint8 {
	var o uint8
	if m&up != 0 {
		o |= left
	}
	if m&down != 0 {
		o |= right
	}
	if m&left != 0 {
		o |= up
	}
	if m&right != 0 {
		o |= down
	}
	return o
}

func flipV(c *canvas, H int) *canvas {
	out := &canvas{}
	for y, row := range c.cells {
		for x, p := range row {
			if p.r == 0 && p.mask == 0 {
				continue
			}
			q := out.at(x, H-1-y)
			*q = p
			m := p.mask &^ (up | down)
			if p.mask&up != 0 {
				m |= down
			}
			if p.mask&down != 0 {
				m |= up
			}
			q.mask = m
			switch p.r {
			case '▼':
				q.r = '▲'
			case '▲':
				q.r = '▼'
			}
		}
	}
	return out
}

func flipH(c *canvas, W int) *canvas {
	out := &canvas{}
	for y, row := range c.cells {
		for x, p := range row {
			if p.r == 0 && p.mask == 0 {
				continue
			}
			q := out.at(W-1-x, y)
			*q = p
			m := p.mask &^ (left | right)
			if p.mask&left != 0 {
				m |= right
			}
			if p.mask&right != 0 {
				m |= left
			}
			q.mask = m
			switch p.r {
			case '▶':
				q.r = '◀'
			case '◀':
				q.r = '▶'
			}
		}
	}
	return out
}
