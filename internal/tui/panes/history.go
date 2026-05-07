package panes

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/luca-trifilio/bruno-tui/internal/history"
	"github.com/luca-trifilio/bruno-tui/internal/theme"
)

// HistoryModel is a modal list of history entries.
type HistoryModel struct {
	Visible bool
	Entries []history.Entry
	Cursor  int
}

func (h *HistoryModel) Open(es []history.Entry) {
	h.Entries = es
	h.Visible = true
	h.Cursor = 0
}

func (h *HistoryModel) Close() { h.Visible = false }

func (h *HistoryModel) Selected() (history.Entry, bool) {
	if h.Cursor < 0 || h.Cursor >= len(h.Entries) {
		return history.Entry{}, false
	}
	return h.Entries[h.Cursor], true
}

func (h *HistoryModel) Down() {
	if h.Cursor < len(h.Entries)-1 {
		h.Cursor++
	}
}

func (h *HistoryModel) Up() {
	if h.Cursor > 0 {
		h.Cursor--
	}
}

func (h *HistoryModel) View(width, height int) string {
	if !h.Visible {
		return ""
	}
	cursorStyle := theme.StyleCursorLine
	border := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.Mauve).
		Background(theme.Mantle).
		Padding(0, 1).
		Width(width - 4)

	var b strings.Builder
	b.WriteString(theme.StyleTitle.Foreground(theme.Mauve).Render("History") +
		theme.StyleDim.Render(" (Enter replay, Esc close)") + "\n")
	if len(h.Entries) == 0 {
		b.WriteString(theme.StyleDim.Render("(no history)"))
		return border.Render(b.String())
	}
	maxRows := height - 6
	if maxRows < 5 {
		maxRows = 5
	}
	start := 0
	if h.Cursor >= maxRows {
		start = h.Cursor - maxRows + 1
	}
	end := start + maxRows
	if end > len(h.Entries) {
		end = len(h.Entries)
	}
	for i := start; i < end; i++ {
		e := h.Entries[i]
		statStyle := theme.StyleSuccess
		if e.Error != "" || e.StatusCode >= 500 {
			statStyle = theme.StyleError
		} else if e.StatusCode >= 400 {
			statStyle = theme.StyleWarning
		}
		stat := fmt.Sprintf("%-3d", e.StatusCode)
		if e.StatusCode == 0 {
			stat = "ERR"
		}
		ts := e.Timestamp.Local().Format("01-02 15:04:05")
		line := fmt.Sprintf("%s %s %s %s",
			theme.StyleDim.Render(ts),
			statStyle.Render(stat),
			theme.MethodStyle(e.Method).Render(fmt.Sprintf("%-6s", e.Method)),
			truncate(e.URL, width-30),
		)
		if i == h.Cursor {
			line = cursorStyle.Render(stripStyle(line))
		}
		b.WriteString(line + "\n")
	}
	return border.Render(b.String())
}
