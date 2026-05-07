package panes

import (
	"strings"

	"github.com/charmbracelet/x/ansi"
)

// truncate trims s to fit within width display columns, preserving ANSI codes.
func truncate(s string, width int) string {
	if width <= 0 {
		return s
	}
	return ansi.Truncate(s, width, "…")
}

// pad truncates s to width columns and then right-pads with spaces so the
// returned string always occupies exactly width display columns. ANSI escape
// codes are counted correctly (they have zero display width).
func pad(s string, width int) string {
	if width <= 0 {
		return s
	}
	s = ansi.Truncate(s, width, "…")
	visual := ansi.StringWidth(s)
	if visual < width {
		s += strings.Repeat(" ", width-visual)
	}
	return s
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
