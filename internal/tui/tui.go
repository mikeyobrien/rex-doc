// Package tui runs rdxt's full-screen viewer.
package tui

import (
	"bytes"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/mattn/go-runewidth"
	"golang.org/x/term"

	"github.com/travelingflwr/rex-doc/internal/doc"
)

type viewer struct {
	path   string
	src    []byte
	mod    time.Time
	size   int64
	doc    *doc.Doc
	top    int
	width  int
	height int
	out    *os.File
	flash  string
	flashT time.Time
	// pending holds a key that starts a two-key command, like "g" in "gg".
	pending string
	// showFM shows the frontmatter card (toggled with m).
	showFM bool
}

// Run shows path full screen until the user quits.
func Run(path string) error {
	in, out := os.Stdin, os.Stdout
	if !term.IsTerminal(int(in.Fd())) {
		tty, err := os.Open("/dev/tty")
		if err != nil {
			return fmt.Errorf("no terminal to read keys from: %w", err)
		}
		defer tty.Close()
		in = tty
	}
	v := &viewer{path: path, out: out}
	if err := v.load(); err != nil {
		return err
	}
	old, err := term.MakeRaw(int(in.Fd()))
	if err != nil {
		return err
	}
	defer term.Restore(int(in.Fd()), old)

	// Alternate screen, hidden cursor, mouse clicks and wheel (SGR encoding).
	v.write("\x1b[?1049h\x1b[?25l\x1b[?1000h\x1b[?1006h")
	defer v.write("\x1b[?1006l\x1b[?1000l\x1b[?25h\x1b[?1049l")

	v.resize()
	v.draw()

	keys := make(chan []byte, 16)
	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := in.Read(buf)
			if err != nil {
				close(keys)
				return
			}
			keys <- append([]byte(nil), buf[:n]...)
		}
	}()
	winch := make(chan os.Signal, 1)
	signal.Notify(winch, syscall.SIGWINCH)
	defer signal.Stop(winch)
	tick := time.NewTicker(400 * time.Millisecond)
	defer tick.Stop()

	for {
		select {
		case b, ok := <-keys:
			if !ok {
				return nil
			}
			if v.input(b) {
				return nil
			}
		case <-winch:
			v.resize()
		case <-tick.C:
			// Some terminals (Rex on iPad) don't always send SIGWINCH, so
			// check the size here too.
			if w, h, err := term.GetSize(int(v.out.Fd())); err == nil && (w != v.width || h != v.height) {
				v.resize()
			}
			v.checkReload()
			if v.flash != "" && time.Since(v.flashT) > 2*time.Second {
				v.flash = ""
			}
		}
		v.draw()
	}
}

func (v *viewer) write(s string) { v.out.WriteString(s) }

func (v *viewer) load() error {
	st, err := os.Stat(v.path)
	if err != nil {
		return err
	}
	src, err := os.ReadFile(v.path)
	if err != nil {
		return err
	}
	v.src, v.mod, v.size = src, st.ModTime(), st.Size()
	return nil
}

// relayout lays the document out again, keeping the reader's place.
func (v *viewer) relayout() {
	anchor := -1
	if v.doc != nil && v.top > 0 {
		anchor = v.doc.SrcForLine(v.top)
	}
	v.doc = doc.Render(v.src, v.width, doc.Options{ShowFrontmatter: v.showFM})
	if anchor >= 0 {
		v.top = v.doc.LineForSrc(anchor)
	}
	v.clamp()
}

func (v *viewer) resize() {
	w, h, err := term.GetSize(int(v.out.Fd()))
	if err != nil || w <= 0 {
		w, h = 80, 24
	}
	v.width, v.height = w, h
	v.relayout()
}

func (v *viewer) checkReload() {
	st, err := os.Stat(v.path)
	if err != nil || (st.ModTime().Equal(v.mod) && st.Size() == v.size) {
		return
	}
	if v.load() == nil {
		v.relayout()
		v.setFlash("reloaded")
	}
}

func (v *viewer) setFlash(s string) { v.flash, v.flashT = s, time.Now() }

// rows is the number of document rows: the screen less the status line and
// the blank line above it.
func (v *viewer) rows() int { return max(v.height-2, 1) }

func (v *viewer) maxTop() int { return max(len(v.doc.Lines)-v.rows(), 0) }

func (v *viewer) clamp() { v.top = min(max(v.top, 0), v.maxTop()) }

func (v *viewer) scroll(n int) { v.top += n; v.clamp() }

var mouseRE = regexp.MustCompile(`^\x1b\[<(\d+);(\d+);(\d+)([Mm])`)

// input handles a chunk of input and reports whether to quit.
func (v *viewer) input(b []byte) bool {
	for len(b) > 0 {
		if m := mouseRE.FindSubmatch(b); m != nil {
			btn, _ := strconv.Atoi(string(m[1]))
			switch btn &^ (4 | 8 | 16) {
			case 64:
				v.scroll(-3)
			case 65:
				v.scroll(3)
			}
			b = b[len(m[0]):]
			continue
		}
		key, rest := nextKey(b)
		b = rest
		if v.key(key) {
			return true
		}
	}
	return false
}

// Escape sequences for keys rdxt uses.
var seqs = map[string]string{
	"\x1b[A": "up", "\x1b[B": "down", "\x1bOA": "up", "\x1bOB": "down",
	"\x1b[5~": "pgup", "\x1b[6~": "pgdn",
	"\x1b[H": "home", "\x1b[F": "end", "\x1b[1~": "home", "\x1b[4~": "end", "\x1bOH": "home", "\x1bOF": "end",
}

func nextKey(b []byte) (string, []byte) {
	if b[0] == 0x1b {
		for s, name := range seqs {
			if bytes.HasPrefix(b, []byte(s)) {
				return name, b[len(s):]
			}
		}
		if len(b) == 1 {
			return "esc", nil
		}
		// An unknown sequence: skip to its final byte.
		if len(b) > 1 && (b[1] == '[' || b[1] == 'O') {
			for i := 2; i < len(b); i++ {
				if b[i] >= 0x40 && b[i] <= 0x7e {
					return "", b[i+1:]
				}
			}
			return "", nil
		}
		return "esc", b[1:]
	}
	r := []rune(string(b))
	return string(r[0]), b[len(string(r[0])):]
}

func (v *viewer) key(k string) bool {
	if v.pending == "g" {
		v.pending = ""
		if k == "g" {
			v.top = 0
			return false
		}
	}
	half := max(v.rows()/2, 1)
	page := max(v.rows()-2, 1)
	switch k {
	case "q", "\x03":
		return true
	case "j", "down", "\x05", "\r":
		v.scroll(1)
	case "k", "up", "\x19":
		v.scroll(-1)
	case "\x04":
		v.scroll(half)
	case "\x15":
		v.scroll(-half)
	case " ", "pgdn", "\x06":
		v.scroll(page)
	case "b", "pgup", "\x02":
		v.scroll(-page)
	case "g":
		v.pending = "g"
	case "home":
		v.top = 0
	case "G", "end":
		v.top = v.maxTop()
	case "m":
		if v.doc.HasFrontmatter {
			v.showFM = !v.showFM
			v.relayout()
			if v.showFM {
				v.top = 0
			}
		}
	case "}":
		v.jumpHeading(1)
	case "{":
		v.jumpHeading(-1)
	case "r":
		if v.load() == nil {
			v.relayout()
			v.setFlash("reloaded")
		}
	}
	return false
}

// jumpHeading moves to the next (dir 1) or previous (dir -1) heading.
func (v *viewer) jumpHeading(dir int) {
	hs := v.doc.Headings
	if dir > 0 {
		for _, h := range hs {
			if h.Line > v.top {
				v.top = h.Line
				v.clamp()
				return
			}
		}
		return
	}
	for i := len(hs) - 1; i >= 0; i-- {
		if hs[i].Line < v.top {
			v.top = hs[i].Line
			v.clamp()
			return
		}
	}
	v.top = 0
}

func (v *viewer) draw() {
	var b strings.Builder
	b.WriteString("\x1b[?2026h\x1b[H")
	rows := v.rows()
	for r := 0; r < rows; r++ {
		b.WriteString("\x1b[2K")
		b.WriteString(v.doc.Format(v.top+r, true))
		b.WriteString("\r\n")
	}
	b.WriteString("\x1b[2K\r\n\x1b[2K")
	b.WriteString(v.status())
	b.WriteString("\x1b[?2026l")
	v.write(b.String())
}

func (v *viewer) status() string {
	name := filepath.Base(v.path)
	pos := "All"
	if v.maxTop() > 0 {
		switch v.top {
		case 0:
			pos = "Top"
		case v.maxTop():
			pos = "End"
		default:
			pos = fmt.Sprintf("%d%%", v.top*100/v.maxTop())
		}
	}
	right := "View · " + pos
	if v.doc.HasFrontmatter && !v.showFM {
		right = "m frontmatter · " + right
	}
	if v.flash != "" {
		right = v.flash + " · " + right
	}
	margin := v.doc.Margin
	room := v.width - margin*2 - runewidth.StringWidth(right) - 2
	if room < 4 {
		margin = 0
		room = v.width - runewidth.StringWidth(right) - 2
	}
	name = runewidth.Truncate(name, max(room, 0), "…")
	gap := v.width - margin*2 - runewidth.StringWidth(name) - runewidth.StringWidth(right)
	return "\x1b[90m" + strings.Repeat(" ", margin) + name + strings.Repeat(" ", max(gap, 1)) + right + "\x1b[0m"
}
