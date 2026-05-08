package panes

import (
	"path/filepath"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/luca-trifilio/brio/internal/canonical"
	"github.com/luca-trifilio/brio/internal/theme"
)

// PickerResult is returned by Update to signal the outcome.
type PickerResult int

// Picker outcome values.
const (
	PickerContinue  PickerResult = iota // still selecting
	PickerSelected                      // Enter pressed — caller should switch
	PickerCancelled                     // Esc pressed — caller should close
)

// PickerModel is a fuzzy finder over loaded collections.
type PickerModel struct {
	input       textinput.Model
	collections []*canonical.Collection
	matches     []int // indices into collections, ordered by score
	cursor      int
}

// NewPicker constructs an empty picker.
func NewPicker() *PickerModel {
	ti := textinput.New()
	ti.Prompt = "  "
	ti.CharLimit = 128
	ti.Width = 50
	return &PickerModel{input: ti}
}

// Open prepares the picker to be shown over the given list of collections.
func (p *PickerModel) Open(cs []*canonical.Collection) {
	p.collections = cs
	p.input.SetValue("")
	p.input.Focus()
	p.cursor = 0
	p.recompute()
}

// Update processes a key event. The returned PickerResult tells the caller
// what (if anything) to do on confirmation/cancel. The chosen index is the
// index into the slice passed to Open.
func (p *PickerModel) Update(msg tea.KeyMsg) (PickerResult, int, tea.Cmd) {
	switch msg.String() {
	case "esc":
		p.input.Blur()
		return PickerCancelled, -1, nil
	case "enter":
		if len(p.matches) == 0 {
			return PickerContinue, -1, nil
		}
		if p.cursor < 0 || p.cursor >= len(p.matches) {
			return PickerContinue, -1, nil
		}
		p.input.Blur()
		return PickerSelected, p.matches[p.cursor], nil
	case "down", "ctrl+n":
		if p.cursor < len(p.matches)-1 {
			p.cursor++
		}
		return PickerContinue, -1, nil
	case "up", "ctrl+p":
		if p.cursor > 0 {
			p.cursor--
		}
		return PickerContinue, -1, nil
	}
	var cmd tea.Cmd
	p.input, cmd = p.input.Update(msg)
	p.recompute()
	return PickerContinue, -1, cmd
}

// recompute filters and ranks collections by the current query.
func (p *PickerModel) recompute() {
	q := strings.ToLower(strings.TrimSpace(p.input.Value()))
	type scored struct {
		ix    int
		score int
	}
	var s []scored
	for i, c := range p.collections {
		score := matchScore(c, q)
		if score < 0 {
			continue
		}
		s = append(s, scored{ix: i, score: score})
	}
	sort.SliceStable(s, func(i, j int) bool { return s[i].score > s[j].score })
	p.matches = p.matches[:0]
	for _, x := range s {
		p.matches = append(p.matches, x.ix)
	}
	if p.cursor >= len(p.matches) {
		p.cursor = 0
	}
}

// matchScore returns -1 when the collection does not match q, otherwise a
// rank (higher is better). Empty q matches all with neutral score.
func matchScore(c *canonical.Collection, q string) int {
	if q == "" {
		return 1
	}
	name := strings.ToLower(c.DisplayName())
	if name == "" {
		name = strings.ToLower(filepath.Base(c.Root))
	}
	path := strings.ToLower(c.Root)
	switch {
	case name == q:
		return 100
	case strings.HasPrefix(name, q):
		return 80
	case strings.Contains(name, q):
		return 60
	case strings.Contains(filepath.Base(path), q):
		return 40
	case strings.Contains(path, q):
		return 20
	}
	return -1
}

// View renders the picker overlay.
func (p *PickerModel) View(width, height int) string {
	if width < 30 {
		width = 30
	}
	if height < 8 {
		height = 8
	}
	innerW := width - 4
	if innerW < 20 {
		innerW = 20
	}

	keyStyle := lipgloss.NewStyle().Foreground(theme.Mauve).Bold(true)
	dimStyle := lipgloss.NewStyle().Foreground(theme.Overlay1)
	cursorStyle := lipgloss.NewStyle().Background(theme.Surface1).Foreground(theme.Text)

	var b strings.Builder
	b.WriteString("\n")
	b.WriteString("  " + p.input.View() + "\n")
	b.WriteString("  " + dimStyle.Render(strings.Repeat("─", innerW)) + "\n")

	if len(p.matches) == 0 {
		if len(p.collections) == 0 {
			b.WriteString("  " + dimStyle.Render("No collections configured.") + "\n")
		} else {
			b.WriteString("  " + dimStyle.Render("No matches.") + "\n")
		}
	} else {
		// Show up to height-6 rows.
		maxRows := height - 6
		if maxRows < 1 {
			maxRows = 1
		}
		start := 0
		if p.cursor >= maxRows {
			start = p.cursor - maxRows + 1
		}
		end := start + maxRows
		if end > len(p.matches) {
			end = len(p.matches)
		}
		for i := start; i < end; i++ {
			c := p.collections[p.matches[i]]
			label := c.DisplayName()
			if label == "" {
				label = filepath.Base(c.Root)
			}
			row := "  " + label + "  " + dimStyle.Render(c.Root)
			if i == p.cursor {
				row = cursorStyle.Render(row)
			}
			b.WriteString(row + "\n")
		}
	}

	b.WriteString("  " + dimStyle.Render(strings.Repeat("─", innerW)) + "\n")
	b.WriteString("  " + keyStyle.Render("Enter") + dimStyle.Render(" select  ") +
		keyStyle.Render("Esc") + dimStyle.Render(" cancel"))

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.Mauve).
		Background(theme.Mantle).
		Padding(0, 1).
		Width(innerW + 4)

	return box.Render(theme.StyleTitle.Foreground(theme.Mauve).Render("  Pick collection") + "\n" + b.String())
}
