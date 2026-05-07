---
name: bubbles-viewport
description: Scrollable content viewport from charmbracelet/bubbles v1. Use when adding scrollable text/content panes to a Bubble Tea TUI, wiring keyboard scrolling, or composing viewport with lipgloss layouts.
---

# bubbles-viewport

Scrollable content viewport from charmbracelet/bubbles v1.

## Setup

```go
import "github.com/charmbracelet/bubbles/viewport"

vp := viewport.New(width, height)
vp.SetContent(renderedString)   // set all content at once
```

## Scrolling API

```go
vp.YOffset                       // current scroll position (int field, read/write)
vp.SetYOffset(n)                 // clamps to valid range

vp.ScrollDown(n)                 // move down n lines
vp.ScrollUp(n)                   // move up n lines
vp.HalfPageDown()
vp.HalfPageUp()
vp.PageDown()
vp.PageUp()

vp.AtTop() bool                  // true if scrolled to top
vp.AtBottom() bool               // true if at or past bottom
vp.ScrollPercent() float64       // 0.0 – 1.0
```

## Update and View

```go
// Delegate messages for keyboard/mouse handling
case tea.KeyMsg, tea.MouseMsg:
    var cmd tea.Cmd
    vp, cmd = vp.Update(msg)
    return m, cmd

// Render
vp.View()   // returns the visible portion as a string
```

## Resize

```go
// On WindowSizeMsg:
vp.Width  = newW
vp.Height = newH
```

## When to use viewport vs manual scroll

Use `viewport` when:
- Content is static text set via `SetContent`
- You want built-in key bindings (j/k, d/u, f/b, g/G)

Use a manual `offset int` in a custom model (like `ResponseModel` in bruno-tui) when:
- You need custom key prefix handling (e.g. `5j` = scroll 5 lines)
- You need to rebuild lines on resize
- You want tight control over what `g`/`G`/`gg` do

## Scroll indicator pattern

```go
pct := int(vp.ScrollPercent() * 100)
indicator := fmt.Sprintf("── %d%% (%d/%d lines)", pct, vp.YOffset+1, totalLines)
```
