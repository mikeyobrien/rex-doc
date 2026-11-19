package mermaid

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// Pie charts become bars: a slice's share of the whole, as a bar and a
// percentage, which reads better in text than a circle would.

var pieSlice = regexp.MustCompile(`^\s*"([^"]*)"\s*:\s*([0-9.]+)\s*$`)

func pie(src string, width int) ([]Line, error) {
	var title string
	type slice struct {
		label string
		value float64
	}
	var slices []slice
	showData := false
	for _, raw := range strings.Split(src, "\n") {
		l := strings.TrimSpace(raw)
		low := strings.ToLower(l)
		switch {
		case strings.HasPrefix(low, "pie"):
			showData = strings.Contains(low, "showdata")
			if i := strings.Index(low, "title"); i >= 0 {
				title = strings.TrimSpace(l[i+5:])
			}
		case strings.HasPrefix(low, "title"):
			title = strings.TrimSpace(l[5:])
		case pieSlice.MatchString(l):
			m := pieSlice.FindStringSubmatch(l)
			v, err := strconv.ParseFloat(m[2], 64)
			if err == nil && v >= 0 {
				slices = append(slices, slice{m[1], v})
			}
		}
	}
	total := 0.0
	labelW := 0
	for _, s := range slices {
		total += s.value
		labelW = max(labelW, textWidth(s.label))
	}
	if len(slices) == 0 || total == 0 {
		return nil, ErrEmpty
	}
	labelW = min(labelW, 24)
	barMax := min(max(width-labelW-16, 10), 50)
	bars := []Role{Bar1, Bar2, Bar3, Bar4, Bar5, Bar6}
	var out []Line
	if title != "" {
		out = append(out, Line{{Text: title, Role: Name}}, Line{})
	}
	for i, s := range slices {
		share := s.value / total
		n := int(share*float64(barMax) + 0.5)
		label := s.label
		if textWidth(label) > labelW {
			label = string([]rune(label)[:labelW-1]) + "…"
		}
		num := fmt.Sprintf("%5.1f%%", share*100)
		if showData {
			num += "  " + strconv.FormatFloat(s.value, 'f', -1, 64)
		}
		out = append(out, Line{
			{Text: label + strings.Repeat(" ", labelW-textWidth(label)) + "  ", Role: Plain},
			{Text: strings.Repeat("█", max(n, 1)), Role: bars[i%len(bars)]},
			{Text: strings.Repeat(" ", barMax-max(n, 1)) + " " + num, Role: Dim},
		})
	}
	return out, nil
}
