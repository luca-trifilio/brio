package panes

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/luca-trifilio/brio/internal/theme"
)

// HelpEntry is a single keybinding row.
type HelpEntry struct {
	Key  string
	Desc string
}

// HelpSection is a named group of keybindings.
type HelpSection struct {
	Title   string
	Entries []HelpEntry
}

// HelpModel is a context-sensitive which-key style modal.
type HelpModel struct {
	Visible  bool
	sections []HelpSection
}

func (h *HelpModel) Open(sections []HelpSection) {
	h.sections = sections
	h.Visible = true
}

func (h *HelpModel) Close() { h.Visible = false }

func (h *HelpModel) View(width, height int) string {
	if !h.Visible {
		return ""
	}

	keyStyle := lipgloss.NewStyle().Foreground(theme.Mauve).Bold(true)
	descStyle := lipgloss.NewStyle().Foreground(theme.Text)
	sectionStyle := lipgloss.NewStyle().Foreground(theme.Lavender).Bold(true)
	dimStyle := lipgloss.NewStyle().Foreground(theme.Overlay1)

	// Find the longest key across all sections for alignment.
	maxKey := 0
	for _, s := range h.sections {
		for _, e := range s.Entries {
			if len(e.Key) > maxKey {
				maxKey = len(e.Key)
			}
		}
	}

	var b strings.Builder
	b.WriteString(theme.StyleTitle.Foreground(theme.Mauve).Render("  Keybindings"))
	b.WriteString("  " + dimStyle.Render("(? or esc to close)") + "\n")
	b.WriteString(dimStyle.Render(strings.Repeat("─", width-6)) + "\n")

	for si, s := range h.sections {
		if si > 0 {
			b.WriteString("\n")
		}
		b.WriteString(sectionStyle.Render(s.Title) + "\n")
		for _, e := range s.Entries {
			pad := strings.Repeat(" ", maxKey-len(e.Key))
			b.WriteString("  " + keyStyle.Render(e.Key) + pad + "  " + descStyle.Render(e.Desc) + "\n")
		}
	}

	innerW := width - 6
	if innerW < 20 {
		innerW = 20
	}

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.Mauve).
		Background(theme.Mantle).
		Padding(0, 1).
		Width(innerW)

	return box.Render(b.String())
}

// Context-specific keybinding definitions.

func GlobalSections() []HelpSection {
	return []HelpSection{
		{
			Title: "Navigation",
			Entries: []HelpEntry{
				{"Tab / Shift+Tab", "cycle panes"},
				{"j / k", "move down / up"},
				{"l / h", "expand / collapse"},
				{"Enter", "execute request"},
			},
		},
		{
			Title: "Environment",
			Entries: []HelpEntry{
				{"] / [", "cycle env forward / backward (global)"},
				{":env <name>", "switch env by name"},
			},
		},
		{
			Title: "Actions",
			Entries: []HelpEntry{
				{"yc", "copy as curl"},
				{"V", "toggle vars panel"},
				{"H", "open history"},
				{":", "command mode"},
				{"q / Ctrl+C", "quit"},
			},
		},
	}
}

func TreeSections() []HelpSection {
	return []HelpSection{
		{
			Title: "Collections tree",
			Entries: []HelpEntry{
				{"j / k", "move down / up (one row)"},
				{"d / u", "half page down / up"},
				{"gg / G", "jump to top / bottom"},
				{"l", "expand collection / folder"},
				{"h", "collapse"},
				{"Enter", "execute selected request"},
			},
		},
		{
			Title: "Actions",
			Entries: []HelpEntry{
				{"yc", "copy selected request as curl"},
				{"H", "open history"},
				{"V", "toggle vars panel"},
			},
		},
	}
}

func EnvSections() []HelpSection {
	return []HelpSection{
		{
			Title: "Environment",
			Entries: []HelpEntry{
				{"j / k", "move down / up"},
				{"Enter", "select environment"},
				{"e", "open env file in $EDITOR"},
				{":env <name>", "switch env by name"},
			},
		},
	}
}

func ResponseSections() []HelpSection {
	return []HelpSection{
		{
			Title: "Scroll",
			Entries: []HelpEntry{
				{"j / k", "line down / up"},
				{"d / u", "half page down / up"},
				{"f / b", "full page down / up"},
				{"gg / G", "top / bottom"},
				{"<n>G", "go to line n"},
			},
		},
		{
			Title: "Search",
			Entries: []HelpEntry{
				{"/", "search forward"},
				{"?", "search backward"},
				{"n / N", "next / previous match"},
			},
		},
		{
			Title: "Leap (flash.nvim style)",
			Entries: []HelpEntry{
				{"s", "leap: type chars to filter, then a label to jump"},
				{"Backspace", "remove last leap character"},
				{"Enter", "jump to first leap target"},
				{"Esc", "cancel leap"},
			},
		},
		{
			Title: "Visual (linewise)",
			Entries: []HelpEntry{
				{"v / V", "enter visual linewise selection"},
				{"j / k", "extend selection down / up"},
				{"d / u / f / b", "extend by half / full page"},
				{"g / G", "extend to top / bottom"},
				{"y", "yank selection to clipboard"},
				{"Esc / v", "cancel visual"},
			},
		},
	}
}

func RequestSections() []HelpSection {
	return []HelpSection{
		{
			Title: "Scroll",
			Entries: []HelpEntry{
				{"j / k", "line down / up"},
				{"d / u", "half page down / up"},
				{"f / b", "full page down / up"},
				{"gg / G", "top / bottom"},
				{"<n>G", "go to line n"},
			},
		},
		{
			Title: "Actions",
			Entries: []HelpEntry{
				{"Enter", "execute request"},
				{"yc", "copy as curl"},
			},
		},
	}
}
