# Environment safety tiers

## Classification

```go
type EnvTier int
const (
    TierSafe    EnvTier = iota // local, test, dev, …
    TierCaution                // staging, uat, pre-prod, …
    TierDanger                 // prod, production, live, …
)

func ClassifyEnv(name string) EnvTier {
    lower := strings.ToLower(name)
    switch {
    case strings.Contains(lower, "prod") || strings.Contains(lower, "live"):
        return TierDanger
    case strings.Contains(lower, "stag") || strings.Contains(lower, "uat") ||
        strings.Contains(lower, "pre"):
        return TierCaution
    default:
        return TierSafe
    }
}
```

## Visual badges

Put these in the `theme` package so they're available everywhere:

```go
// EnvTierIcon returns a styled single-character tier indicator.
//   ● green  → safe   (local / test / dev)
//   ▲ orange → caution (staging / uat / pre-prod)
//   ⚠ red   → danger  (prod / live)
func EnvTierIcon(name string) string {
    switch ClassifyEnv(name) {
    case TierDanger:
        return lipgloss.NewStyle().Foreground(Red).Bold(true).Render("⚠")
    case TierCaution:
        return lipgloss.NewStyle().Foreground(Peach).Render("▲")
    default:
        return lipgloss.NewStyle().Foreground(Green).Render("●")
    }
}

func EnvTierStyle(name string) lipgloss.Style {
    switch ClassifyEnv(name) {
    case TierDanger:  return lipgloss.NewStyle().Foreground(Red).Bold(true)
    case TierCaution: return lipgloss.NewStyle().Foreground(Peach)
    default:          return lipgloss.NewStyle().Foreground(Green)
    }
}

// EnvBadge renders "icon name" for inline use (e.g. status bar).
func EnvBadge(name string) string {
    if name == "" { return StyleDim.Render("(no env)") }
    return EnvTierIcon(name) + " " + EnvTierStyle(name).Render(name)
}
```

## Tier-aware sort (safe → caution → danger, then alpha)

Export as `SortEnvNames` for use in both the env pane model and `NewModel`:

```go
func SortEnvNames(names []string) {
    sort.Slice(names, func(i, j int) bool {
        ti := theme.ClassifyEnv(names[i])
        tj := theme.ClassifyEnv(names[j])
        if ti != tj { return ti < tj }
        return strings.ToLower(names[i]) < strings.ToLower(names[j])
    })
}
```

Call in both `NewEnv(c)` and `SetCollection(c)` instead of `sort.Strings`.
Also call in `NewModel` when seeding `activeEnvs` so the default is the safest env.

## Blocking mutating methods in danger envs

### tree model

```go
type TreeModel struct {
    BlockedMethods map[string]bool // e.g. {"POST": true, "PUT": true, "PATCH": true}
    // ...
}

func (t *TreeModel) SetBlockedMethods(methods map[string]bool) {
    t.BlockedMethods = methods
    t.Rebuild()
    t.clamp()
}

// In appendFolder, skip blocked requests:
for _, r := range f.Requests {
    if t.BlockedMethods[string(r.Method)] { continue }
    t.rows = append(t.rows, TreeNode{ /* request row */ })
}
```

### Sync helper (call on every env change)

```go
func (m *Model) syncBlockedMethods() {
    if theme.MutatingMethodsBlocked(m.activeEnvName()) {
        m.tree.SetBlockedMethods(map[string]bool{"POST": true, "PUT": true, "PATCH": true})
    } else {
        m.tree.SetBlockedMethods(nil)
    }
}

func IsMutatingMethod(method string) bool {
    switch method { case "POST", "PUT", "PATCH": return true }
    return false
}

func MutatingMethodsBlocked(envName string) bool {
    return ClassifyEnv(envName) == TierDanger
}
```

Call `syncBlockedMethods()` from: `syncEnvPane()`, `selectEnv()`, `cycleEnv()`,
`:env` command handler, and `NewModel()` startup.

### Hard guard in executeSelected and replayHistory

```go
if theme.IsMutatingMethod(string(sel.Request.Method)) &&
    theme.MutatingMethodsBlocked(m.activeEnvName()) {
    m.statusLn = "⚠ " + string(sel.Request.Method) + " blocked in production"
    return m, nil
}
```

The tree already hides blocked requests; the guard is a safety net for history
replay, which can execute arbitrary requests regardless of tree state.

## Env pane rendering with tier colours

```go
for i, n := range e.names {
    icon      := theme.EnvTierIcon(n)
    nameStyle := theme.EnvTierStyle(n)

    activeMarker := "  "
    if n == active {
        activeMarker = nameStyle.Render("● ")
    }

    line := activeMarker + icon + " " + nameStyle.Render(n)
    if focused && i == e.cursor {
        line = cursorStyle.Render(truncate(stripStyle(line), width))
    }
    b.WriteString(line + "\n")
}
```
