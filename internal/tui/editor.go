package tui

import (
	"strings"
	"unicode"

	"github.com/mattn/go-runewidth"
)

// editor is Source mode's text buffer: the raw file, always editable, with
// vim-like keys. Lines are kept as rune slices so the cursor moves by
// character.
type editor struct {
	lines [][]rune
	row   int // cursor line
	col   int // cursor column, in runes
	want  int // preferred display column for j and k
	top   int // first buffer line on screen
	// insert is true in insert mode, false in normal mode.
	insert bool
	// pending is the first key of a two-key command (d, y, g, Z).
	pending string
	// command is the text typed after ":" while command is active.
	command    string
	commanding bool

	undo, redo []snapshot
	// reg is the yank register; linewise registers hold whole lines.
	reg      [][]rune
	linewise bool

	dirty bool
	// half is half a screen of lines, set by the viewer, for Shift+↑/↓.
	half int
	// scolded is set when a double space was refused, for the viewer to say so.
	scolded bool
}

type snapshot struct {
	lines    [][]rune
	row, col int
}

func newEditor(src []byte) *editor {
	e := &editor{}
	e.set(string(src))
	return e
}

func (e *editor) set(text string) {
	parts := strings.Split(text, "\n")
	// A file's final newline doesn't make an extra empty line.
	if len(parts) > 1 && parts[len(parts)-1] == "" {
		parts = parts[:len(parts)-1]
	}
	e.lines = make([][]rune, len(parts))
	for i, p := range parts {
		e.lines[i] = []rune(strings.TrimRight(p, "\r"))
	}
	if len(e.lines) == 0 {
		e.lines = [][]rune{{}}
	}
	e.clampCursor()
}

// text returns the buffer as file contents, with a final newline.
func (e *editor) text() string {
	var b strings.Builder
	for _, l := range e.lines {
		b.WriteString(string(l))
		b.WriteByte('\n')
	}
	return b.String()
}

// offsetOf returns the byte offset in text() where line row starts.
func (e *editor) offsetOf(row int) int {
	off := 0
	for i := 0; i < row && i < len(e.lines); i++ {
		off += len(string(e.lines[i])) + 1
	}
	return off
}

// rowAt returns the line containing byte offset off.
func (e *editor) rowAt(off int) int {
	for i, l := range e.lines {
		n := len(string(l)) + 1
		if off < n {
			return i
		}
		off -= n
	}
	return len(e.lines) - 1
}

func (e *editor) cur() []rune { return e.lines[e.row] }

func (e *editor) clampCursor() {
	e.row = min(max(e.row, 0), len(e.lines)-1)
	// The cursor may sit after the last character in both modes, so arrow
	// keys and taps can reach the end of a line.
	e.col = min(max(e.col, 0), len(e.cur()))
}

func (e *editor) save() {
	cp := make([][]rune, len(e.lines))
	for i, l := range e.lines {
		cp[i] = append([]rune(nil), l...)
	}
	e.undo = append(e.undo, snapshot{cp, e.row, e.col})
	if len(e.undo) > 500 {
		e.undo = e.undo[1:]
	}
	e.redo = nil
}

func (e *editor) restore(from, to *[]snapshot) bool {
	if len(*from) == 0 {
		return false
	}
	cp := make([][]rune, len(e.lines))
	copy(cp, e.lines)
	*to = append(*to, snapshot{cp, e.row, e.col})
	s := (*from)[len(*from)-1]
	*from = (*from)[:len(*from)-1]
	e.lines, e.row, e.col = s.lines, s.row, s.col
	e.dirty = true
	e.clampCursor()
	return true
}

// ---- Motions ----

func class(r rune) int {
	switch {
	case unicode.IsSpace(r):
		return 0
	case unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_':
		return 1
	}
	return 2
}

// wordForward moves to the start of the next word, across lines.
func (e *editor) wordForward() {
	l := e.cur()
	if e.col < len(l) {
		c := class(l[e.col])
		for e.col < len(l) && class(l[e.col]) == c && c != 0 {
			e.col++
		}
	}
	for {
		l = e.cur()
		for e.col < len(l) && class(l[e.col]) == 0 {
			e.col++
		}
		if e.col < len(l) || e.row == len(e.lines)-1 {
			break
		}
		e.row++
		e.col = 0
		if len(e.cur()) == 0 {
			break
		}
	}
	e.clampCursor()
}

// wordBack moves to the start of the previous word, across lines.
func (e *editor) wordBack() {
	for {
		if e.col == 0 {
			if e.row == 0 {
				return
			}
			e.row--
			e.col = len(e.cur())
			if e.col == 0 {
				return
			}
		}
		l := e.cur()
		e.col--
		for e.col > 0 && class(l[e.col]) == 0 {
			e.col--
		}
		if class(l[e.col]) == 0 {
			continue
		}
		c := class(l[e.col])
		for e.col > 0 && class(l[e.col-1]) == c {
			e.col--
		}
		return
	}
}

// wordEnd moves to the end of the current or next word.
func (e *editor) wordEnd() {
	l := e.cur()
	if e.col < len(l)-1 {
		e.col++
	} else if e.row < len(e.lines)-1 {
		e.row++
		e.col = 0
	}
	for {
		l = e.cur()
		for e.col < len(l) && class(l[e.col]) == 0 {
			e.col++
		}
		if e.col < len(l) || e.row == len(e.lines)-1 {
			break
		}
		e.row++
		e.col = 0
	}
	l = e.cur()
	if e.col < len(l) {
		c := class(l[e.col])
		for e.col < len(l)-1 && class(l[e.col+1]) == c {
			e.col++
		}
	}
	e.clampCursor()
}

func (e *editor) firstNonBlank() {
	e.col = 0
	for e.col < len(e.cur()) && unicode.IsSpace(e.cur()[e.col]) {
		e.col++
	}
	e.clampCursor()
}

func dispCol(l []rune, col int) int { return runewidth.StringWidth(string(l[:min(col, len(l))])) }

// moveRow moves the cursor up or down, keeping its display column.
func (e *editor) moveRow(n int) {
	e.row = min(max(e.row+n, 0), len(e.lines)-1)
	l := e.cur()
	col, w := 0, 0
	for col < len(l) {
		rw := runewidth.RuneWidth(l[col])
		if w+rw > e.want {
			break
		}
		w += rw
		col++
	}
	e.col = col
	e.clampCursor()
}

// ---- Edits ----

func (e *editor) insertRunes(rs []rune) {
	l := e.cur()
	nl := make([]rune, 0, len(l)+len(rs))
	nl = append(nl, l[:e.col]...)
	nl = append(nl, rs...)
	nl = append(nl, l[e.col:]...)
	e.lines[e.row] = nl
	e.col += len(rs)
	e.dirty = true
}

// newline splits the line at the cursor. A list item continues its marker,
// and an empty item ends the list.
func (e *editor) newline() {
	l := e.cur()
	before, after := append([]rune(nil), l[:e.col]...), append([]rune(nil), l[e.col:]...)
	cont := listContinuation(string(before))
	if cont != "" && emptyItem(string(before)) && len(after) == 0 {
		// An empty list item: end the list instead of continuing it.
		e.lines[e.row] = []rune{}
		e.col = 0
		e.dirty = true
		return
	}
	e.lines[e.row] = before
	rest := append([]rune(cont), after...)
	e.lines = append(e.lines[:e.row+1], append([][]rune{rest}, e.lines[e.row+1:]...)...)
	e.row++
	e.col = len([]rune(cont))
	e.dirty = true
}

// listContinuation returns the marker to start the next line with when line
// is a Markdown list item: "- ", "1. " (incremented), "- [ ] ", "> ".
func listContinuation(line string) string {
	indent := line[:len(line)-len(strings.TrimLeft(line, " \t"))]
	rest := line[len(indent):]
	for _, m := range []string{"- [ ] ", "- [x] ", "- [X] ", "* [ ] ", "- ", "* ", "+ ", "> "} {
		if strings.HasPrefix(rest, m) {
			if strings.Contains(m, "[") {
				return indent + m[:2] + "[ ] "
			}
			return indent + m
		}
	}
	n := 0
	for n < len(rest) && rest[n] >= '0' && rest[n] <= '9' {
		n++
	}
	if n > 0 && n < len(rest)-1 && (rest[n] == '.' || rest[n] == ')') && rest[n+1] == ' ' {
		num := 0
		for _, c := range rest[:n] {
			num = num*10 + int(c-'0')
		}
		return indent + itoa(num+1) + string(rest[n]) + " "
	}
	return ""
}

// listMarker returns line's own list marker with its indent ("  - ",
// "3. ", "- [x] ", "> "), or "" when it isn't a list item.
func listMarker(line string) string {
	indent := line[:len(line)-len(strings.TrimLeft(line, " \t"))]
	rest := line[len(indent):]
	for _, m := range []string{"- [ ] ", "- [x] ", "- [X] ", "* [ ] ", "- ", "* ", "+ ", "> "} {
		if strings.HasPrefix(rest, m) || rest+" " == m {
			return indent + m
		}
	}
	n := 0
	for n < len(rest) && rest[n] >= '0' && rest[n] <= '9' {
		n++
	}
	if n > 0 && n < len(rest) && (rest[n] == '.' || rest[n] == ')') && (n+1 == len(rest) || rest[n+1] == ' ') {
		return indent + rest[:n+1] + " "
	}
	return ""
}

// emptyItem reports whether line is only a list marker, with nothing after.
func emptyItem(line string) bool {
	m := listMarker(line)
	return m != "" && strings.TrimSpace(line) == strings.TrimSpace(m)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}

func (e *editor) backspace() {
	// Right after a list marker ("- ", "1. ", "- [ ] ", "> "), Backspace
	// removes the whole marker, so a new line can become anything.
	if e.col > 0 && e.col == len(e.cur()) {
		before := string(e.cur())
		if emptyItem(before) && strings.HasSuffix(before, " ") {
			indent := len([]rune(before)) - len([]rune(strings.TrimLeft(before, " \t")))
			e.lines[e.row] = append([]rune(nil), e.cur()[:indent]...)
			e.col = indent
			e.dirty = true
			return
		}
	}
	if e.col > 0 {
		l := e.cur()
		e.lines[e.row] = append(append([]rune(nil), l[:e.col-1]...), l[e.col:]...)
		e.col--
		e.dirty = true
		return
	}
	if e.row == 0 {
		return
	}
	prev := e.lines[e.row-1]
	e.col = len(prev)
	e.lines[e.row-1] = append(append([]rune(nil), prev...), e.cur()...)
	e.lines = append(e.lines[:e.row], e.lines[e.row+1:]...)
	e.row--
	e.dirty = true
}

func (e *editor) deleteChar() {
	l := e.cur()
	if e.col >= len(l) {
		return
	}
	e.reg, e.linewise = [][]rune{{l[e.col]}}, false
	e.lines[e.row] = append(append([]rune(nil), l[:e.col]...), l[e.col+1:]...)
	e.dirty = true
	e.clampCursor()
}

func (e *editor) deleteLine() {
	e.reg, e.linewise = [][]rune{append([]rune(nil), e.cur()...)}, true
	if len(e.lines) == 1 {
		e.lines[0] = []rune{}
	} else {
		e.lines = append(e.lines[:e.row], e.lines[e.row+1:]...)
	}
	e.dirty = true
	e.clampCursor()
	e.firstNonBlank()
}

func (e *editor) yankLine() {
	e.reg, e.linewise = [][]rune{append([]rune(nil), e.cur()...)}, true
}

// paste puts the register after (or before) the cursor.
func (e *editor) paste(after bool) {
	if len(e.reg) == 0 {
		return
	}
	if e.linewise {
		at := e.row
		if after {
			at++
		}
		var add [][]rune
		for _, l := range e.reg {
			add = append(add, append([]rune(nil), l...))
		}
		e.lines = append(e.lines[:at], append(add, e.lines[at:]...)...)
		e.row = at
		e.dirty = true
		e.firstNonBlank()
		return
	}
	if after && len(e.cur()) > 0 {
		e.col++
	}
	e.insertRunes(e.reg[0])
	e.col--
	e.clampCursor()
}

func (e *editor) openLine(below bool) {
	indent := []rune{}
	for _, r := range e.cur() {
		if r != ' ' && r != '\t' {
			break
		}
		indent = append(indent, r)
	}
	at := e.row
	if below {
		at++
	}
	e.lines = append(e.lines[:at], append([][]rune{indent}, e.lines[at:]...)...)
	e.row = at
	e.col = len(indent)
	e.insert = true
	e.dirty = true
}

// ---- Keys ----

// result tells the viewer what to do after a key in Source mode.
type result int

const (
	stay   result = iota
	toView        // go back to View mode
	write         // save the file
	writeQuit
	quitIt
	forceQuit
	neovim
	notSaved // :q with unsaved changes
	unknownCommand
)

// key handles one key in Source mode.
func (e *editor) key(k string) result {
	if e.commanding {
		return e.commandKey(k)
	}
	if e.insert {
		e.insertKey(k)
		return stay
	}
	return e.normalKey(k)
}

func (e *editor) insertKey(k string) {
	switch k {
	case "esc", "\x03":
		e.insert = false
		e.clampCursor()
	case "\r":
		e.newline()
	case "\x7f", "\b":
		e.backspace()
	case "\t":
		e.insertRunes([]rune("  "))
	case "left":
		e.col = max(e.col-1, 0)
	case "right":
		e.col = min(e.col+1, len(e.cur()))
	case "up":
		e.want = dispCol(e.cur(), e.col)
		e.moveRow(-1)
	case "down":
		e.want = dispCol(e.cur(), e.col)
		e.moveRow(1)
	case "shift-up":
		e.want = dispCol(e.cur(), e.col)
		e.moveRow(-e.half)
	case "shift-down":
		e.want = dispCol(e.cur(), e.col)
		e.moveRow(e.half)
	case "home":
		e.col = 0
	case "end":
		e.col = len(e.cur())
	default:
		if r := []rune(k); len(r) == 1 && (r[0] >= 0x20 && r[0] != 0x7f) {
			if r[0] == ' ' && e.doubleSpace() {
				e.scolded = true
				return
			}
			e.insertRunes(r)
		}
	}
}

// doubleSpace reports whether a space typed now would be a second space
// between words. Indentation and code blocks are left alone.
func (e *editor) doubleSpace() bool {
	l := e.cur()
	if e.col < 2 || l[e.col-1] != ' ' || l[e.col-2] == ' ' || l[e.col-2] == '\t' {
		return false
	}
	if e.col < len(l) && l[e.col] == ' ' {
		return false
	}
	return !e.inCodeBlock()
}

// inCodeBlock reports whether the cursor line is inside a fenced code block.
func (e *editor) inCodeBlock() bool {
	in := false
	for i := 0; i < e.row; i++ {
		t := strings.TrimSpace(string(e.lines[i]))
		if strings.HasPrefix(t, "```") || strings.HasPrefix(t, "~~~") {
			in = !in
		}
	}
	return in
}

func (e *editor) normalKey(k string) result {
	if p := e.pending; p != "" {
		e.pending = ""
		switch p + k {
		case "dd":
			e.save()
			e.deleteLine()
		case "yy":
			e.yankLine()
		case "gg":
			e.row, e.col = 0, 0
			e.firstNonBlank()
		case "ZZ":
			return writeQuit
		case "ZQ":
			return forceQuit
		}
		return stay
	}
	setWant := true
	switch k {
	case "esc":
		return toView
	case "h", "left":
		e.col = max(e.col-1, 0)
	case "l", "right":
		e.col++
		e.clampCursor()
	case "j", "down":
		e.moveRow(1)
		setWant = false
	case "\r":
		// Enter in normal mode moves to the next line, as in vim.
		e.moveRow(1)
		e.firstNonBlank()
	case "k", "up":
		e.moveRow(-1)
		setWant = false
	case "\x04", "shift-down":
		e.moveRow(e.half)
		setWant = false
	case "\x15", "shift-up":
		e.moveRow(-e.half)
		setWant = false
	case "w":
		e.wordForward()
	case "b":
		e.wordBack()
	case "e":
		e.wordEnd()
	case "0", "home":
		e.col = 0
	case "^":
		e.firstNonBlank()
	case "$", "end":
		e.col = len(e.cur())
		e.clampCursor()
	case "G":
		e.row = len(e.lines) - 1
		e.firstNonBlank()
	case "g", "d", "y", "Z":
		e.pending = k
	case "i":
		e.save()
		e.insert = true
	case "a":
		e.save()
		e.insert = true
		e.col = min(e.col+1, len(e.cur()))
	case "I":
		e.save()
		e.insert = true
		e.firstNonBlank()
	case "A":
		e.save()
		e.insert = true
		e.col = len(e.cur())
	case "o":
		e.save()
		e.openLine(true)
	case "O":
		e.save()
		e.openLine(false)
	case "x", "delete":
		e.save()
		e.deleteChar()
	case "p":
		e.save()
		e.paste(true)
	case "P":
		e.save()
		e.paste(false)
	case "u":
		e.restore(&e.undo, &e.redo)
	case "\x12":
		e.restore(&e.redo, &e.undo)
	case "E":
		return neovim
	case ":":
		e.commanding, e.command = true, ""
	}
	if setWant {
		e.want = dispCol(e.cur(), e.col)
	}
	return stay
}

func (e *editor) commandKey(k string) result {
	switch k {
	case "esc", "\x03":
		e.commanding = false
		return stay
	case "\x7f", "\b":
		if r := []rune(e.command); len(r) > 0 {
			e.command = string(r[:len(r)-1])
		} else {
			e.commanding = false
		}
		return stay
	case "\r":
		e.commanding = false
		switch strings.TrimSpace(e.command) {
		case "w":
			return write
		case "wq", "x":
			return writeQuit
		case "q":
			if e.dirty {
				return notSaved
			}
			return quitIt
		case "q!":
			return forceQuit
		case "":
			return stay
		}
		return unknownCommand
	}
	if r := []rune(k); len(r) == 1 && r[0] >= 0x20 {
		e.command += k
	}
	return stay
}
