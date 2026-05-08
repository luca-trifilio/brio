package panes

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/luca-trifilio/brio/internal/canonical"
	"github.com/luca-trifilio/brio/internal/theme"
)

// DiagnosticsModel is a modal listing canonical.Diagnostic items.
type DiagnosticsModel struct {
	Visible bool
	diags   []canonical.Diagnostic
	cursor  int
}

// NewDiagnostics returns a hidden DiagnosticsModel.
func NewDiagnostics() *DiagnosticsModel {
	return &DiagnosticsModel{}
}

// Open shows the modal with the given diagnostics.
func (d *DiagnosticsModel) Open(diags []canonical.Diagnostic) {
	d.diags = diags
	d.cursor = 0
	d.Visible = true
}

// Close hides the modal.
func (d *DiagnosticsModel) Close() { d.Visible = false }

// Toggle flips visibility, opening with diags when not yet visible.
func (d *DiagnosticsModel) Toggle(diags []canonical.Diagnostic) {
	if d.Visible {
		d.Visible = false
		return
	}
	d.Open(diags)
}

// Up moves the cursor up.
func (d *DiagnosticsModel) Up() {
	if d.cursor > 0 {
		d.cursor--
	}
}

// Down moves the cursor down.
func (d *DiagnosticsModel) Down() {
	if d.cursor < len(d.diags)-1 {
		d.cursor++
	}
}

// Selected returns the diagnostic under the cursor (if any).
func (d *DiagnosticsModel) Selected() (canonical.Diagnostic, bool) {
	if d.cursor < 0 || d.cursor >= len(d.diags) {
		return canonical.Diagnostic{}, false
	}
	return d.diags[d.cursor], true
}

// SeverityIcon maps a severity to a single-character icon.
func SeverityIcon(s canonical.Severity) string {
	switch s {
	case canonical.SeverityError:
		return "✖"
	case canonical.SeverityWarn:
		return "⚠"
	case canonical.SeverityInfo:
		return "ℹ"
	}
	return "·"
}

// View renders the diagnostics modal at the given size.
func (d *DiagnosticsModel) View(width, height int) string {
	if width < 30 {
		width = 30
	}
	if height < 8 {
		height = 8
	}
	innerW := width - 4

	keyStyle := lipgloss.NewStyle().Foreground(theme.Mauve).Bold(true)
	dimStyle := lipgloss.NewStyle().Foreground(theme.Overlay1)
	cursorStyle := lipgloss.NewStyle().Background(theme.Surface1).Foreground(theme.Text)

	var b strings.Builder
	b.WriteString("\n")
	if len(d.diags) == 0 {
		b.WriteString("  " + dimStyle.Render("No diagnostics.") + "\n")
	} else {
		maxRows := height - 6
		if maxRows < 1 {
			maxRows = 1
		}
		start := 0
		if d.cursor >= maxRows {
			start = d.cursor - maxRows + 1
		}
		end := start + maxRows
		if end > len(d.diags) {
			end = len(d.diags)
		}
		for i := start; i < end; i++ {
			diag := d.diags[i]
			icon := SeverityIcon(diag.Severity)
			loc := diag.Path
			if diag.Line > 0 {
				loc = fmt.Sprintf("%s:%d", diag.Path, diag.Line)
			}
			row := "  " + icon + " " + diag.Msg + "  " + dimStyle.Render(loc)
			if i == d.cursor {
				row = cursorStyle.Render(row)
			}
			b.WriteString(row + "\n")
		}
	}
	b.WriteString("  " + dimStyle.Render(strings.Repeat("─", innerW)) + "\n")
	b.WriteString("  " + keyStyle.Render("Enter") + dimStyle.Render(" open  ") +
		keyStyle.Render("Esc") + dimStyle.Render(" close"))

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.Yellow).
		Background(theme.Mantle).
		Padding(0, 1).
		Width(innerW + 4)
	return box.Render(theme.StyleTitle.Foreground(theme.Yellow).Render("  Diagnostics") + "\n" + b.String())
}
