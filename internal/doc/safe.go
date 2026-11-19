package doc

import "strings"

// Safe makes text from a document safe to send to a terminal: control
// characters (ESC and the rest of C0, DEL, and C1) are shown as visible
// symbols instead, so a document can't change the clipboard, rename tabs,
// fake links or hide text with escape sequences. Tabs become spaces.
func Safe(s string) string {
	clean := true
	for _, r := range s {
		if isControl(r) {
			clean = false
			break
		}
	}
	if clean {
		return s
	}
	var b strings.Builder
	for _, r := range s {
		switch {
		case r == '\t':
			b.WriteString("    ")
		case r == 0x1b:
			b.WriteRune('␛')
		case r < 0x20:
			b.WriteRune(0x2400 + r) // ␀ ␁ … control pictures
		case r == 0x7f:
			b.WriteRune('␡')
		case r >= 0x80 && r <= 0x9f:
			b.WriteRune('�')
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

func isControl(r rune) bool { return r < 0x20 || r == 0x7f || (r >= 0x80 && r <= 0x9f) }
