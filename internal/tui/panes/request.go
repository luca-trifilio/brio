package panes

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/luca-trifilio/bruno-tui/internal/interp"
	"github.com/luca-trifilio/bruno-tui/internal/model"
)

// RenderRequest produces a read-only summary of req with vars interpolated.
func RenderRequest(req *model.Request, scope *interp.VarScope, width, height int, focused bool) string {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
	keyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("11"))
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("244"))

	title := "Request"
	if focused {
		title = "▌ " + title
	} else {
		title = "  " + title
	}
	var b strings.Builder
	b.WriteString(titleStyle.Render(title))
	b.WriteString("\n")

	if req == nil {
		b.WriteString(dimStyle.Render("  (no request selected)"))
		return clampHeight(b.String(), height)
	}

	interp := func(s string) string {
		if scope == nil {
			return s
		}
		return scope.Interpolate(s)
	}

	method := string(req.Method)
	url := interp(req.URL)
	b.WriteString(keyStyle.Render(method) + " " + url + "\n")
	b.WriteString(dimStyle.Render(req.SourcePath) + "\n")
	b.WriteString("\n")

	if len(req.Headers) > 0 {
		b.WriteString(keyStyle.Render("Headers") + "\n")
		for _, h := range req.Headers {
			if h.Disabled {
				continue
			}
			b.WriteString("  " + h.Name + ": " + interp(h.Value) + "\n")
		}
		b.WriteString("\n")
	}

	if len(req.QueryParams) > 0 {
		b.WriteString(keyStyle.Render("Query") + "\n")
		for _, p := range req.QueryParams {
			if p.Disabled {
				continue
			}
			b.WriteString("  " + p.Name + " = " + interp(p.Value) + "\n")
		}
		b.WriteString("\n")
	}

	if req.Body.Type != "" && req.Body.Raw != "" {
		b.WriteString(keyStyle.Render("Body ("+req.Body.Type+")") + "\n")
		b.WriteString(indent(interp(req.Body.Raw), "  "))
		b.WriteString("\n")
	}

	if req.HasPreRequestScript {
		b.WriteString(dimStyle.Render("(has pre-request script — not executed in MVP)") + "\n")
	}

	out := b.String()
	out = wrapLines(out, width)
	return clampHeight(out, height)
}

func indent(s, prefix string) string {
	lines := strings.Split(s, "\n")
	for i, l := range lines {
		lines[i] = prefix + l
	}
	return strings.Join(lines, "\n")
}

func wrapLines(s string, width int) string {
	if width <= 0 {
		return s
	}
	lines := strings.Split(s, "\n")
	for i, l := range lines {
		lines[i] = truncate(l, width)
	}
	return strings.Join(lines, "\n")
}

func clampHeight(s string, height int) string {
	if height <= 0 {
		return s
	}
	lines := strings.Split(s, "\n")
	if len(lines) > height {
		lines = lines[:height]
	}
	return strings.Join(lines, "\n")
}
