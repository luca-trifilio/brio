---
name: bubbletea-model
description: Core tea.Model interface, message types, and program setup for charmbracelet/bubbletea v1. Use when scaffolding a new Bubble Tea app, structuring Init/Update/View, defining custom message types, or wiring tea.NewProgram options.
---

# bubbletea-model

Core tea.Model interface, message types, and program setup for charmbracelet/bubbletea v1.

## tea.Model interface

```go
type Model interface {
    Init() tea.Cmd          // called once at startup; return initial command or nil
    Update(tea.Msg) (tea.Model, tea.Cmd)  // handle messages, return new model + command
    View() string           // render current state to string; called after every Update
}
```

## Key message types

```go
// Keyboard input — match with msg.String()
case tea.KeyMsg:
    switch msg.String() {
    case "ctrl+c", "q": return m, tea.Quit
    case "enter":       // confirm
    case "esc":         // cancel
    case "tab":         // next pane
    case "j", "down":   // move down
    case "k", "up":     // move up
    case "l", "right":  // expand / right
    case "h", "left":   // collapse / left
    case ":":           // enter command mode
    }

// Terminal resize — always handle this
case tea.WindowSizeMsg:
    m.width, m.height = msg.Width, msg.Height
    return m, nil
```

## Commands

```go
tea.Quit                          // exit the program
tea.Batch(cmd1, cmd2, ...)        // run commands concurrently
tea.Sequence(cmd1, cmd2, ...)     // run commands sequentially
tea.Tick(d, func(t time.Time) tea.Msg { ... })  // periodic timer

// Custom async command:
func doWork() tea.Cmd {
    return func() tea.Msg {
        result := expensiveOperation()
        return myResultMsg{result}
    }
}
```

## Program setup

```go
p := tea.NewProgram(
    initialModel,
    tea.WithAltScreen(),          // full-screen mode
    tea.WithMouseCellMotion(),    // mouse support
    tea.WithFPS(60),              // render frame rate
)
if _, err := p.Run(); err != nil {
    log.Fatal(err)
}
```

## Modal key dispatch pattern (used in bruno-tui)

```go
type Mode int
const (
    ModeNormal Mode = iota
    ModeCommand
    ModeInsert
)

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.WindowSizeMsg:
        m.width, m.height = msg.Width, msg.Height
        return m, nil
    case tea.KeyMsg:
        return m.handleKey(msg)
    }
    return m, nil
}

func (m *Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
    switch m.mode {
    case ModeNormal:  return m.handleNormalKey(msg)
    case ModeCommand: return m.handleCommandKey(msg)
    case ModeInsert:  return m.handleInsertKey(msg)
    }
    return m, nil
}
```

## Two-key sequence pattern (e.g. gg)

Use a `pendingX bool` flag in the model:

```go
if m.pendingG {
    m.pendingG = false
    if msg.String() == "g" {
        // execute gg action
        return m, nil
    }
    // not a gg — fall through
}
if msg.String() == "g" {
    m.pendingG = true
    return m, nil
}
```
