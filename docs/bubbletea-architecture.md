# Bubble Tea Architecture Best Practices

Reference for structuring brio's TUI layer as the codebase grows toward v2.

---

## Component Model

Each pane is its own Go file with its own `type Model struct`, `Init()`, `Update()`, and `View()` methods. The root application model holds all pane models as struct fields. Routing lives entirely in the root `Update()`.

```
internal/tui/
  app.go          ← root Model, top-level Update/View
  panes/
    tree.go       ← collection tree pane
    request.go    ← request detail pane
    response.go   ← response pane
    env.go        ← environment switcher
    history.go    ← history pane
    settings.go   ← settings pane
    diagnostics.go← diagnostics pane (new in v2)
    picker.go     ← collection fuzzy picker overlay (new in v2)
```

Standard bubbles components (`viewport.Model`, `list.Model`, `textinput.Model`) are embedded directly inside each pane's model struct and delegated to in `Update()`.

---

## Message Flow

Bubble Tea is top-down: every `tea.Msg` arrives at the root model first.

```
Root Update(msg)
  ├── route to active pane: pane.Update(msg) → (pane, cmd)
  ├── inspect msg for cross-pane concerns (window resize, env switch, etc.)
  └── batch returned cmds: tea.Batch(cmd1, cmd2, ...)
```

**Sibling-to-sibling communication** is done via the parent. A pane emits a custom message type; the root observes it and updates both siblings. Panes never reference each other directly.

```go
// pane emits
type EnvSwitchedMsg struct{ Name string }

// root observes
case EnvSwitchedMsg:
    m.activeEnv = msg.Name
    m.request, cmd1 = m.request.Update(msg)
    m.response, cmd2 = m.response.Update(msg)
    return m, tea.Batch(cmd1, cmd2)
```

---

## Focus Management

Only one pane receives keyboard input at a time. Track focus with an enum in the root model.

```go
type focusedPane int
const (
    focusTree focusedPane = iota
    focusRequest
    focusResponse
    // ...
)
```

On focus change, send `Focus()`/`Blur()` to the affected panes so they can pause/resume animations and update their cursor appearance. `Tab`/`Shift+Tab` cycles focus; vim-style pane jumps set it directly.

Window resize (`tea.WindowSizeMsg`) is broadcast to **all** panes, not just the focused one. Each pane recalculates its dimensions. Use `tea.WindowSizeMsg` as the trigger to initialize `viewport.Model` — never initialize viewports with zero size.

---

## Modal / Overlay Pattern

Modals (fuzzy picker, dialogs) sit above the main layout. The root model holds an `overlay` field and a boolean `overlayVisible`.

```go
type Model struct {
    // ...panes...
    picker         panes.PickerModel
    overlayVisible bool
}
```

In `Update()`, when the overlay is visible, route all input to the overlay model first:

```go
if m.overlayVisible {
    m.picker, cmd = m.picker.Update(msg)
    // check for picker close/confirm messages
    return m, cmd
}
// otherwise route to active pane
```

In `View()`, render the base layout first, then composite the overlay on top using `lipgloss.Place` or the `bubbletea-overlay` helper. The overlay must explicitly block input to the background — do not route to both simultaneously.

---

## Commands (tea.Cmd) Discipline

- Commands are the only way to trigger side effects (HTTP requests, file reads, hook execution).
- Always return commands from `Update()` — never call goroutines or blocking I/O directly.
- Batch multiple commands at the root level with `tea.Batch()`.
- Long-running ops (HTTP, hook scripts) return a `tea.Cmd` that sends a result message when done. The pane shows a spinner while waiting.
- Never store `tea.Cmd` in model state — they are fire-and-forget.

---

## Testing

**Unit tests on `Update()`** are the primary tool. Instantiate a model, call `model.Update(msg)`, assert on the returned model state and command type. This stays fast and doesn't require a running terminal.

```go
m := panes.NewTree(testCollection)
next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
assert.Equal(t, 1, next.(panes.TreeModel).Cursor())
assert.Nil(t, cmd)
```

For integration-style tests (full program flow), use `teatest` (`charmbracelet/x/exp/teatest`): start the program, send key events, assert on `FinalModel()` or partial output.

For snapshot-style view tests, use `catwalk`: apply a sequence of messages, verify the rendered `View()` string against a golden file.

**Do not test `View()` output for layout pixel-perfection** — terminal width varies. Test that expected strings *appear* in the output, not their position.

---

## Key Patterns to Preserve from Current brio

- **Mode enum** (`normal`, `insert`, `command`) lives in the root model, not in individual panes. Panes observe it but do not own it.
- **Vim count prefix** accumulation belongs in the root model; panes receive the final resolved action.
- **Status line** is rendered by the root `View()`, not by any pane. Panes emit `StatusMsg` custom messages.
- **Leap (flash.nvim-style jump)** is a cross-pane concern — it lives in the root model and coordinates with whichever pane is focused.
