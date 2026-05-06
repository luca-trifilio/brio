package panes

import (
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// VarsModel implements a key/value editor for runtime variable overrides.
type VarsModel struct {
	Visible bool
	rows    []varRow
	cursor  int
	editing bool
	input   textinput.Model
	col     int // 0 = key, 1 = value (when editing)
}

type varRow struct {
	Key   string
	Value string
}

// NewVars returns a fresh vars panel.
func NewVars() *VarsModel {
	ti := textinput.New()
	ti.Prompt = ""
	ti.CharLimit = 256
	ti.Width = 40
	return &VarsModel{input: ti}
}

// Toggle shows / hides the panel.
func (v *VarsModel) Toggle() { v.Visible = !v.Visible }

// Editing reports whether the panel is in text-editing mode.
func (v *VarsModel) Editing() bool { return v.editing }

// Snapshot returns the current overrides as a map.
func (v *VarsModel) Snapshot() map[string]string {
	out := map[string]string{}
	for _, r := range v.rows {
		if r.Key == "" {
			continue
		}
		out[r.Key] = r.Value
	}
	return out
}

// Set applies a key=value override programmatically (used by `:set`).
func (v *VarsModel) Set(k, val string) {
	for i := range v.rows {
		if v.rows[i].Key == k {
			v.rows[i].Value = val
			return
		}
	}
	v.rows = append(v.rows, varRow{Key: k, Value: val})
	sort.Slice(v.rows, func(i, j int) bool { return v.rows[i].Key < v.rows[j].Key })
}

// Update handles input only when Visible.
func (v *VarsModel) Update(msg tea.Msg) tea.Cmd {
	if !v.Visible {
		return nil
	}
	if v.editing {
		switch m := msg.(type) {
		case tea.KeyMsg:
			switch m.String() {
			case "esc":
				v.editing = false
				v.input.Blur()
				return nil
			case "enter":
				val := v.input.Value()
				v.commitEdit(val)
				v.editing = false
				v.input.Blur()
				return nil
			}
		}
		var cmd tea.Cmd
		v.input, cmd = v.input.Update(msg)
		return cmd
	}
	switch m := msg.(type) {
	case tea.KeyMsg:
		switch m.String() {
		case "j", "down":
			if v.cursor < len(v.rows)*2+1 {
				v.cursor++
			}
		case "k", "up":
			if v.cursor > 0 {
				v.cursor--
			}
		case "a":
			v.rows = append(v.rows, varRow{})
			v.cursor = len(v.rows)*2 - 2
			v.beginEdit()
		case "d":
			if i := v.cursor / 2; i < len(v.rows) {
				v.rows = append(v.rows[:i], v.rows[i+1:]...)
				if v.cursor >= len(v.rows)*2 && v.cursor > 0 {
					v.cursor -= 2
				}
			}
		case "enter", "i":
			v.beginEdit()
		}
	}
	return nil
}

func (v *VarsModel) beginEdit() {
	i := v.cursor / 2
	if i >= len(v.rows) {
		return
	}
	v.col = v.cursor % 2
	cur := v.rows[i].Key
	if v.col == 1 {
		cur = v.rows[i].Value
	}
	v.input.SetValue(cur)
	v.input.CursorEnd()
	v.input.Focus()
	v.editing = true
}

func (v *VarsModel) commitEdit(val string) {
	i := v.cursor / 2
	if i >= len(v.rows) {
		return
	}
	if v.col == 0 {
		v.rows[i].Key = val
	} else {
		v.rows[i].Value = val
	}
}

// View renders the vars panel.
func (v *VarsModel) View(width int) string {
	if !v.Visible {
		return ""
	}
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("13"))
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	cursorStyle := lipgloss.NewStyle().Background(lipgloss.Color("236")).Foreground(lipgloss.Color("15"))
	border := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("13")).
		Padding(0, 1).
		Width(width - 2)

	var b strings.Builder
	b.WriteString(titleStyle.Render("Runtime overrides") + "\n")
	if len(v.rows) == 0 {
		b.WriteString(dimStyle.Render("(empty — press 'a' to add, 'd' to delete, Enter to edit)") + "\n")
	}
	for i, r := range v.rows {
		k := r.Key
		val := r.Value
		if k == "" {
			k = dimStyle.Render("<key>")
		}
		if val == "" {
			val = dimStyle.Render("<value>")
		}
		keyCell := k
		valCell := val
		if v.cursor/2 == i && !v.editing {
			if v.cursor%2 == 0 {
				keyCell = cursorStyle.Render(stripStyle(k))
			} else {
				valCell = cursorStyle.Render(stripStyle(val))
			}
		}
		if v.editing && v.cursor/2 == i {
			if v.col == 0 {
				keyCell = v.input.View()
			} else {
				valCell = v.input.View()
			}
		}
		b.WriteString("  " + keyCell + " = " + valCell + "\n")
	}
	b.WriteString(dimStyle.Render("[a]dd  [d]elete  [Enter] edit  [Esc] close"))
	return border.Render(b.String())
}
