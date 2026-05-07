---
name: lipgloss-layout
description: Multi-pane layout composition with charmbracelet/lipgloss v1 — JoinVertical, JoinHorizontal, Place, width/height sizing. Use when assembling multi-pane TUI layouts, aligning blocks, or composing headers/sidebars/content regions.
---

# lipgloss-layout

Multi-pane layout composition with charmbracelet/lipgloss v1.

## Joining blocks

```go
// Stack vertically — pos controls horizontal alignment of shorter strings
lipgloss.JoinVertical(lipgloss.Left, top, bottom)     // left-align
lipgloss.JoinVertical(lipgloss.Center, top, bottom)
lipgloss.JoinVertical(lipgloss.Right, top, bottom)

// Place side by side — pos controls vertical alignment
lipgloss.JoinHorizontal(lipgloss.Top, left, right)    // align tops
lipgloss.JoinHorizontal(lipgloss.Center, left, right)
lipgloss.JoinHorizontal(lipgloss.Bottom, left, right)
```

## Placement (centering overlays)

```go
// Place string in a box of given size at given position
lipgloss.Place(
    totalWidth, totalHeight,
    lipgloss.Center, lipgloss.Center,  // hPos, vPos
    content,
    lipgloss.WithWhitespaceChars(" "),
)

lipgloss.PlaceHorizontal(width, lipgloss.Center, str)
lipgloss.PlaceVertical(height, lipgloss.Center, str)
```

## Position constants

```go
lipgloss.Top    = 0.0
lipgloss.Bottom = 1.0
lipgloss.Center = 0.5
lipgloss.Left   = 0.0
lipgloss.Right  = 1.0
// Any float64 between 0 and 1 is valid
```

## Measuring before layout

```go
// MUST use lipgloss.Width — not len() — because ANSI codes inflate len()
w := lipgloss.Width(block)
h := lipgloss.Height(block)
```

## Bruno-tui layout structure

```
┌─ sidebar (sidebarW) ─┬─ right (rightW) ──────────────────────┐
│                      │                                        │
│  Collections         │  Request (reqHeight = bodyHeight/2)   │
│  (bodyHeight)        │                                        │
│                      ├──────────────────┬─────────────────── │
│                      │ Response         │ Environment         │
│                      │ (respHeight,     │ (envW = sidebarW)  │
│                      │  respW)          │                     │
└──────────────────────┴──────────────────┴─────────────────── ┘
```

```go
// Widths
sidebarW := max(30, m.width/4)
rightW   := m.width - sidebarW - 1
envW     := sidebarW
respW    := rightW - envW - 1

// Heights
bodyHeight := m.height - 2       // minus status bar + command line
reqHeight  := bodyHeight / 2
respHeight := bodyHeight - reqHeight

// Assemble
sidebar   := boxed(tree, sidebarW, bodyHeight, focused == PaneTree)
reqBox    := boxed(reqView, rightW, reqHeight, focused == PaneRequest)
bottomRow := lipgloss.JoinHorizontal(lipgloss.Top,
    boxed(respView, respW, respHeight, focused == PaneResponse),
    boxed(envView,  envW,  respHeight, focused == PaneEnv),
)
right  := lipgloss.JoinVertical(lipgloss.Left, reqBox, bottomRow)
body   := lipgloss.JoinHorizontal(lipgloss.Top, sidebar, right)
```

## Boxed pane helper

```go
func boxed(content string, w, h int, focused bool) string {
    color := theme.BorderUnfocused
    if focused {
        color = theme.BorderFocused
    }
    return lipgloss.NewStyle().
        Border(lipgloss.RoundedBorder()).
        BorderForeground(color).
        Width(w - 2).
        Height(h - 2).
        Render(content)
}
```

## Pane content sizing

When passing width/height to a pane's View, subtract border overhead:
- `Width(w-2)` + `Height(h-2)` in `boxed` consumes 1 border col/row each side
- Pass `rightW-4` as content width to RenderRequest (2 border + 2 padding)
- Pass `respHeight-2` as content height (title + separator = 2 header lines)
