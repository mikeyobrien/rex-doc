package mermaid

import (
	"errors"
	"math/rand"
	"strings"
	"testing"
)

func text(lines []Line) string {
	var b strings.Builder
	for _, l := range lines {
		for _, r := range l {
			b.WriteString(r.Text)
		}
		b.WriteByte('\n')
	}
	return b.String()
}

func TestSequenceDiagram(t *testing.T) {
	src := "sequenceDiagram\n  autonumber\n  participant C as Client\n  C->>S: SYN\n  S-->>C: SYN/ACK\n  loop retry\n    C->>S: GET\n  end\n  Note over C,S: done\n"
	lines, err := Render(src, 100)
	if err != nil {
		t.Fatal(err)
	}
	out := text(lines)
	for _, want := range []string{"Client", "1. SYN", "2. SYN/ACK", "▶", "◀", "╌", "loop", "retry", "done", "┊"} {
		if !strings.Contains(out, want) {
			t.Errorf("sequence diagram is missing %q:\n%s", want, out)
		}
	}
}

func TestFlowchartDirections(t *testing.T) {
	cases := map[string]string{"TD": "▼", "BT": "▲", "LR": "▶", "RL": "◀"}
	for dir, arrow := range cases {
		lines, err := Render("flowchart "+dir+"\n  A[Start] -->|go| B{Check}\n  B --> C((End))\n", 100)
		if err != nil {
			t.Fatalf("%s: %v", dir, err)
		}
		out := text(lines)
		for _, want := range []string{"Start", "Check", "End", "go", arrow} {
			if !strings.Contains(out, want) {
				t.Errorf("%s flowchart is missing %q:\n%s", dir, want, out)
			}
		}
	}
}

func TestFlowchartEdgeKinds(t *testing.T) {
	lines, err := Render("graph TD\n  A ==> B\n  C -.-> D\n  E --- F\n", 100)
	if err != nil {
		t.Fatal(err)
	}
	out := text(lines)
	if !strings.ContainsAny(out, "┃━") || !strings.ContainsAny(out, "┆┄") {
		t.Errorf("heavy and dotted edges should look different:\n%s", out)
	}
}

func TestCyclesDontLoopForever(t *testing.T) {
	if _, err := Render("graph LR\n  A --> B --> C --> A\n  C --> C\n", 100); err != nil {
		t.Fatal(err)
	}
}

func TestPie(t *testing.T) {
	lines, err := Render("pie title Pets\n  \"Dogs\" : 3\n  \"Cats\" : 1\n", 80)
	if err != nil {
		t.Fatal(err)
	}
	out := text(lines)
	if !strings.Contains(out, "75.0%") || !strings.Contains(out, "25.0%") || !strings.Contains(out, "█") {
		t.Errorf("pie chart:\n%s", out)
	}
}

func TestOtherKindsAndWidth(t *testing.T) {
	if _, err := Render("classDiagram\n  A <|-- B\n", 100); !errors.Is(err, ErrUnsupported) {
		t.Errorf("classDiagram: %v, want ErrUnsupported", err)
	}
	if _, err := Render("sequenceDiagram\n  A->>B: "+strings.Repeat("long ", 40)+"\n", 40); !errors.Is(err, ErrTooWide) {
		t.Errorf("a long message in 40 columns: %v, want ErrTooWide", err)
	}
	if _, err := Render("", 80); !errors.Is(err, ErrEmpty) {
		t.Errorf("empty: %v, want ErrEmpty", err)
	}
}

// Diagrams come from documents written by others: nothing may panic.
func TestGarbageDoesntPanic(t *testing.T) {
	parts := []string{"A", "B[x]", "-->", "-.->", "==>", "|l|", "{", "}", "((", "))", "&", ";", ":", "end",
		"loop", "alt", "else", "Note over A:", "->>", "-->>", "participant", "as", "\"", "%%", "\n", " "}
	r := rand.New(rand.NewSource(1))
	for _, head := range []string{"flowchart TD\n", "graph LR\n", "sequenceDiagram\n", "pie\n"} {
		for i := 0; i < 300; i++ {
			var b strings.Builder
			b.WriteString(head)
			for j := 0; j < r.Intn(40); j++ {
				b.WriteString(parts[r.Intn(len(parts))])
			}
			Render(b.String(), 80)
		}
	}
}
