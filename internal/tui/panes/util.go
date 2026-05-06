package panes

import (
	"github.com/charmbracelet/lipgloss"
)

// truncate trims s to fit within width display columns.
func truncate(s string, width int) string {
	if width <= 0 {
		return s
	}
	if lipgloss.Width(s) <= width {
		return s
	}
	// Naive truncate to width-1 then add ellipsis.
	runes := []rune(s)
	for len(runes) > 0 && lipgloss.Width(string(runes))+1 > width {
		runes = runes[:len(runes)-1]
	}
	return string(runes) + "…"
}

// stripStyle removes ANSI escape sequences from s.
func stripStyle(s string) string {
	out := make([]rune, 0, len(s))
	in := false
	for _, r := range s {
		if r == 0x1b {
			in = true
			continue
		}
		if in {
			if r == 'm' {
				in = false
			}
			continue
		}
		out = append(out, r)
	}
	return string(out)
}
