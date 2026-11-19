package tui

import (
	"fmt"
	"image"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"syscall"

	"golang.org/x/term"

	"github.com/travelingflwr/rex-doc/internal/doc"
)

// RunImage shows one image filling the window, for a closer look at a
// figure. It's what a tap on a figure opens in a floating Rex window. q,
// Esc or the ✕ pill closes it.
func RunImage(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	cfg, _, err := image.DecodeConfig(f)
	f.Close()
	if err != nil {
		return fmt.Errorf("can't read %s: %w", filepath.Base(path), err)
	}
	in, out := os.Stdin, os.Stdout
	old, err := term.MakeRaw(int(in.Fd()))
	if err != nil {
		return err
	}
	defer term.Restore(int(in.Fd()), old)
	out.WriteString("\x1b[?1049h\x1b[?25l\x1b[?1000h\x1b[?1006h")
	defer out.WriteString("\x1b_Ga=d,d=A,q=2\x1b\\\x1b[?1006l\x1b[?1000l\x1b[?25h\x1b[?1049l")

	const id = 0x00A11E
	sentFor := ""
	draw := func() int {
		w, h, _ := term.GetSize(int(out.Fd()))
		cw, ch := cellSize(int(out.Fd()))
		// Fit the image in the window below the top line and above the caption.
		maxCols, maxRows := max(w-4, 1), min(max(h-4, 1), doc.MaxFigureRows)
		cols := min(maxCols, (cfg.Width+cw-1)/cw)
		rows := max(int(float64(cols*cw)*float64(cfg.Height)/float64(cfg.Width)/float64(ch)+0.5), 1)
		if rows > maxRows {
			rows = maxRows
			cols = max(int(float64(rows*ch)*float64(cfg.Width)/float64(cfg.Height)/float64(cw)+0.5), 1)
		}
		v := &viewer{out: out, images: newImageState()}
		if key := fmt.Sprint(cols, rows); key != sentFor {
			out.WriteString("\x1b_Ga=d,d=A,q=2\x1b\\")
			if data, err := scaledCopy(path, min(cols*cw*2, cfg.Width), min(rows*ch*2, cfg.Height)); err == nil {
				v.sendImage(thumb{id: id, cols: cols, rows: rows, png: data})
			}
			sentFor = key
		}
		var b strings.Builder
		b.WriteString("\x1b[?2026h\x1b[H\x1b[2J")
		closeP, cw2 := pill(" ✕ ", doc.Red, true)
		name := fitName(filepath.Base(path), max(w-cw2-6, 4))
		b.WriteString("  \x1b[1m" + name + "\x1b[0m" + strings.Repeat(" ", max(w-4-visibleWidth(name)-cw2, 1)) + closeP + "\r\n\r\n")
		left := strings.Repeat(" ", max((w-cols)/2, 0))
		color := fmt.Sprintf("\x1b[38;2;%d;%d;%dm", id>>16&0xFF, id>>8&0xFF, id&0xFF)
		for r := 0; r < rows; r++ {
			b.WriteString(left + color + doc.PlaceholderRow(r, cols) + "\x1b[0m\r\n")
		}
		caption := fmt.Sprintf("%d × %d px · q to close", cfg.Width, cfg.Height)
		b.WriteString("\r\n" + strings.Repeat(" ", max((w-len(caption))/2, 0)) + "\x1b[90;3m" + caption + "\x1b[0m")
		b.WriteString("\x1b[?2026l")
		out.WriteString(b.String())
		return w - 2 - cw2
	}
	closeX := draw()

	winch := make(chan os.Signal, 1)
	signal.Notify(winch, syscall.SIGWINCH)
	defer signal.Stop(winch)
	keys := make(chan []byte)
	go func() {
		buf := make([]byte, 256)
		for {
			n, err := in.Read(buf)
			if err != nil {
				close(keys)
				return
			}
			keys <- append([]byte(nil), buf[:n]...)
		}
	}()
	click := regexp.MustCompile(`\x1b\[<0;(\d+);(\d+)m`)
	for {
		select {
		case <-winch:
			closeX = draw()
		case b, ok := <-keys:
			if !ok {
				return nil
			}
			s := string(b)
			if s == "q" || s == "\x1b" || s == "\x03" {
				return nil
			}
			if m := click.FindStringSubmatch(s); m != nil {
				x, _ := strconv.Atoi(m[1])
				y, _ := strconv.Atoi(m[2])
				if y == 1 && x-1 >= closeX {
					return nil
				}
			}
		}
	}
}

func fitName(s string, w int) string {
	if visibleWidth(s) <= w {
		return s
	}
	r := []rune(s)
	for len(r) > 0 && visibleWidth(string(r))+1 > w {
		r = r[:len(r)-1]
	}
	return string(r) + "…"
}
