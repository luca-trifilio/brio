package panes

import (
	"sort"
	"strings"

	"github.com/luca-trifilio/brio/internal/canonical"
	"github.com/luca-trifilio/brio/internal/theme"
)

// SortEnvNames orders environment names by safety tier (safe → caution → danger)
// and alphabetically within each tier. Exported so NewModel can use the same order.
func SortEnvNames(names []string) {
	sort.Slice(names, func(i, j int) bool {
		ti := theme.ClassifyEnv(names[i])
		tj := theme.ClassifyEnv(names[j])
		if ti != tj {
			return ti < tj // TierSafe(0) < TierCaution(1) < TierDanger(2)
		}
		return strings.ToLower(names[i]) < strings.ToLower(names[j])
	})
}

// EnvModel holds cursor state for the environment switcher pane.
type EnvModel struct {
	names  []string
	cursor int
}

// NewEnv builds an EnvModel from a collection's environments.
func NewEnv(c *canonical.Collection) *EnvModel {
	m := &EnvModel{}
	if c != nil {
		for _, env := range c.Environments {
			m.names = append(m.names, env.Name)
		}
		SortEnvNames(m.names)
	}
	return m
}

// SetCollection rebuilds the name list when the active collection changes.
func (e *EnvModel) SetCollection(c *canonical.Collection) {
	e.names = nil
	if c != nil {
		for _, env := range c.Environments {
			e.names = append(e.names, env.Name)
		}
		SortEnvNames(e.names)
	}
	if e.cursor >= len(e.names) {
		e.cursor = 0
	}
}

// SyncCursor positions the cursor on the currently active env name.
func (e *EnvModel) SyncCursor(active string) {
	for i, n := range e.names {
		if n == active {
			e.cursor = i
			return
		}
	}
}

func (e *EnvModel) Down() {
	if len(e.names) == 0 {
		return
	}
	e.cursor = (e.cursor + 1) % len(e.names)
}

func (e *EnvModel) Up() {
	if len(e.names) == 0 {
		return
	}
	e.cursor = (e.cursor - 1 + len(e.names)) % len(e.names)
}

// Names returns the sorted list of environment names.
func (e *EnvModel) Names() []string { return e.names }

// SetCursor moves the cursor to the given index (clamped to valid range).
func (e *EnvModel) SetCursor(idx int) {
	if len(e.names) == 0 {
		return
	}
	if idx < 0 {
		idx = 0
	}
	if idx >= len(e.names) {
		idx = len(e.names) - 1
	}
	e.cursor = idx
}

// Selected returns the env name under the cursor, if any.
func (e *EnvModel) Selected() (string, bool) {
	if len(e.names) == 0 || e.cursor >= len(e.names) {
		return "", false
	}
	return e.names[e.cursor], true
}

// View renders the env switcher.
func (e *EnvModel) View(active string, width int, focused bool) string {
	cursorStyle := theme.StyleCursorLine

	title := "Environment"
	if focused {
		title = "▌ " + title
	} else {
		title = "  " + title
	}

	separator := theme.StyleDim.Render(strings.Repeat("─", width))

	var b strings.Builder
	b.WriteString(theme.StyleTitle.Render(title))
	b.WriteString("\n")
	b.WriteString(separator)
	b.WriteString("\n")
	if len(e.names) == 0 {
		b.WriteString(theme.StyleDim.Render("  (no environments)"))
		return wrapLines(b.String(), width)
	}
	for i, n := range e.names {
		isCursor := focused && i == e.cursor
		isActive := n == active

		icon := theme.EnvTierIcon(n)
		nameStyle := theme.EnvTierStyle(n)

		// Active bullet inherits the tier color so it reads as one unit.
		activeMarker := "  "
		if isActive {
			activeMarker = nameStyle.Render("● ")
		}

		line := activeMarker + icon + " " + nameStyle.Render(n)
		if isCursor {
			line = cursorStyle.Render(truncate(stripStyle(line), width))
		}
		b.WriteString(line + "\n")
	}
	if focused {
		b.WriteString(theme.StyleHint.Render("  Enter select · e edit · [ ] cycle"))
	} else {
		b.WriteString(theme.StyleHint.Render("  [ ] cycle · :env <name>"))
	}
	return wrapLines(b.String(), width)
}
