package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/travelingflwr/rex-doc/internal/doc"
)

func TestDocumentsCantSendEscapeSequences(t *testing.T) {
	// A document trying to set the clipboard (OSC 52) and the tab title.
	src := "# Hi\n\nText \x1b]52;c;ZXZpbA==\x07 and \x1b]0;pwned\x07 more.\n"
	d := doc.Render([]byte(src), 80, doc.Options{})
	for i := range d.Lines {
		out := d.Format(i, true)
		// Our own styling uses ESC [ ... m; nothing else may get through.
		rest := strings.NewReplacer("\x1b[0m", "").Replace(out)
		for j := strings.Index(rest, "\x1b"); j >= 0; j = strings.Index(rest, "\x1b") {
			if !strings.HasPrefix(rest[j:], "\x1b[") {
				t.Fatalf("line %d sends a raw escape sequence: %q", i, out)
			}
			rest = rest[j+2:]
		}
		if strings.Contains(out, "\x07") || strings.Contains(out, "]52;") && !strings.Contains(out, "␛]52;") {
			t.Fatalf("line %d passes a control sequence through: %q", i, out)
		}
	}
}

func TestLinksNeverRunPrograms(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "run-me.command")
	os.WriteFile(script, []byte("#!/bin/sh\necho hi\n"), 0o755)
	plain := filepath.Join(dir, "data.csv")
	os.WriteFile(plain, []byte("a,b\n"), 0o644)
	tool := filepath.Join(dir, "tool")
	os.WriteFile(tool, []byte("x"), 0o755)
	for _, f := range []string{script, tool, filepath.Join(dir, "Thing.app")} {
		if !runnable(f) {
			t.Errorf("runnable(%s) = false, want true", filepath.Base(f))
		}
	}
	if runnable(plain) {
		t.Errorf("runnable(data.csv) = true, want false")
	}
}
