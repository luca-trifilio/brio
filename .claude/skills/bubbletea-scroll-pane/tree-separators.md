# Tree separator rows

## Pattern: NodeSeparator kind

Inject blank spacer rows into `t.rows` during `Rebuild()` to visually separate
top-level groups (e.g. collections). Navigation must skip them transparently.

### 1. Add the kind constant

```go
const (
    NodeCollection TreeNodeKind = iota
    NodeFolder
    NodeRequest
    NodeSeparator // blank spacer row between top-level collections
)
```

### 2. Inject in Rebuild()

```go
func (t *TreeModel) Rebuild() {
    t.rows = t.rows[:0]
    for ix, c := range t.Collections {
        if ix > 0 {
            t.rows = append(t.rows, TreeNode{Kind: NodeSeparator})
        }
        t.rows = append(t.rows, TreeNode{ /* collection row */ })
        // ... append children
    }
    // clamp cursor, then ensure it doesn't park on a separator
    if t.Cursor >= len(t.rows) { t.Cursor = len(t.rows) - 1 }
    if t.Cursor < 0 { t.Cursor = 0 }
    t.skipSeparator(+1)
}
```

### 3. skipSeparator helper

Advances cursor in `dir` (±1) until it sits on a real row. Falls back to the
opposite direction if it walks off the slice boundary.

```go
func (t *TreeModel) skipSeparator(dir int) {
    for t.Cursor >= 0 && t.Cursor < len(t.rows) &&
        t.rows[t.Cursor].Kind == NodeSeparator {
        t.Cursor += dir
    }
    // walked off the front
    if t.Cursor < 0 {
        t.Cursor = 0
        for t.Cursor < len(t.rows) && t.rows[t.Cursor].Kind == NodeSeparator {
            t.Cursor++
        }
    }
    // walked off the back
    if t.Cursor >= len(t.rows) {
        t.Cursor = len(t.rows) - 1
        for t.Cursor > 0 && t.rows[t.Cursor].Kind == NodeSeparator {
            t.Cursor--
        }
    }
}
```

### 4. Navigation methods skip separators

```go
func (t *TreeModel) Down() {
    next := t.Cursor + 1
    for next < len(t.rows) && t.rows[next].Kind == NodeSeparator { next++ }
    if next < len(t.rows) { t.Cursor = next }
    t.clamp()
}

func (t *TreeModel) Up() {
    prev := t.Cursor - 1
    for prev >= 0 && t.rows[prev].Kind == NodeSeparator { prev-- }
    if prev >= 0 { t.Cursor = prev }
    t.clamp()
}

func (t *TreeModel) HalfPageDown() {
    h := t.viewH() / 2
    if h < 1 { h = 1 }
    t.Cursor += h
    if t.Cursor >= len(t.rows) && len(t.rows) > 0 { t.Cursor = len(t.rows) - 1 }
    t.skipSeparator(+1)
    t.clamp()
}

func (t *TreeModel) Top() { t.Cursor = 0; t.offset = 0; t.skipSeparator(+1) }
func (t *TreeModel) Bottom() {
    if len(t.rows) > 0 { t.Cursor = len(t.rows) - 1 }
    t.skipSeparator(-1)
    t.clamp()
}
```

### 5. Render as blank line in View()

```go
if r.Kind == NodeSeparator {
    lines = append(lines, strings.Repeat(" ", contentW)+bar)
    continue
}
```

The blank line participates in scrollbar accounting normally (it occupies one
viewport slot), keeping the scrollbar thumb proportionally correct.

## Key invariant

The cursor **never** parks on a `NodeSeparator` row. Every method that moves
the cursor (`Rebuild`, `Down`, `Up`, `HalfPageDown`, `HalfPageUp`, `Top`,
`Bottom`) must call `skipSeparator` before returning.
