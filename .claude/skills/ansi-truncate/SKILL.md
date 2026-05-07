---
name: ansi-truncate
description: ANSI-safe string truncation and width measurement using charmbracelet/x/ansi. Use when truncating, padding, or measuring lipgloss-styled or otherwise ANSI-escaped strings in Go TUIs, where naive len() or rune slicing would corrupt escape sequences.
---

# ansi-truncate

ANSI-safe string truncation and width measurement with charmbracelet/x/ansi.

## Why not len() or naive rune slicing

Naive truncation corrupts ANSI escape sequences — the terminal renders garbage or invisible text.
Always use these functions when dealing with lipgloss-styled strings.

## Truncate

```go
import "github.com/charmbracelet/x/ansi"

// Truncate to `length` display columns, append `tail` if truncated.
// ANSI-safe: escape sequences are excluded from width measurement and preserved.
result := ansi.Truncate(s, width, "…")
result := ansi.Truncate(s, width, "")    // no tail
```

## TruncateLeft

```go
// Remove from the LEFT side, prefix with `prefix` if truncated.
result := ansi.TruncateLeft(s, n, "…")
```

## Strip

```go
// Remove all ANSI escape sequences, return plain text.
plain := ansi.Strip(styled)
```

## StringWidth

```go
// Display width in terminal cells — handles ANSI, wide chars (CJK, emoji).
w := ansi.StringWidth(s)
```

## Grapheme vs wide-char variants

| Grapheme (default)     | Wide-char variant         |
|------------------------|---------------------------|
| `Truncate`             | `TruncateWc`              |
| `TruncateLeft`         | `TruncateLeftWc`          |
| `StringWidth`          | `StringWidthWc`           |

Grapheme variants treat emoji combinations (e.g. 👨‍👩‍👧) as a single unit. Prefer the default variants.

## Usage in bruno-tui

`internal/tui/panes/util.go` wraps this:

```go
func truncate(s string, width int) string {
    if width <= 0 {
        return s
    }
    return ansi.Truncate(s, width, "…")
}
```

Use `truncate(line, width)` on every line before appending to a pane's output.
Use `ansi.Strip(s)` (or the local `stripStyle`) before passing a styled string to `StyleCursorLine.Render()` to avoid double-styling.
