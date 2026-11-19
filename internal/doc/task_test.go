package doc

import (
	"strings"
	"testing"
)

func TestTaskLinesKnowTheirMarker(t *testing.T) {
	src := []byte("- [ ] open\n- [x] done\n")
	d := Render(src, 80, Options{})
	found := 0
	for _, l := range d.Lines {
		if l.TaskCols > 0 {
			found++
			line := string(src[strings.LastIndexByte(string(src[:l.Task]), '\n')+1:])
			line, _, _ = strings.Cut(line, "\n")
			if !strings.Contains(line, "[ ]") && !strings.Contains(line, "[x]") {
				t.Errorf("task line %q has no checkbox", line)
			}
		}
	}
	if found != 2 {
		t.Fatalf("found %d task lines, want 2", found)
	}
}
