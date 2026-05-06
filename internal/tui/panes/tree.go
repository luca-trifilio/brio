// Package panes contains the per-pane rendering logic for the TUI.
package panes

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/luca-trifilio/bruno-tui/internal/model"
)

// TreeNodeKind tells whether a tree row points at a collection, folder, or
// request.
type TreeNodeKind int

const (
	NodeCollection TreeNodeKind = iota
	NodeFolder
	NodeRequest
)

// TreeNode is one visible row in the collection tree.
type TreeNode struct {
	Kind         TreeNodeKind
	Depth        int
	Label        string
	CollectionIx int             // which collection this row belongs to
	Folder       *model.Folder   // populated for folders
	Request      *model.Request  // populated for requests
	Path         string          // unique identity (filesystem path or synthetic)
	Expandable   bool
	Expanded     bool
}

// TreeModel holds visible rows and selection state.
type TreeModel struct {
	Collections []*model.Collection
	// Expanded tracks which folder/collection node paths are open.
	Expanded map[string]bool
	// Cursor is the visible-row index.
	Cursor int

	rows []TreeNode
}

// NewTree builds a tree with the root of every collection expanded by default.
func NewTree(cs []*model.Collection) *TreeModel {
	t := &TreeModel{Collections: cs, Expanded: map[string]bool{}}
	for _, c := range cs {
		t.Expanded[c.Path] = true
	}
	t.Rebuild()
	return t
}

// Rebuild regenerates the visible row list from the expanded set.
func (t *TreeModel) Rebuild() {
	t.rows = t.rows[:0]
	for ix, c := range t.Collections {
		t.rows = append(t.rows, TreeNode{
			Kind:         NodeCollection,
			Depth:        0,
			Label:        c.DisplayName(),
			CollectionIx: ix,
			Path:         c.Path,
			Expandable:   true,
			Expanded:     t.Expanded[c.Path],
		})
		if !t.Expanded[c.Path] {
			continue
		}
		if c.Root != nil {
			t.appendFolder(c.Root, ix, 1, true)
		}
	}
	if t.Cursor >= len(t.rows) {
		t.Cursor = len(t.rows) - 1
	}
	if t.Cursor < 0 {
		t.Cursor = 0
	}
}

func (t *TreeModel) appendFolder(f *model.Folder, ix, depth int, isRoot bool) {
	// Don't render the synthetic root folder line itself; only its children.
	if !isRoot {
		t.rows = append(t.rows, TreeNode{
			Kind:         NodeFolder,
			Depth:        depth,
			Label:        f.Name,
			CollectionIx: ix,
			Folder:       f,
			Path:         f.Path,
			Expandable:   true,
			Expanded:     t.Expanded[f.Path],
		})
		if !t.Expanded[f.Path] {
			return
		}
		depth++
	}
	for _, sub := range f.Folders {
		t.appendFolder(sub, ix, depth, false)
	}
	for _, r := range f.Requests {
		t.rows = append(t.rows, TreeNode{
			Kind:         NodeRequest,
			Depth:        depth,
			Label:        r.Name,
			CollectionIx: ix,
			Request:      r,
			Path:         r.SourcePath,
		})
	}
}

// Rows returns the visible rows.
func (t *TreeModel) Rows() []TreeNode { return t.rows }

// Selected returns the row under the cursor (zero-value if empty).
func (t *TreeModel) Selected() (TreeNode, bool) {
	if t.Cursor < 0 || t.Cursor >= len(t.rows) {
		return TreeNode{}, false
	}
	return t.rows[t.Cursor], true
}

// Down moves the cursor down by one.
func (t *TreeModel) Down() {
	if t.Cursor < len(t.rows)-1 {
		t.Cursor++
	}
}

// Up moves the cursor up by one.
func (t *TreeModel) Up() {
	if t.Cursor > 0 {
		t.Cursor--
	}
}

// Expand opens the row under the cursor (if expandable).
func (t *TreeModel) Expand() {
	n, ok := t.Selected()
	if !ok || !n.Expandable {
		return
	}
	t.Expanded[n.Path] = true
	t.Rebuild()
}

// Collapse closes the row under the cursor (or jumps to its parent line).
func (t *TreeModel) Collapse() {
	n, ok := t.Selected()
	if !ok {
		return
	}
	if n.Expandable && t.Expanded[n.Path] {
		t.Expanded[n.Path] = false
		t.Rebuild()
		return
	}
	// Jump to nearest ancestor row.
	for i := t.Cursor - 1; i >= 0; i-- {
		if t.rows[i].Depth < n.Depth {
			t.Cursor = i
			return
		}
	}
}

// View renders the tree.
func (t *TreeModel) View(width, height int, focused bool) string {
	if width <= 0 {
		width = 30
	}

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
	cursorStyle := lipgloss.NewStyle().Background(lipgloss.Color("236")).Foreground(lipgloss.Color("15"))
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	folderStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("11"))
	reqStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("15"))

	var lines []string
	title := "Collections"
	if focused {
		title = "▌ " + title
	} else {
		title = "  " + title
	}
	lines = append(lines, titleStyle.Render(title))

	for i, r := range t.rows {
		indent := strings.Repeat("  ", r.Depth)
		marker := " "
		if r.Expandable {
			if r.Expanded {
				marker = "▾"
			} else {
				marker = "▸"
			}
		}
		var label string
		switch r.Kind {
		case NodeCollection:
			label = folderStyle.Bold(true).Render(r.Label)
		case NodeFolder:
			label = folderStyle.Render(r.Label)
		case NodeRequest:
			method := ""
			if r.Request != nil {
				method = string(r.Request.Method)
			}
			label = dimStyle.Render(padMethod(method)) + " " + reqStyle.Render(r.Label)
		}
		line := indent + marker + " " + label
		if i == t.Cursor && focused {
			line = cursorStyle.Render(truncate(stripStyle(line), width))
		} else {
			line = truncate(line, width)
		}
		lines = append(lines, line)
	}
	if len(lines) == 1 {
		lines = append(lines, dimStyle.Render("  (empty)"))
	}
	// Clamp to height.
	if height > 0 && len(lines) > height {
		// Scroll so cursor stays visible.
		cursorLine := t.Cursor + 1
		start := 0
		if cursorLine >= height {
			start = cursorLine - height + 1
		}
		end := start + height
		if end > len(lines) {
			end = len(lines)
		}
		lines = lines[start:end]
	}
	return strings.Join(lines, "\n")
}

func padMethod(m string) string {
	if len(m) >= 6 {
		return m[:6]
	}
	return m + strings.Repeat(" ", 6-len(m))
}
