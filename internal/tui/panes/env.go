package panes

import (
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/luca-trifilio/bruno-tui/internal/model"
)

// RenderEnv shows the environment switcher for the focused collection.
func RenderEnv(c *model.Collection, active string, width int, focused bool) string {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	activeStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Bold(true)

	title := "Environment"
	if focused {
		title = "▌ " + title
	} else {
		title = "  " + title
	}

	var b strings.Builder
	b.WriteString(titleStyle.Render(title))
	b.WriteString("\n")
	if c == nil || len(c.Environments) == 0 {
		b.WriteString(dimStyle.Render("  (no environments)"))
		return wrapLines(b.String(), width)
	}
	names := make([]string, 0, len(c.Environments))
	for n := range c.Environments {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		marker := "  "
		style := dimStyle
		if n == active {
			marker = "● "
			style = activeStyle
		}
		b.WriteString(marker + style.Render(n) + "\n")
	}
	b.WriteString(dimStyle.Render("    use :env <name>"))
	return wrapLines(b.String(), width)
}
