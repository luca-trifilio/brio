package panes

import (
	"strings"

	"github.com/luca-trifilio/brio/internal/model"
	"github.com/luca-trifilio/brio/internal/theme"
)

// TreeNodeKind tells whether a tree row points at a collection, folder, or request.
type TreeNodeKind int

const (
	NodeCollection TreeNodeKind = iota
	NodeFolder
	NodeRequest
	NodeSeparator // blank spacer row injected between top-level collections
)

// TreeNode is one visible row in the collection tree.
type TreeNode struct {
	Kind         TreeNodeKind
	Depth        int
	Label        string
	CollectionIx int
	Folder       *model.Folder
	Request      *model.Request
	Path         string
	Expandable   bool
	Expanded     bool
}

// TreeModel holds visible rows and selection state.
type TreeModel struct {
	Collections    []*model.Collection
	Expanded       map[string]bool
	BlockedMethods map[string]bool // HTTP methods hidden in danger-tier envs
	Cursor         int
	rows           []TreeNode

	offset int // first visible row index
	height int // last known view height (set in View)
}

// NewTree builds a tree with all nodes collapsed by default.
func NewTree(cs []*model.Collection) *TreeModel {
	t := &TreeModel{Collections: cs, Expanded: map[string]bool{}}
	t.Rebuild()
	return t
}

// SetBlockedMethods updates the set of HTTP methods hidden from the tree and
// rebuilds the visible row list. Pass nil to show all methods.
func (t *TreeModel) SetBlockedMethods(methods map[string]bool) {
	t.BlockedMethods = methods
	t.Rebuild()
	t.clamp()
}

// Rebuild regenerates the visible row list from the expanded set.
func (t *TreeModel) Rebuild() {
	t.rows = t.rows[:0]
	for ix, c := range t.Collections {
		// Insert a blank separator before every collection except the first.
		if ix > 0 {
			t.rows = append(t.rows, TreeNode{Kind: NodeSeparator})
		}
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
	// Never leave the cursor parked on a separator.
	t.skipSeparator(+1)
}

// skipSeparator advances the cursor in the given direction (±1) until it
// sits on a non-separator row.  Falls back to the opposite direction if
// it would walk off the end of the slice.
func (t *TreeModel) skipSeparator(dir int) {
	for t.Cursor >= 0 && t.Cursor < len(t.rows) && t.rows[t.Cursor].Kind == NodeSeparator {
		t.Cursor += dir
	}
	if t.Cursor < 0 {
		t.Cursor = 0
		for t.Cursor < len(t.rows) && t.rows[t.Cursor].Kind == NodeSeparator {
			t.Cursor++
		}
	}
	if t.Cursor >= len(t.rows) {
		t.Cursor = len(t.rows) - 1
		for t.Cursor > 0 && t.rows[t.Cursor].Kind == NodeSeparator {
			t.Cursor--
		}
	}
}

// folderHasUsable returns true when the folder (or any of its descendants)
// contains at least one request that is not blocked by the current method filter.
func (t *TreeModel) folderHasUsable(f *model.Folder) bool {
	for _, r := range f.Requests {
		if !t.BlockedMethods[string(r.Method)] {
			return true
		}
	}
	for _, sub := range f.Folders {
		if t.folderHasUsable(sub) {
			return true
		}
	}
	return false
}

func (t *TreeModel) appendFolder(f *model.Folder, ix, depth int, isRoot bool) {
	if !isRoot {
		// When a method filter is active, hide folders that contain no usable
		// requests so the tree doesn't show empty, unexpandable folders.
		if len(t.BlockedMethods) > 0 && !t.folderHasUsable(f) {
			return
		}
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
			if t.BlockedMethods[string(r.Method)] {
				continue
			}
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

func (t *TreeModel) Rows() []TreeNode { return t.rows }

// Offset returns the current scroll offset (index of the first visible row).
func (t *TreeModel) Offset() int { return t.offset }

// IsExpanded reports whether the node at path is currently expanded.
func (t *TreeModel) IsExpanded(path string) bool { return t.Expanded[path] }

// SetCursor moves the cursor to the given absolute row index (clamped, separators skipped).
func (t *TreeModel) SetCursor(row int) {
	if row < 0 {
		row = 0
	}
	if row >= len(t.rows) {
		row = len(t.rows) - 1
	}
	t.Cursor = row
	t.skipSeparator(+1)
	t.clamp()
}

func (t *TreeModel) Selected() (TreeNode, bool) {
	if t.Cursor < 0 || t.Cursor >= len(t.rows) {
		return TreeNode{}, false
	}
	return t.rows[t.Cursor], true
}

// viewH returns the number of scrollable content rows given the last known height.
// Layout: 2 header lines (title + separator) + viewH content rows = height.
func (t *TreeModel) viewH() int {
	h := t.height - 2
	if h < 1 {
		h = 1
	}
	return h
}

// clamp keeps offset in range and the cursor within the visible window.
func (t *TreeModel) clamp() {
	if t.height == 0 {
		return
	}
	viewH := t.viewH()
	maxOffset := len(t.rows) - viewH
	if maxOffset < 0 {
		maxOffset = 0
	}
	if t.offset > maxOffset {
		t.offset = maxOffset
	}
	if t.offset < 0 {
		t.offset = 0
	}
	// Scroll to keep cursor visible.
	if t.Cursor < t.offset {
		t.offset = t.Cursor
	}
	if t.Cursor >= t.offset+viewH {
		t.offset = t.Cursor - viewH + 1
	}
}

func (t *TreeModel) Down() {
	next := t.Cursor + 1
	for next < len(t.rows) && t.rows[next].Kind == NodeSeparator {
		next++
	}
	if next < len(t.rows) {
		t.Cursor = next
	}
	t.clamp()
}

func (t *TreeModel) Up() {
	prev := t.Cursor - 1
	for prev >= 0 && t.rows[prev].Kind == NodeSeparator {
		prev--
	}
	if prev >= 0 {
		t.Cursor = prev
	}
	t.clamp()
}

func (t *TreeModel) HalfPageDown() {
	h := t.viewH() / 2
	if h < 1 {
		h = 1
	}
	t.Cursor += h
	if t.Cursor >= len(t.rows) && len(t.rows) > 0 {
		t.Cursor = len(t.rows) - 1
	}
	t.skipSeparator(+1)
	t.clamp()
}

func (t *TreeModel) HalfPageUp() {
	h := t.viewH() / 2
	if h < 1 {
		h = 1
	}
	t.Cursor -= h
	if t.Cursor < 0 {
		t.Cursor = 0
	}
	t.skipSeparator(-1)
	t.clamp()
}

func (t *TreeModel) Top() {
	t.Cursor = 0
	t.offset = 0
	t.skipSeparator(+1)
}

func (t *TreeModel) Bottom() {
	if len(t.rows) > 0 {
		t.Cursor = len(t.rows) - 1
	}
	t.skipSeparator(-1)
	t.clamp()
}

func (t *TreeModel) Expand() {
	n, ok := t.Selected()
	if !ok || !n.Expandable {
		return
	}
	// When expanding a collection, collapse all other collections first.
	if n.Kind == NodeCollection && !t.Expanded[n.Path] {
		for _, c := range t.Collections {
			if c.Path != n.Path {
				t.Expanded[c.Path] = false
			}
		}
	}
	t.Expanded[n.Path] = true
	t.Rebuild()
	t.clamp()
}

func (t *TreeModel) Collapse() {
	n, ok := t.Selected()
	if !ok {
		return
	}
	if n.Expandable && t.Expanded[n.Path] {
		t.Expanded[n.Path] = false
		t.Rebuild()
		t.clamp()
		return
	}
	for i := t.Cursor - 1; i >= 0; i-- {
		if t.rows[i].Depth < n.Depth {
			t.Cursor = i
			t.clamp()
			return
		}
	}
}

func (t *TreeModel) View(width, height int, focused bool) string {
	if width <= 0 {
		width = 30
	}
	if height > 0 {
		t.height = height
	}
	t.clamp()

	separator := theme.StyleDim.Render(strings.Repeat("─", width))
	title := "Collections"
	if focused {
		title = "▌ " + title
	} else {
		title = "  " + title
	}


	viewH := t.viewH()
	// Reserve 1 column on the right for the scrollbar.
	contentW := width - 1
	if contentW < 1 {
		contentW = 1
	}
	scrollable := len(t.rows) > viewH

	var scrollbar []string
	if scrollable {
		scrollbar = buildScrollbar(t.offset, len(t.rows), viewH)
	}

	start := t.offset
	end := t.offset + viewH
	if end > len(t.rows) {
		end = len(t.rows)
	}

	lines := make([]string, 0, 2+viewH)
	lines = append(lines, theme.StyleTitle.Render(title))
	lines = append(lines, separator)

	for i := 0; i < viewH; i++ {
		bar := " "
		if scrollable && i < len(scrollbar) {
			bar = scrollbar[i]
		}

		rowIdx := start + i
		if rowIdx >= end {
			// Filler row — keeps scrollbar track continuous.
			lines = append(lines, strings.Repeat(" ", contentW)+bar)
			continue
		}

		r := t.rows[rowIdx]
		indentStr := strings.Repeat("  ", r.Depth)
		marker := " "
		if r.Expandable {
			if r.Expanded {
				marker = "▾"
			} else {
				marker = "▸"
			}
		}

		if r.Kind == NodeSeparator {
			lines = append(lines, strings.Repeat(" ", contentW)+bar)
			continue
		}

		var label string
		switch r.Kind {
		case NodeCollection:
			label = theme.StyleCollection.Render(r.Label)
		case NodeFolder:
			label = theme.StyleFocused.Render(r.Label)
		case NodeRequest:
			method := ""
			if r.Request != nil {
				method = string(r.Request.Method)
			}
			label = theme.MethodStyle(method).Render(padMethod(method)) + " " + theme.StyleText.Render(r.Label)
		}

		line := indentStr + marker + " " + label
		if rowIdx == t.Cursor && focused {
			line = theme.StyleCursorLine.Render(pad(stripStyle(line), contentW)) + bar
		} else {
			line = pad(line, contentW) + bar
		}
		lines = append(lines, line)
	}

	// Fallback for completely empty tree.
	if len(t.rows) == 0 {
		lines = append(lines, theme.StyleDim.Render("  (empty)"))
	}

	return strings.Join(lines, "\n")
}

func padMethod(m string) string {
	if len(m) >= 6 {
		return m[:6]
	}
	return m + strings.Repeat(" ", 6-len(m))
}
