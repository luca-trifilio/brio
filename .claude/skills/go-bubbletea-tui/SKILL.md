---
name: go-bubbletea-tui
description: Patterns for building vim-style modal TUIs in Go with Bubble Tea + Lip Gloss. Use when building terminal UIs with Normal/Insert/Command modes, multi-pane layouts, async HTTP execution, vim scroll in panes, cursor + relative line numbers, leap/flash motion, in-pane search, proportional scrollbar, context-sensitive help modal, accordion tree, and centralized theming.
---

# Go Bubble Tea TUI Patterns

## Dependencies

```go
// go.mod
require (
    github.com/charmbracelet/bubbletea v0.x
    github.com/charmbracelet/bubbles  v0.x
    github.com/charmbracelet/lipgloss v0.x
    github.com/spf13/cobra            v1.x
    github.com/atotto/clipboard       v0.x
    github.com/aws/aws-sdk-go-v2      v1.x  // if AWS SigV4 needed
)
```

## Modal state machine

Define mode and pane enums, then dispatch keys per mode:

```go
type Mode int
const (
    ModeNormal Mode = iota
    ModeInsert
    ModeCommand
)

type Model struct {
    mode        Mode
    focusedPane Pane
    commandBuf  string
    statusLine  string
    // ... data fields
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        switch m.mode {
        case ModeNormal:  return m.handleNormalKey(msg)
        case ModeCommand: return m.handleCommandKey(msg)
        case ModeInsert:  return m.handleInsertKey(msg)
        }
    case tea.WindowSizeMsg:
        m.width, m.height = msg.Width, msg.Height
        return m, nil
    }
    return m, nil
}
```

## Command line (`:` mode)

```go
func (m Model) handleNormalKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
    switch msg.String() {
    case ":":
        m.mode = ModeCommand
        m.commandBuf = ""
    case "j": m.moveDown()
    case "k": m.moveUp()
    case "l": m.expand()
    case "h": m.collapse()
    case "enter": return m, m.executeRequest()
    }
    return m, nil
}

func (m Model) handleCommandKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
    switch msg.String() {
    case "enter":
        return m.executeCommand(m.commandBuf)
    case "esc":
        m.mode = ModeNormal
        m.commandBuf = ""
    case "backspace":
        if len(m.commandBuf) > 0 {
            m.commandBuf = m.commandBuf[:len(m.commandBuf)-1]
        }
    default:
        m.commandBuf += msg.String()
    }
    return m, nil
}
```

## Lip Gloss multi-pane layout

```go
func (m Model) View() string {
    statusBar := lipgloss.NewStyle().
        Width(m.width).
        Background(lipgloss.Color("62")).
        Render(fmt.Sprintf(" %s | %s ", m.mode, m.activeEnv))

    sidebar := lipgloss.JoinVertical(lipgloss.Left,
        m.treePane.View(m.width/4, m.height-4),
        m.envPane.View(m.width/4, 3),
    )

    rightTop := m.requestPane.View(3*m.width/4, (m.height-4)/2)
    rightBot := m.responsePane.View(3*m.width/4, (m.height-4)/2)
    right := lipgloss.JoinVertical(lipgloss.Left, rightTop, rightBot)

    body := lipgloss.JoinHorizontal(lipgloss.Top, sidebar, right)
    cmdLine := m.renderCommandLine()

    return lipgloss.JoinVertical(lipgloss.Left, statusBar, body, cmdLine)
}
```

Always handle `tea.WindowSizeMsg` in every pane — pass width/height down on resize.

## Status bar layout (top + bottom)

Two-bar pattern: top bar for context (collection, env), bottom bar for mode + status + hints.

```go
func (m *Model) renderStatusBar() string {
    // Top: context only — no mode indicator here
    content := collName + theme.StyleDim.Render(" │ env=") + theme.StyleActive.Render(m.activeEnvName())
    return theme.StyleStatusBar.Width(m.width).Render(content)
}

func (m *Model) renderCommandLine() string {
    if m.mode == ModeCommand {
        return ansi.Truncate(m.cmd.View(), m.width, "")
    }
    left := modeStyle.Render(m.mode.String())   // NORMAL / INSERT / COMMAND
    leftW := lipgloss.Width(left)
    center := m.statusLn                         // e.g. "200 OK in 142ms"
    centerW := lipgloss.Width(center)
    right := theme.StyleHint.Render("? help")
    rightW := lipgloss.Width(right)

    remaining := m.width - leftW - rightW
    padLeft := (remaining - centerW) / 2
    if padLeft < 1 { padLeft = 1 }
    padRight := remaining - centerW - padLeft
    if padRight < 0 { padRight = 0 }
    return left + strings.Repeat(" ", padLeft) + center + strings.Repeat(" ", padRight) + right
}
```

**Key rule**: always show mode indicator even in Normal — it pays off the moment insert/editing is added.

## Vim scroll in read-only panes

**Critical bug to avoid**: never use `append(subSlice, item)` where `subSlice = bigSlice[a:b]` — Go will write `item` into `bigSlice[b]`, corrupting backing data. Always copy first:

```go
visible := make([]string, viewH)
copy(visible, r.lines[r.offset:end])  // safe — own backing array
```

Full stateful model with cursor (separate from offset), numeric prefix, and correct `viewH`:

```go
type ResponseModel struct {
    lines  []string
    offset int   // first visible line
    cursor int   // highlighted line (absolute)
    height int
    width  int
    count  int   // numeric prefix (5j = move 5)
}

func (r *ResponseModel) viewH() int {
    h := r.height - 3  // subtract header(2) + scrollbar-hint(1)
    if h < 1 { h = 1 }
    return h
}

func (r *ResponseModel) clamp() {
    // IMPORTANT: use viewH(), NOT height — clamp() called before View() computes viewH
    max := len(r.lines) - r.viewH()
    if max < 0 { max = 0 }
    if r.offset > max { r.offset = max }
    if r.offset < 0  { r.offset = 0  }
    if r.cursor >= len(r.lines) && len(r.lines) > 0 { r.cursor = len(r.lines) - 1 }
    if r.cursor < 0 { r.cursor = 0 }
    // Keep cursor visible — scroll to follow
    if r.cursor < r.offset { r.offset = r.cursor }
    if r.cursor >= r.offset+r.viewH() { r.offset = r.cursor - r.viewH() + 1 }
}

func (r *ResponseModel) HandleKey(key string) bool {
    // ... numeric prefix accumulation ...
    n := r.n(); r.count = 0
    switch key {
    case "j", "down":   r.cursor += n
    case "k", "up":     r.cursor -= n
    case "d", "ctrl+d": r.cursor += r.halfPage()
    case "u", "ctrl+u": r.cursor -= r.halfPage()
    case "f", "ctrl+f": r.cursor += r.viewH()
    case "b", "ctrl+b": r.cursor -= r.viewH()
    case "g":           r.cursor = 0; r.offset = 0
    case "G":
        if n > 1 { r.cursor = n - 1 } else { r.cursor = len(r.lines) - 1 }
    default: return false
    }
    r.clamp()
    return true
}
```

**Two-key sequences (e.g. `gg`, `yc`)** — handle via pending flags in the root model. **Critical ordering rule**: pending-flag checks must come BEFORE pane delegation, otherwise the second key gets consumed by the pane's `HandleKey` instead of completing the sequence.

```go
// CORRECT order: pending checks first, then pane delegation
if m.pendingG {
    m.pendingG = false
    if s == "g" && m.focused == PaneResponse {
        m.response.HandleKey("g")
        return m, nil
    }
    // not gg — discard pending, fall through
}
if m.pendingYank {
    m.pendingYank = false
    if s == "c" {
        return m.copyCurl()
    }
    // not yc — discard pending, fall through
}
if m.focused == PaneResponse {
    if m.response.Searching() { m.response.HandleKey(s); return m, nil }
    if s == "g" { m.pendingG = true; return m, nil }
    if m.response.HandleKey(s) { return m, nil }
}
// global switch
switch s {
case "y": m.pendingYank = true; return m, nil
// ...
}
```

**Visual feedback**: show the pending key in the status bar so the user knows the TUI received the first key. Without this, users think nothing happened and press `y` again.

```go
// In renderCommandLine():
pendingKey := ""
if m.pendingYank { pendingKey = "y" } else if m.pendingG { pendingKey = "g" }
if pendingKey != "" {
    hint = theme.StyleActive.Render(pendingKey) + theme.StyleHint.Render("…  ? help")
} else {
    hint = theme.StyleHint.Render("? help")
}
```

## Relative line numbers

Gutter on the left: cursor line = absolute number (bold), all others = distance from cursor (dim).

```go
gutterW := len(fmt.Sprintf("%d", len(r.lines))) + 1  // digits + 1 space
contentW := width - 1 - gutterW                       // -1 for scrollbar

for i, line := range visible {
    absLine := r.offset + i
    var gutterStr string
    if absLine == r.cursor {
        n := fmt.Sprintf("%*d ", gutterW-1, absLine+1)
        gutterStr = gutterCursorStyle.Render(n)  // bold, accent color
    } else {
        rel := absLine - r.cursor
        if rel < 0 { rel = -rel }
        n := fmt.Sprintf("%*d ", gutterW-1, rel)
        gutterStr = gutterDimStyle.Render(n)
    }
    rows[i] = gutterStr + cell + bar
}
```

## Proportional scrollbar

1-column track on the right edge. Thumb (`┃`) position and height are proportional to content.

```go
func buildScrollbar(offset, totalLines, viewH int) []string {
    bar := make([]string, viewH)
    track := theme.StyleDim.Render("│")
    thumb := lipgloss.NewStyle().Foreground(theme.Overlay2).Render("┃")

    if totalLines <= viewH {
        for i := range bar { bar[i] = track }
        return bar
    }

    thumbH := viewH * viewH / totalLines
    if thumbH < 1 { thumbH = 1 }
    maxOffset := totalLines - viewH
    thumbTop := offset * (viewH - thumbH) / maxOffset

    for i := range bar {
        if i >= thumbTop && i < thumbTop+thumbH { bar[i] = thumb } else { bar[i] = track }
    }
    return bar
}
```

Reserve 1 column: `contentW := width - 1`. Append bar char to each row: `rows[i] = cell + bar[i]`.

Scroll indicator shows cursor line number, not offset:
```go
fmt.Sprintf("── %d%% (%d/%d lines)", pct, r.cursor+1, len(r.lines))
```

## In-pane search (`/` and `?`)

```go
type ResponseModel struct {
    // ...
    searchBuf     string
    searchQuery   string
    searchForward bool
    searching     bool
    matches       []int
    matchIdx      int
}

// In HandleKey:
case "/": r.searching = true; r.searchForward = true; r.searchBuf = ""
case "?": r.searching = true; r.searchForward = false; r.searchBuf = ""
case "n": r.jumpToMatch(r.searchForward)
case "N": r.jumpToMatch(!r.searchForward)

// When searching==true, absorb all keys including q/ctrl+c:
if r.searching {
    switch key {
    case "enter": r.searchQuery = r.searchBuf; r.searching = false; r.recomputeMatches(); r.jumpToMatch(r.searchForward)
    case "esc":   r.searching = false; r.searchBuf = ""
    case "backspace": r.searchBuf = r.searchBuf[:len-1]
    default: if len(key)==1 { r.searchBuf += key }
    }
    return true
}
```

**Root model must check `Searching()` before intercepting `q`/`ctrl+c`:**
```go
if m.focused == PaneResponse {
    if m.response.Searching() { m.response.HandleKey(s); return m, nil }
    switch s {
    case "q", "ctrl+c": return m, tea.Quit
    }
    if m.response.HandleKey(s) { return m, nil }
}
```

Match highlighting in View:
```go
matchStyle       := lipgloss.NewStyle().Background(theme.Surface0).Foreground(theme.Yellow)
activeMatchStyle := lipgloss.NewStyle().Background(theme.Yellow).Foreground(theme.Base)

// Per row: highlight if line matches query, use activeMatchStyle for current match
```

Bottom bar shows search prompt while typing (`/foo█`) and match count after (`[2/11]`).

## Leap motion (flash.nvim style)

`s` → type chars to filter → labeled targets overlay → press label to jump.

```go
const leapLabels = "asdfjklghqwertyuiopzxcvbnmASDFJKLGHQWERTYUIOPZXCVBNM"

type leapTarget struct {
    absLine int
    col     int    // byte offset of match in plain text
    label   string // single char shown at position
}

// In HandleKey when leaping==true:
// - printable key + no targets yet: append to leapChar, rebuild targets
// - printable key + targets exist: check if it's a label → jump; else append to leapChar
// - backspace: remove last char, rebuild targets  
// - enter: jump to first target
// - esc: cancel
// Auto-jump when only 1 target remains after each keystroke.
```

Render: dim all non-matching lines, replace matched char with colored label badge:
```go
leapDimStyle   := lipgloss.NewStyle().Foreground(theme.Surface2)
leapLabelStyle := lipgloss.NewStyle().Background(theme.Mauve).Foreground(theme.Base).Bold(true)
```

Bottom bar during leap: `s›foo█  3 targets — type more or a label, enter=first, esc=cancel`

## Context-sensitive help modal (`?` / which-key)

```go
type HelpSection struct {
    Title   string
    Entries []HelpEntry  // {Key, Desc}
}

type HelpModel struct {
    Visible  bool
    sections []HelpSection
}

func (h *HelpModel) Open(sections []HelpSection) { h.sections = sections; h.Visible = true }
func (h *HelpModel) Close()                       { h.Visible = false }
```

Root model: `helpSections()` returns pane-specific sections + global sections appended.

```go
func (m *Model) helpSections() []HelpSection {
    switch m.focused {
    case PaneTree:     return append(TreeSections(), GlobalSections()...)
    case PaneResponse: return append(ResponseSections(), GlobalSections()...)
    // etc.
    }
}
```

Key dispatch: help modal absorbs ALL keys when visible (check before history/vars):
```go
if m.help.Visible {
    switch msg.String() {
    case "esc", "q", "?": m.help.Close()
    }
    return m, nil
}
```

Trigger: `case "?"` in normal key handler → `m.help.Open(m.helpSections())`.

Show `? help` hint in bottom-right of the command line.

Render via the same `overlay()` helper used for history/vars modals.

## Accordion tree (single-open collections)

When expanding a collection node, close all sibling collections first:

```go
func (t *TreeModel) Expand() {
    n, ok := t.Selected()
    if !ok || !n.Expandable { return }
    if n.Kind == NodeCollection && !t.Expanded[n.Path] {
        for _, c := range t.Collections {
            if c.Path != n.Path { t.Expanded[c.Path] = false }
        }
    }
    t.Expanded[n.Path] = true
    t.Rebuild()
}
```

Only applies to top-level collection nodes — folder expansion within a collection is unaffected.

## Centralized theme package (Catppuccin Macchiato)

Create `internal/theme/theme.go` with palette constants and semantic styles. Never scatter `lipgloss.Color("244")` raw values across panes.

```go
package theme

import "github.com/charmbracelet/lipgloss"

// Catppuccin Macchiato
const (
    Rosewater = lipgloss.Color("#f4dbd6")
    Peach     = lipgloss.Color("#f5a97f")
    Yellow    = lipgloss.Color("#eed49f")
    Green     = lipgloss.Color("#a6da95")
    Sky       = lipgloss.Color("#91d7e3")
    Blue      = lipgloss.Color("#8aadf4")
    Lavender  = lipgloss.Color("#b7bdf8")
    Mauve     = lipgloss.Color("#c6a0f6")
    Red       = lipgloss.Color("#ed8796")
    Text      = lipgloss.Color("#cad3f5")
    Subtext0  = lipgloss.Color("#a5adcb")
    Overlay2  = lipgloss.Color("#939ab7")
    Overlay1  = lipgloss.Color("#8087a2")
    Overlay0  = lipgloss.Color("#6e738d")
    Surface2  = lipgloss.Color("#5b6078")
    Surface1  = lipgloss.Color("#494d64")
    Surface0  = lipgloss.Color("#363a4f")
    Base      = lipgloss.Color("#24273a")
    Mantle    = lipgloss.Color("#1e2030")
)

var (
    StyleTitle      = lipgloss.NewStyle().Foreground(Lavender).Bold(true)
    StyleCollection = lipgloss.NewStyle().Foreground(Peach).Bold(true)
    StyleText       = lipgloss.NewStyle().Foreground(Text)
    StyleDim        = lipgloss.NewStyle().Foreground(Overlay1)
    StyleActive     = lipgloss.NewStyle().Foreground(Green).Bold(true)
    StyleSuccess    = lipgloss.NewStyle().Foreground(Green)
    StyleError      = lipgloss.NewStyle().Foreground(Red)
    StyleWarning    = lipgloss.NewStyle().Foreground(Yellow)
    StyleHint       = lipgloss.NewStyle().Foreground(Overlay1)

    StyleCursorLine = lipgloss.NewStyle().Background(Surface1).Foreground(Sky).Bold(true)

    BorderFocused   = Blue
    BorderUnfocused = Surface2
)
```

**Key rules:**
- `StyleCursorLine` — use in every pane for the selected row. Uniform color = uniform UX.
- `StyleCollection` (Peach) vs `StyleTitle` (Lavender) — different colors so they're visually distinct.
- `MethodStyle(method string)` function returns per-method color (GET=Green, POST=Blue, PUT=Yellow, PATCH=Peach, DELETE=Red).

## Stateful env pane with cross-pane sync

When multiple data sources can change what a pane shows (e.g. env pane depends on the currently selected collection), use `SetCollection` + `SyncCursor`:

```go
func (e *EnvModel) SetCollection(c *model.Collection) {
    e.names = nil
    for n := range c.Environments { e.names = append(e.names, n) }
    sort.Strings(e.names)
    if e.cursor >= len(e.names) { e.cursor = 0 }
}

func (e *EnvModel) SyncCursor(active string) {
    for i, n := range e.names {
        if n == active { e.cursor = i; return }
    }
}
```

Call `syncEnvPane()` from the root model whenever the selected collection may have changed.

## Async HTTP with spinner

```go
type executeMsg struct{ Resp httpx.Response }

func runRequestCmd(...) tea.Cmd {
    return func() tea.Msg {
        resp := httpx.Execute(req)
        return executeMsg{resp}
    }
}

// In Update:
case executeMsg:
    m.loading = false
    m.response.SetResponse(&msg.Resp, 0, 0) // dimensions filled on next View call
```

Pass `width=0, height=0` when setting from the message handler — `View()` will resize on the next render call.

## Overlay panels (modal pattern)

All modals (history, vars, help) share the same pattern:

```go
type SomeModal struct { Visible bool; /* data */ }

// In layout.go, after building body:
if m.someModal.Visible {
    ov := m.someModal.View(m.width-8, m.height-4)
    body = overlay(body, ov, m.width, m.height)
}

// overlay() manually blits modal lines over body lines at centered position
```

Key dispatch order matters — check modals in priority order (help → history → vars → mode dispatch).

## Clipboard

```go
import "github.com/atotto/clipboard"
clipboard.WriteAll(curlString)
```

Note: on macOS uses `pbcopy`; on Linux requires `xclip` or `xsel`.

**curl generation: never put `#` comments inline on a continuation line.** In a `\`-continued shell command, `#` comments out everything after it on that line — including the URL if they're on the same line. Put comments on their own line before the `curl` command:

```go
// WRONG — URL gets commented out:
// curl \
//   # AWS SigV4 auth http://example.com

// CORRECT — comment on its own line:
if r.AuthMode == "awsv4" {
    b.WriteString("# AWS SigV4 auth — set credentials in env file\n")
}
b.WriteString("curl")
// ... flags ...
b.WriteString(" " + shellQuote(finalURL))
```

## goreleaser + Homebrew tap

```yaml
# .goreleaser.yaml
builds:
  - goos: [darwin, linux]
    goarch: [amd64, arm64]
    ldflags: ["-s -w -X main.version={{.Version}}"]
archives:
  - format: tar.gz
brews:
  - name: bruno-tui
    repository:
      owner: luca-trifilio
      name: homebrew-bruno-tui
      token: "{{ .Env.HOMEBREW_TAP_GITHUB_TOKEN }}"
```

Note: `brews` is deprecated in newer goreleaser in favor of `homebrew_casks` — check changelog when upgrading.

## Project structure

```
main.go
internal/
  cli/root.go          # cobra, loads collections, starts bubbletea
  theme/theme.go       # Catppuccin palette + semantic styles
  model/               # typed structs + loader
  httpx/               # HTTP executor
  history/             # append-only JSONL
  tui/
    app.go             # root tea.Model + key dispatch + pendingG/pendingYank/help flags
    mode.go            # Mode/Pane enums
    layout.go          # lipgloss composition + overlay() helper
    panes/
      tree.go          # accordion expand, StyleCursorLine
      env.go           # SetCollection/SyncCursor
      request.go       # pure render
      response.go      # cursor+offset, relative line numbers, scrollbar, search, leap
      vars.go          # editable key/value, StyleCursorLine
      history.go       # modal list, StyleCursorLine
      help.go          # HelpModel + per-pane HelpSection definitions
```
