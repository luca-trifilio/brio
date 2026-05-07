---
name: bubbletea-async
description: Async HTTP requests, spinner animation, and loading state in charmbracelet/bubbletea. Use when running long-lived work (HTTP calls, file IO) off the Update loop with tea.Cmd, dispatching result messages, and managing in-flight/loading UI.
---

# bubbletea-async

Async HTTP requests, spinner animation, and loading state in charmbracelet/bubbletea.

## Async command pattern

```go
// 1. Define a result message type
type executeMsg struct {
    Resp httpx.Response
    Err  error
}

// 2. Build the tea.Cmd — runs in a goroutine, returns a Msg
func runRequestCmd(req *model.Request) tea.Cmd {
    return func() tea.Msg {
        resp, err := httpx.Execute(req)
        return executeMsg{Resp: resp, Err: err}
    }
}

// 3. Dispatch in Update
case executeMsg:
    m.loading = false
    m.response.SetResponse(&msg.Resp, 0, 0)
    return m, nil
```

## Spinner integration

```go
import "github.com/charmbracelet/bubbles/spinner"

type Model struct {
    spinner spinner.Model
    loading bool
}

func NewModel() *Model {
    sp := spinner.New()
    sp.Spinner = spinner.Dot          // Dot, Line, MiniDot, Points, Pulse, Globe…
    sp.Style = lipgloss.NewStyle().Foreground(theme.Yellow)
    return &Model{spinner: sp}
}

// In Update — always forward TickMsg even when not loading
case spinner.TickMsg:
    var cmd tea.Cmd
    m.spinner, cmd = m.spinner.Update(msg)
    if m.loading {
        return m, cmd   // keep ticking only while loading
    }
    return m, nil

// Start loading
func (m *Model) startRequest(req *model.Request) tea.Cmd {
    m.loading = true
    return tea.Batch(
        m.spinner.Tick,         // start animation
        runRequestCmd(req),     // fire HTTP call
    )
}

// In View
func (m *Model) renderStatus() string {
    if m.loading {
        return m.spinner.View() + " running…"
    }
    return m.statusLine
}
```

## Pass dimensions lazily

When setting response data from the `executeMsg` handler, dimensions aren't known yet.
Pass `0, 0` and let `View()` resize on the next render:

```go
case executeMsg:
    m.response.SetResponse(&msg.Resp, 0, 0)

// In ResponseModel.View():
func (r *ResponseModel) View(width, height int, focused bool) string {
    if width != r.width || height != r.height {
        r.Resize(width, height)  // lazy resize here
    }
    // ...
}
```
