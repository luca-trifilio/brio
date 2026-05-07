package panes

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/luca-trifilio/brio/internal/config"
	"github.com/luca-trifilio/brio/internal/theme"
)

// SettingsModel is a modal overlay that displays the brio configuration and
// lists all configured credential-refresh hooks.
type SettingsModel struct {
	Visible bool
}

// NewSettings returns a new, hidden SettingsModel.
func NewSettings() SettingsModel { return SettingsModel{} }

func (s *SettingsModel) Open()  { s.Visible = true }
func (s *SettingsModel) Close() { s.Visible = false }

// View renders the settings modal. cfgPath and cfg are passed by the caller;
// width/height are the inner dimensions available for the box.
func (s *SettingsModel) View(cfg *config.Config, cfgPath string, width, height int) string {
	if width < 20 {
		width = 20
	}

	keyStyle := lipgloss.NewStyle().Foreground(theme.Mauve).Bold(true)
	dimStyle := lipgloss.NewStyle().Foreground(theme.Overlay1)
	labelStyle := lipgloss.NewStyle().Foreground(theme.Lavender)
	valueStyle := lipgloss.NewStyle().Foreground(theme.Text)
	bulletStyle := lipgloss.NewStyle().Foreground(theme.Green).Bold(true)
	errStyle := lipgloss.NewStyle().Foreground(theme.Red)

	innerW := width - 6
	if innerW < 16 {
		innerW = 16
	}

	var b strings.Builder

	// ── Config path ───────────────────────────────────────────────────────────
	b.WriteString("\n")
	b.WriteString("  " + labelStyle.Render("Config") + "  " + valueStyle.Render(cfgPath) + "\n")
	b.WriteString("\n")
	b.WriteString("  " + dimStyle.Render(strings.Repeat("─", innerW)) + "\n")
	b.WriteString("\n")

	// ── Hooks ─────────────────────────────────────────────────────────────────
	b.WriteString("  " + theme.StyleTitle.Render("Hooks") + "\n\n")

	if cfg == nil || len(cfg.Hooks) == 0 {
		b.WriteString("  " + dimStyle.Render("No hooks configured.  Press ") +
			keyStyle.Render("e") +
			dimStyle.Render(" to edit config.") + "\n")
	} else {
		for _, h := range cfg.Hooks {
			// Hook name line.
			b.WriteString("  " + bulletStyle.Render("●") + " " + theme.StyleBold.Render(h.Name) + "\n")

			// Trigger summary.
			trigger := formatStatusList(h.Trigger.Status)
			if h.Trigger.Body != "" {
				trigger += dimStyle.Render("  ·  ") + labelStyle.Render("body=") + valueStyle.Render(h.Trigger.Body)
			}
			if h.Trigger.Tier != "" {
				trigger += dimStyle.Render("  ·  ") + labelStyle.Render("tier=") + valueStyle.Render(h.Trigger.Tier)
			}
			b.WriteString("    " + labelStyle.Render("trigger") + "  " + trigger + "\n")

			// Script path.
			b.WriteString("    " + labelStyle.Render("script ") + "  " + valueStyle.Render(h.Script.Path) + "\n")

			// Output.
			out := valueStyle.Render(h.Output.Type)
			if h.Output.Format != "" {
				out += dimStyle.Render("  ·  ") + valueStyle.Render(h.Output.Format)
			}
			b.WriteString("    " + labelStyle.Render("output ") + "  " + out + "\n")

			// Vars summary.
			if len(h.Vars) > 0 {
				b.WriteString("    " + labelStyle.Render("vars   ") + "  " + formatVars(h.Vars, dimStyle) + "\n")
			}

			// Error (if script.env has an issue, etc. — placeholder for future).
			if h.Script.Path == "" {
				b.WriteString("    " + errStyle.Render("⚠ script.path is required") + "\n")
			}

			b.WriteString("\n")
		}
	}

	// ── Footer ────────────────────────────────────────────────────────────────
	b.WriteString("  " + dimStyle.Render(strings.Repeat("─", innerW)) + "\n")
	footer := keyStyle.Render("e") + dimStyle.Render(" edit  ") +
		keyStyle.Render("r") + dimStyle.Render(" reload  ") +
		keyStyle.Render("q") + dimStyle.Render(" close")
	b.WriteString("  " + footer + "\n")

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.Mauve).
		Background(theme.Mantle).
		Padding(0, 1).
		Width(innerW + 4)

	return box.Render(theme.StyleTitle.Foreground(theme.Mauve).Render("  Settings") + "\n" + b.String())
}

// formatStatusList renders a comma-separated list of status codes.
func formatStatusList(codes []int) string {
	if len(codes) == 0 {
		return lipgloss.NewStyle().Foreground(theme.Overlay1).Render("(none)")
	}
	parts := make([]string, len(codes))
	for i, c := range codes {
		parts[i] = lipgloss.NewStyle().Foreground(theme.Yellow).Render(fmt.Sprintf("%d", c))
	}
	return strings.Join(parts, lipgloss.NewStyle().Foreground(theme.Overlay1).Render(","))
}

// formatVars renders up to 2 explicit mappings, then "+N more".
func formatVars(vars map[string]string, dimStyle lipgloss.Style) string {
	if len(vars) == 0 {
		return ""
	}
	keys := make([]string, 0, len(vars))
	for k := range vars {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	arrow := dimStyle.Render("→")
	valStyle := lipgloss.NewStyle().Foreground(theme.Text)
	keyStyle := lipgloss.NewStyle().Foreground(theme.Lavender)

	const maxShow = 2
	parts := make([]string, 0, maxShow)
	for i, k := range keys {
		if i >= maxShow {
			break
		}
		parts = append(parts, keyStyle.Render(k)+arrow+valStyle.Render(vars[k]))
	}
	out := strings.Join(parts, dimStyle.Render("  "))
	if len(keys) > maxShow {
		out += dimStyle.Render(fmt.Sprintf("  (+%d more)", len(keys)-maxShow))
	}
	return out
}
