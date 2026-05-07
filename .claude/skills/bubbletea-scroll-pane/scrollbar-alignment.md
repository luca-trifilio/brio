# Scrollbar Column Alignment & Tree Content Filtering

## Scrollbar character alignment bug

When a pane reserves 1 column for a scrollbar (`contentW := width - 1`) and appends the
`│`/`┃` character after each row, use **`pad`** instead of **`truncate`**.

`truncate(line, contentW)` clips long lines but does **not** right-fill short ones — the
scrollbar character glues directly to the last text character instead of sitting at the
right edge:

```
▸ b2b_commission│   ← wrong: │ immediately after text
▸ bank_to_wallet│
```

`pad(line, contentW)` truncates **and** right-pads with spaces to exactly `contentW`
display columns (ANSI-aware), so the scrollbar always lands in the reserved column:

```go
// WRONG
line = truncate(line, contentW) + bar

// CORRECT
line = pad(line, contentW) + bar
```

The same fix applies to the cursor (highlighted) row — strip styles, pad, re-apply style:

```go
// WRONG
line = cursorStyle.Render(truncate(stripStyle(line), contentW)) + bar

// CORRECT
line = cursorStyle.Render(pad(stripStyle(line), contentW)) + bar
```

`pad` implementation (ANSI-safe):

```go
func pad(s string, width int) string {
    if width <= 0 { return s }
    s = ansi.Truncate(s, width, "…")
    visual := ansi.StringWidth(s)
    if visual < width {
        s += strings.Repeat(" ", width-visual)
    }
    return s
}
```

## Tree content filtering (hide empty folders)

When a method filter is active (e.g. blocking mutating methods in prod), skip folder rows
that contain no usable requests — otherwise the tree shows unexpandable empty folders.

Add a recursive helper and guard it before appending the folder node:

```go
// folderHasUsable returns true when the folder (or any descendant) has at least one
// request whose HTTP method is not in the blocked set.
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
        // Hide folders with no usable requests when a filter is active.
        if len(t.BlockedMethods) > 0 && !t.folderHasUsable(f) {
            return
        }
        t.rows = append(t.rows, TreeNode{ /* … */ })
        // …
    }
    // … recurse into sub-folders and requests …
}
```

The guard is `len(t.BlockedMethods) > 0` — when no filter is set the check is skipped
entirely, so normal (non-prod) rendering has zero overhead.
