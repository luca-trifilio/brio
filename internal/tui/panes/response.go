package panes

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/luca-trifilio/bruno-tui/internal/httpx"
)

// RenderResponse formats an httpx.Response for display.
func RenderResponse(resp *httpx.Response, width, height int, focused bool) string {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	okStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	errStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("9"))

	title := "Response"
	if focused {
		title = "▌ " + title
	} else {
		title = "  " + title
	}
	var b strings.Builder
	b.WriteString(titleStyle.Render(title))
	b.WriteString("\n")

	if resp == nil {
		b.WriteString(dimStyle.Render("  (no response yet — press Enter on a request)"))
		return clampHeight(b.String(), height)
	}

	if resp.Err != nil {
		b.WriteString(errStyle.Render("ERR ") + resp.Err.Error() + "\n")
	} else {
		st := okStyle
		if resp.StatusCode >= 400 {
			st = errStyle
		}
		b.WriteString(st.Render(fmt.Sprintf("%s ", resp.Status)))
		b.WriteString(dimStyle.Render(fmt.Sprintf("(%s)", resp.Elapsed.Round(1e6))))
		b.WriteString("\n")
	}
	b.WriteString(dimStyle.Render(resp.Method+" "+resp.URL) + "\n")
	b.WriteString("\n")

	if len(resp.Body) > 0 {
		body := pretty(resp.Body, resp.Headers.Get("Content-Type"))
		b.WriteString(body)
	}

	out := b.String()
	out = wrapLines(out, width)
	return clampHeight(out, height)
}

func pretty(body []byte, contentType string) string {
	if strings.Contains(strings.ToLower(contentType), "json") || looksJSON(body) {
		var v any
		if err := json.Unmarshal(body, &v); err == nil {
			out, err := json.MarshalIndent(v, "", "  ")
			if err == nil {
				return string(out)
			}
		}
	}
	return string(body)
}

func looksJSON(b []byte) bool {
	s := strings.TrimSpace(string(b))
	return strings.HasPrefix(s, "{") || strings.HasPrefix(s, "[")
}
