---
name: lipgloss-styles
description: Style API for charmbracelet/lipgloss v1 — colors, typography, borders, padding, margins. Use when styling TUI strings with foreground/background colors, bold/italic, borders, or spacing in a Bubble Tea / lipgloss app.
---

# lipgloss-styles

Style API for charmbracelet/lipgloss v1 — colors, typography, borders, spacing.

## Creating and chaining styles

```go
import "github.com/charmbracelet/lipgloss"

// Styles are immutable and chainable; each method returns a new Style.
s := lipgloss.NewStyle().
    Foreground(lipgloss.Color("#cad3f5")).
    Background(lipgloss.Color("#24273a")).
    Bold(true).
    Padding(0, 1)

rendered := s.Render("hello")
```

## Color types

```go
lipgloss.Color("#f5a97f")          // TrueColor hex
lipgloss.Color("214")              // ANSI 256
lipgloss.Color("1")                // ANSI 16

// Adaptive: different color for light vs dark terminal background
lipgloss.AdaptiveColor{Light: "#333", Dark: "#ccc"}
```

## Typography

```go
.Bold(true)
.Italic(true)
.Faint(true)       // dim / subdued
.Underline(true)
.Strikethrough(true)
.Reverse(true)     // swap fg/bg
.Inline(true)      // single line — ignores width/height/padding
```

## Sizing

```go
.Width(40)         // fixed width; pads or truncates content
.Height(10)        // fixed height; pads with newlines
.MaxWidth(80)
.MaxHeight(20)
```

## Borders

```go
.Border(lipgloss.RoundedBorder())           // ╭─╮
.Border(lipgloss.NormalBorder())            // ┌─┐
.Border(lipgloss.ThickBorder())
.Border(lipgloss.DoubleBorder())
.Border(lipgloss.HiddenBorder())            // space-only border (reserves room)

.BorderForeground(lipgloss.Color("#8aadf4"))  // all sides same color
.BorderForeground(top, right, bottom, left)   // per-side colors

// Selective sides:
.BorderTop(true).BorderBottom(false)
```

## Padding and margin

```go
// CSS shorthand: 1 arg = all, 2 = vert/horiz, 4 = top/right/bottom/left
.Padding(0, 1)        // 0 top/bottom, 1 left/right
.Margin(1, 0)         // 1 top/bottom, 0 left/right
.PaddingLeft(2)
.MarginTop(1)
```

## Measuring rendered strings

```go
// Always use lipgloss.Width / Height — not len() — for ANSI-aware measurement
w := lipgloss.Width(rendered)    // display columns (handles wide chars, ANSI)
h := lipgloss.Height(rendered)   // newline count
w, h = lipgloss.Size(rendered)
```

## Style inheritance

```go
base := lipgloss.NewStyle().Foreground(theme.Text)
title := base.Copy().Bold(true).Foreground(theme.Lavender)  // inherits then overrides
```

## Catppuccin Macchiato palette (bruno-tui theme)

```go
// internal/theme/theme.go defines these — never use raw hex in pane code
theme.StyleTitle      // Lavender bold  — pane headers
theme.StyleCollection // Peach bold     — collection names in tree
theme.StyleText       // Text           — normal body text
theme.StyleDim        // Overlay1       — secondary / hints
theme.StyleFocused    // Blue           — active cursor item label
theme.StyleCursorLine // Sky + Surface1 bg — selected row (use in EVERY pane)
theme.StyleSuccess    // Green
theme.StyleError      // Red
theme.StyleWarning    // Yellow
theme.StyleHint       // Overlay1
```
