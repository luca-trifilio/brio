package panes

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/luca-trifilio/brio/internal/config"
	"github.com/luca-trifilio/brio/internal/theme"
)

// CollEditorResult is returned by UpdateCollEditor to signal what happened.
type CollEditorResult int

const (
	CollEditorContinue  CollEditorResult = iota // still editing
	CollEditorSaved                             // s pressed — caller should persist
	CollEditorCancelled                         // esc pressed — discard changes
)

// SettingsModel is a modal overlay that displays the brio configuration and
// lists all configured credential-refresh hooks.
type SettingsModel struct {
	Visible bool

	// collection editor sub-mode
	editCollections bool
	collPaths       []string // working copy (raw, unexpanded)
	cursor          int
	inputting       bool
	input           textinput.Model
}

// NewSettings returns a new, hidden SettingsModel.
func NewSettings() SettingsModel {
	ti := textinput.New()
	ti.Prompt = "> "
	ti.CharLimit = 512
	ti.Width = 56
	ti.TextStyle = lipgloss.NewStyle().Foreground(theme.Text)
	ti.Cursor.Style = lipgloss.NewStyle().Foreground(theme.Yellow)
	return SettingsModel{input: ti}
}

func (s *SettingsModel) Open()  { s.Visible = true }
func (s *SettingsModel) Close() { s.Visible = false }

// ── Collection editor ────────────────────────────────────────────────────────.

// EnterCollEditor enters the collection-editing sub-mode with a working copy
// of paths. Caller owns the original slice; changes are only committed on Save.
func (s *SettingsModel) EnterCollEditor(paths []string) {
	cp := make([]string, len(paths))
	copy(cp, paths)
	s.collPaths = cp
	s.cursor = 0
	s.inputting = false
	s.editCollections = true
}

// CollEditing reports whether the collection editor sub-mode is active.
func (s *SettingsModel) CollEditing() bool { return s.editCollections }

// CollPaths returns the current working copy of collection paths.
func (s *SettingsModel) CollPaths() []string { return s.collPaths }

// UpdateCollEditor processes a key event inside the collection editor.
// It returns a CollEditorResult so the caller can react (persist, discard)
// and any Bubble Tea command needed (e.g. cursor blink).
func (s *SettingsModel) UpdateCollEditor(msg tea.KeyMsg) (CollEditorResult, tea.Cmd) {
	if s.inputting {
		switch msg.String() {
		case "esc":
			// If this was a blank newly-added row, remove it.
			if s.cursor < len(s.collPaths) && s.collPaths[s.cursor] == "" {
				s.collPaths = append(s.collPaths[:s.cursor], s.collPaths[s.cursor+1:]...)
				if s.cursor > 0 && s.cursor >= len(s.collPaths) {
					s.cursor--
				}
			}
			s.inputting = false
			s.input.Blur()
			return CollEditorContinue, nil
		case "enter":
			val := strings.TrimSpace(s.input.Value())
			if val != "" {
				s.collPaths[s.cursor] = val
			} else {
				// Empty entry → delete it
				s.collPaths = append(s.collPaths[:s.cursor], s.collPaths[s.cursor+1:]...)
				if s.cursor > 0 && s.cursor >= len(s.collPaths) {
					s.cursor--
				}
			}
			s.inputting = false
			s.input.Blur()
			return CollEditorContinue, nil
		}
		var cmd tea.Cmd
		s.input, cmd = s.input.Update(msg)
		return CollEditorContinue, cmd
	}

	switch msg.String() {
	case "j", "down":
		if s.cursor < len(s.collPaths)-1 {
			s.cursor++
		}
	case "k", "up":
		if s.cursor > 0 {
			s.cursor--
		}
	case "a":
		s.collPaths = append(s.collPaths, "")
		s.cursor = len(s.collPaths) - 1
		s.beginInput()
	case "d":
		if len(s.collPaths) > 0 {
			s.collPaths = append(s.collPaths[:s.cursor], s.collPaths[s.cursor+1:]...)
			if s.cursor > 0 && s.cursor >= len(s.collPaths) {
				s.cursor--
			}
		}
	case "enter", "i":
		if len(s.collPaths) > 0 {
			s.beginInput()
		}
	case "s":
		s.editCollections = false
		s.inputting = false
		s.input.Blur()
		return CollEditorSaved, nil
	case "esc", "q":
		s.editCollections = false
		s.inputting = false
		s.input.Blur()
		return CollEditorCancelled, nil
	}
	return CollEditorContinue, nil
}

func (s *SettingsModel) beginInput() {
	if s.cursor < len(s.collPaths) {
		s.input.SetValue(s.collPaths[s.cursor])
		s.input.CursorEnd()
		s.input.Focus()
		s.inputting = true
	}
}

// View renders the settings modal. cfgPath and cfg are passed by the caller;
// width/height are the inner dimensions available for the box.
// When the collection editor is active, a dedicated editing view is rendered.
func (s *SettingsModel) View(cfg *config.Config, cfgPath string, width, height int) string {
	if s.editCollections {
		return s.viewCollEditor(width)
	}
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

	// ── Collections ───────────────────────────────────────────────────────────
	b.WriteString("  " + theme.StyleTitle.Render("Collections") + "\n\n")
	if cfg == nil || len(cfg.Collections) == 0 {
		b.WriteString("  " + dimStyle.Render("None configured — Bruno preferences used.") + "\n")
	} else {
		for _, e := range cfg.Collections {
			label := e.Path
			if e.Format != "" {
				label += dimStyle.Render("  [" + e.Format + "]")
			}
			b.WriteString("  " + bulletStyle.Render("●") + " " + valueStyle.Render(label) + "\n")
		}
	}
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
	footer := keyStyle.Render("c") + dimStyle.Render(" edit collections  ") +
		keyStyle.Render("e") + dimStyle.Render(" edit file  ") +
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

// viewCollEditor renders the collection path editor as the settings modal body.
func (s *SettingsModel) viewCollEditor(width int) string {
	keyStyle := lipgloss.NewStyle().Foreground(theme.Mauve).Bold(true)
	dimStyle := lipgloss.NewStyle().Foreground(theme.Overlay1)
	valueStyle := lipgloss.NewStyle().Foreground(theme.Text)
	cursorStyle := lipgloss.NewStyle().Background(theme.Surface1).Foreground(theme.Text)

	innerW := width - 6
	if innerW < 20 {
		innerW = 20
	}

	var b strings.Builder
	b.WriteString("\n")

	if len(s.collPaths) == 0 {
		b.WriteString("  " + dimStyle.Render("No collections — press 'a' to add one.") + "\n")
	} else {
		for i, p := range s.collPaths {
			var row string
			if s.inputting && i == s.cursor {
				row = "  " + theme.StyleBold.Foreground(theme.Yellow).Render(">") + " " + s.input.View()
			} else if i == s.cursor {
				row = "  " + cursorStyle.Render(" > "+p+" ")
			} else {
				row = "  " + dimStyle.Render("   ") + valueStyle.Render(p)
			}
			b.WriteString(row + "\n")
		}
	}

	b.WriteString("\n")
	b.WriteString("  " + dimStyle.Render(strings.Repeat("─", innerW)) + "\n")

	var footerParts []string
	if !s.inputting {
		footerParts = []string{
			keyStyle.Render("a") + dimStyle.Render(" add"),
			keyStyle.Render("d") + dimStyle.Render(" delete"),
			keyStyle.Render("Enter") + dimStyle.Render(" edit"),
			keyStyle.Render("s") + dimStyle.Render(" save"),
			keyStyle.Render("Esc") + dimStyle.Render(" cancel"),
		}
	} else {
		footerParts = []string{
			keyStyle.Render("Enter") + dimStyle.Render(" confirm"),
			keyStyle.Render("Esc") + dimStyle.Render(" discard"),
		}
	}
	b.WriteString("  " + strings.Join(footerParts, dimStyle.Render("  ")) + "\n")

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.Green).
		Background(theme.Mantle).
		Padding(0, 1).
		Width(innerW + 4)

	title := theme.StyleTitle.Foreground(theme.Green).Render("  Collections") +
		lipgloss.NewStyle().Foreground(theme.Overlay1).Render(" [editing]")
	return box.Render(title + "\n" + b.String())
}
