# ANSI-safe line wrapping

## The trap: wrapping at `width` but displaying at `contentW`

`ansi.Hardwrap(s, width, true)` wraps at `width` columns. But every scrollable
pane reserves columns for a scrollbar (1) and optionally a gutter (line numbers,
N digits + 1 space). The actual content area is:

```
contentW = width - 1(scrollbar) - gutterW(line numbers)
```

If you wrap at `width` but then call `pad(line, contentW)`, pad re-truncates the
last few characters of every long wrapped line and appends `…`. The user sees
truncation even though wrapping was enabled.

**Rule: always wrap at `contentW`, not at `width`.**

## Panes without a gutter (scrollbar only)

Wrap at `width - 1`:

```go
func (r *RequestModel) rebuild() {
    wrapW := r.width - 1
    if wrapW < 1 { wrapW = 1 }
    r.lines = buildRequestLines(r.req, r.scope, wrapW)
}
```

## Panes with a gutter (scrollbar + line numbers)

`gutterW` depends on `len(r.lines)`, which depends on wrapping width — a
chicken-and-egg. Break it with a **two-pass rebuild**:

```go
type ResponseModel struct {
    // ...
    contentW int // set by rebuild(), used by View() for pad()
}

func (r *ResponseModel) rebuild() {
    if r.width <= 0 { return }

    // Pass 1: build at full width → line count → gutterW → exact contentW
    pre := buildLines(r.resp, r.resolved, r.width)
    nLines := len(pre)
    if nLines < 1 { nLines = 1 }
    gutterW := len(fmt.Sprintf("%d", nLines)) + 1 // digits + 1 separator space
    r.contentW = r.width - 1 - gutterW             // -1 scrollbar
    if r.contentW < 20 { r.contentW = 20 }

    // Pass 2: rebuild at exact content width — pad() will never truncate
    r.lines = buildLines(r.resp, r.resolved, r.contentW)
    r.recomputeMatches()
}
```

Store `contentW` on the model. In `View()`, use `r.contentW` directly instead
of recomputing from `gutterW`:

```go
// In View():
gutterW  := len(fmt.Sprintf("%d", len(r.lines))) + 1 // still needed for rendering gutter
contentW := r.contentW                                // ← use stored value, not width-1-gutterW
if contentW < 1 { contentW = 1 }
```

The preliminary and final line counts may differ by a few lines (narrower wrap →
more lines), but `gutterW` rarely changes by more than 1 digit. The minor
mismatch is invisible to the user.

## The `wrapLines` helper

```go
import "github.com/charmbracelet/x/ansi"

func wrapLines(s string, width int) string {
    if width <= 0 { return s }
    // preserveSpace=true keeps leading whitespace (indented JSON, HTTP headers…)
    return ansi.Hardwrap(s, width, true)
}
```

`ansi.Hardwrap` preserves ANSI escape sequences, accounts for wide characters,
and keeps leading spaces — essential for syntax-highlighted, indented content.

## Do not use this for tree/navigation panes

Tree rows should stay truncated. Wrapping a tree node would:
- Break the visual hierarchy (indentation communicates depth)
- Corrupt cursor-per-row alignment (the model has one entry per row)
- Make the scrollbar thumb position incorrect
