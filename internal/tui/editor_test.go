package tui

import "testing"

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
