---
name: bubbletea-mouse
description: Mouse support in Bubble Tea v1 TUIs — enabling cell-motion, hit-testing (x,y) to pane, geometry caching for layout-aware dispatch, scroll wheel per pane, click-to-focus and click-to-cursor. Use when adding mouse support to a multi-pane Bubble Tea TUI.
---

# Mouse Support in Bubble Tea v1

## Enable mouse

```go
prog := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())
```

`WithMouseCellMotion()` reports clicks and wheel events only (no motion noise).  
Use `WithMouseAllMotion()` only if hover effects are needed.

## Mouse event types

```go
case tea.MouseMsg:
    msg.X, msg.Y  // 0-indexed screen coordinates
    msg.Button    // tea.MouseButtonLeft, MouseButtonWheelUp, MouseButtonWheelDown, …
    msg.Action    // tea.MouseActionPress, MouseActionRelease, MouseActionMotion
```

Always guard clicks: `if msg.Action != tea.MouseActionPress { return m, nil }`.

## Geometry cache

Layout geometry is computed inside `View()` but needed in `Update()` for hit-testing.  
Cache it on the model and populate it in `renderLayout()`:

```go
type paneGeometry struct {
    sidebarW  int // total column width of sidebar box (border included)
    treeH     int // total row height of tree box (border included)
    envH      int // total row height of env box (border included)
    reqHeight int // total row height of request box (border included)
}

type Model struct {
    // …
    geom paneGeometry
}

// In renderLayout(), after computing dimensions and before rendering:
m.geom = paneGeometry{sidebarW: sidebarW, treeH: treeH, envH: envH, reqHeight: reqHeight}
```

The geometry from the last render is always valid for the current Update — layout doesn't
change between frames unless a `WindowSizeMsg` fires.

## Hit-testing

Screen rows: row 0 = status bar, rows 1..bodyHeight = body, last row = cmd line.

```go
func (m *Model) handleMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
    // Modals absorb all mouse input.
    if m.help.Visible || m.history.Visible || m.vars.Visible {
        return m, nil
    }

    x, y := msg.X, msg.Y
    g := m.geom
    bodyY := y - 1 // 0-indexed within body (subtract status bar row)

    var hovered Pane
    switch {
    case x < g.sidebarW && bodyY >= 0 && bodyY < g.treeH:
        hovered = PaneTree
    case x < g.sidebarW && bodyY >= g.treeH:
        hovered = PaneEnv
    case x >= g.sidebarW && bodyY >= 0 && bodyY < g.reqHeight:
        hovered = PaneRequest
    case x >= g.sidebarW && bodyY >= g.reqHeight:
        hovered = PaneResponse
    default:
        return m, nil
    }
    // … dispatch by button below …
}
```

## Scroll wheel

Scroll the **hovered** pane regardless of which pane has keyboard focus.  
Focus follows scroll — feels natural and matches most terminal app conventions.

```go
case tea.MouseButtonWheelUp:
    m.focused = hovered
    switch hovered {
    case PaneTree:     m.tree.Up(); m.syncEnvPane()
    case PaneEnv:      m.env.Up()
    case PaneRequest:  m.request.HandleKey("k")
    case PaneResponse: m.response.HandleKey("k")
    }

case tea.MouseButtonWheelDown:
    m.focused = hovered
    switch hovered {
    case PaneTree:     m.tree.Down(); m.syncEnvPane()
    case PaneEnv:      m.env.Down()
    case PaneRequest:  m.request.HandleKey("j")
    case PaneResponse: m.response.HandleKey("j")
    }
```

Reusing `HandleKey("j"/"k")` means scroll obeys numeric prefix and keeps cursor/offset in sync.

## Click-to-focus and click-to-cursor

Boxed panes (lipgloss `RoundedBorder`) have a fixed 3-row header inside the box:

```
bodyY offset   content
   0           box top border  (╭─…─╮)
   1           pane title
   2           pane separator  (─────)
   3+          content rows    (index 0, 1, 2…)
```

For the **tree** (box starts at bodyY = 0):

```go
case tea.MouseButtonLeft:
    if msg.Action != tea.MouseActionPress { return m, nil }
    m.focused = hovered
    switch hovered {
    case PaneTree:
        contentY := bodyY - 3 // border + title + separator
        if contentY >= 0 {
            m.tree.SetCursor(m.tree.Offset() + contentY)
            m.syncEnvPane()
            // Toggle expand/collapse on collection and folder rows.
            if sel, ok := m.tree.Selected(); ok && sel.Expandable {
                if m.tree.IsExpanded(sel.Path) {
                    m.tree.Collapse()
                } else {
                    m.tree.Expand()
                }
                m.syncEnvPane()
            }
        }
    case PaneEnv:
        // Env box starts at bodyY = treeH; same 3-row header offset.
        envContentY := bodyY - g.treeH - 3
        if envContentY >= 0 {
            m.env.SetCursor(envContentY)
        }
    }
```

## Required pane helpers

Add these to tree and env models so the mouse handler can drive them:

```go
// tree.go
func (t *TreeModel) Offset() int              { return t.offset }
func (t *TreeModel) IsExpanded(p string) bool { return t.Expanded[p] }
func (t *TreeModel) SetCursor(row int) {
    if row < 0               { row = 0 }
    if row >= len(t.rows)    { row = len(t.rows) - 1 }
    t.Cursor = row
    t.skipSeparator(+1)
    t.clamp()
}

// env.go
func (e *EnvModel) SetCursor(idx int) {
    if len(e.names) == 0     { return }
    if idx < 0               { idx = 0 }
    if idx >= len(e.names)   { idx = len(e.names) - 1 }
    e.cursor = idx
}
```

## Wire into Update

```go
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    // …
    case tea.MouseMsg:
        return m.handleMouse(msg)
    case tea.KeyMsg:
        return m.handleKey(msg)
    }
    return m, nil
}
```
