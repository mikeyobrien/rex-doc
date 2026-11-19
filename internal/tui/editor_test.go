package tui

import (
	"regexp"
	"strings"
	"testing"
)

func TestListContinuation(t *testing.T) {
	cases := map[string]string{
		"- item":       "- ",
		"  * nested":   "  * ",
		"- [x] done":   "- [ ] ",
		"9. nine":      "10. ",
		"> quoted":     "> ",
		"plain text":   "",
		"2024. a year": "2025. ",
	}
	for line, want := range cases {
		if got := listContinuation(line); got != want {
			t.Errorf("listContinuation(%q) = %q, want %q", line, got, want)
		}
	}
}

func TestEnterOnEmptyItemEndsTheList(t *testing.T) {
	e := newEditor([]byte("- one\n"))
	e.insert = true
	e.col = len(e.cur())
	e.newline()
	if got := string(e.cur()); got != "- " {
		t.Fatalf("after Enter: %q, want \"- \"", got)
	}
	e.newline()
	if got := string(e.cur()); got != "" || len(e.lines) != 2 {
		t.Fatalf("after second Enter: %q with %d lines, want an empty second line", got, len(e.lines))
	}
}

func TestUndoRedo(t *testing.T) {
	e := newEditor([]byte("a\nb\nc\n"))
	e.row = 1
	e.key("d")
	e.key("d")
	if e.text() != "a\nc\n" {
		t.Fatalf("dd: %q", e.text())
	}
	e.key("u")
	if e.text() != "a\nb\nc\n" {
		t.Fatalf("u: %q", e.text())
	}
	e.key("\x12")
	if e.text() != "a\nc\n" {
		t.Fatalf("Ctrl-R: %q", e.text())
	}
}

func TestWordMotions(t *testing.T) {
	e := newEditor([]byte("one two, three\nfour\n"))
	e.key("w")
	if e.col != 4 {
		t.Fatalf("w: col %d, want 4", e.col)
	}
	e.key("w")
	if e.col != 7 {
		t.Fatalf("w onto punctuation: col %d, want 7", e.col)
	}
	e.key("w")
	e.key("w")
	if e.row != 1 || e.col != 0 {
		t.Fatalf("w across lines: %d:%d, want 1:0", e.row, e.col)
	}
	e.key("b")
	if e.row != 0 || e.col != 9 {
		t.Fatalf("b back across lines: %d:%d, want 0:9", e.row, e.col)
	}
}

func TestCommands(t *testing.T) {
	e := newEditor([]byte("x\n"))
	for _, k := range []string{":", "w", "\r"} {
		if r := e.key(k); k == "\r" && r != write {
			t.Fatalf(":w returned %v", r)
		}
	}
	e.dirty = true
	e.key(":")
	e.key("q")
	if r := e.key("\r"); r != notSaved {
		t.Fatalf(":q with changes returned %v, want notSaved", r)
	}
}

func TestBackspaceRemovesAWholeMarker(t *testing.T) {
	e := newEditor([]byte("- one\n"))
	e.insert = true
	e.col = len(e.cur())
	e.newline() // "- "
	e.backspace()
	if got := string(e.cur()); got != "" {
		t.Fatalf("after Backspace: %q, want an empty line", got)
	}
}

func TestCursorCanReachTheEndOfALine(t *testing.T) {
	e := newEditor([]byte("abc\n"))
	e.key("$")
	if e.col != 3 {
		t.Fatalf("$: col %d, want 3 (after the last character)", e.col)
	}
}

func TestEnterVariantsAreOneKey(t *testing.T) {
	for _, in := range []string{"\r", "\n", "\r\n"} {
		k, rest := nextKey([]byte(in + "x"))
		if k != "\r" || string(rest) != "x" {
			t.Errorf("nextKey(%q) = %q, %q", in, k, rest)
		}
	}
}

func TestAppendKeys(t *testing.T) {
	e := newEditor([]byte("ab\n"))
	e.key("a")
	e.key("X")
	if got := string(e.cur()); got != "aXb" {
		t.Fatalf("a then X: %q, want aXb", got)
	}
	e.key("esc")
	e.key("A")
	e.key("!")
	e.key("\r")
	e.key("c")
	if e.text() != "aXb!\nc\n" {
		t.Fatalf("A ! Enter c: %q", e.text())
	}
}

func TestSourceCursorLinesUpWithText(t *testing.T) {
	v := &viewer{width: 60, height: 20, path: "x.md", rich: true}
	v.ed = newEditor([]byte("## Heading\n"))
	v.ed.col = 3 // on the H
	var b strings.Builder
	_, col := v.drawSource(&b, 10, 0)
	line := strings.Split(b.String(), "\r\n")[0]
	plain := regexp.MustCompile(`\x1b\[[0-9;]*[A-Za-z]`).ReplaceAllString(line, "")
	plain = strings.TrimPrefix(plain, "")
	if col >= len(plain) || plain[col] != 'H' {
		t.Fatalf("cursor column %d is on %q, want 'H' in %q", col, string(plain[min(col, len(plain)-1)]), plain)
	}
}
