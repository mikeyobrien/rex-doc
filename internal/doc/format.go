package doc

import "strings"

// Format returns line i as text with escape sequences, including the left
// margin. color false leaves out all styling.
func (d *Doc) Format(i int, color bool) string {
	var b strings.Builder
	b.WriteString(spaces(d.Margin))
	if i < 0 || i >= len(d.Lines) {
		return b.String()
	}
	for _, s := range d.Lines[i].Spans {
		if !color {
			b.WriteString(s.Text)
			continue
		}
		sgr := s.Style.SGR()
		b.WriteString(sgr)
		b.WriteString(s.Text)
		if sgr != "" {
			b.WriteString("\x1b[0m")
		}
	}
	return b.String()
}
