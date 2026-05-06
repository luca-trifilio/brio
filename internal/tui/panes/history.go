package panes

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/luca-trifilio/bruno-tui/internal/history"
)

// HistoryModel is a modal list of history entries.
type HistoryModel struct {
	Visible bool
	Entries []history.Entry
	Cursor  int
}

// Open populates and shows the modal.
func (h *HistoryModel) Open(es []history.Entry) {
	h.Entries = es
	h.Visible = true
	h.Cursor = 0
}

// Close hides the modal.
func (h *HistoryModel) Close() { h.Visible = false }

// Selected returns the entry under the cursor.
func (h *HistoryModel) Selected() (history.Entry, bool) {
	if h.Cursor < 0 || h.Cursor >= len(h.Entries) {
		return history.Entry{}, false
	}
	return h.Entries[h.Cursor], true
}

// Down/Up move the cursor.
func (h *HistoryModel) Down() {
	if h.Cursor < len(h.Entries)-1 {
		h.Cursor++
	}
}

// Up moves cursor up.
func (h *HistoryModel) Up() {
	if h.Cursor > 0 {
		h.Cursor--
	}
}

// View renders the modal.
func (h *HistoryModel) View(width, height int) string {
	if !h.Visible {
		return ""
	}
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("14"))
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	cursorStyle := lipgloss.NewStyle().Background(lipgloss.Color("236")).Foreground(lipgloss.Color("15"))
	okStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	errStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
	border := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("14")).
		Padding(0, 1).
		Width(width - 4)

	var b strings.Builder
	b.WriteString(titleStyle.Render("History (Enter replay, Esc close)") + "\n")
	if len(h.Entries) == 0 {
		b.WriteString(dimStyle.Render("(no history)"))
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
		statStyle := okStyle
		if e.Error != "" || e.StatusCode >= 400 {
			statStyle = errStyle
		}
		stat := fmt.Sprintf("%-3d", e.StatusCode)
		if e.StatusCode == 0 {
			stat = "ERR"
		}
		ts := e.Timestamp.Local().Format("01-02 15:04:05")
		line := fmt.Sprintf("%s %s %-6s %s",
			dimStyle.Render(ts),
			statStyle.Render(stat),
			e.Method,
			truncate(e.URL, width-30),
		)
		if i == h.Cursor {
			line = cursorStyle.Render(stripStyle(line))
		}
		b.WriteString(line + "\n")
	}
	return border.Render(b.String())
}
