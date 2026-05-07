# Smart View() change detection

## Problem

A content pane (e.g. Request pane) depends on two external inputs:
- the currently selected `*model.Request` (changes when tree cursor moves)
- the active environment name (changes when user switches env)

The naive approach — calling `SetRequest(req, scope)` explicitly from every
code path that touches either input — is brittle and easy to miss.

## Solution: detect changes inside View()

Track the last-seen request pointer and env name on the model. In `View()`,
compare them to the incoming arguments and rebuild only when something changed.

```go
type RequestModel struct {
    req     *model.Request
    scope   *interp.VarScope
    lastEnv string // last env name used for line-building

    lines  []string
    offset, cursor, height, width, count int
}

func (r *RequestModel) View(
    req      *model.Request,
    scope    *interp.VarScope,
    envName  string,
    width, height int,
    focused  bool,
) string {
    reqChanged := req != r.req
    envChanged := envName != r.lastEnv
    dimChanged := width != r.width || height != r.height

    switch {
    case reqChanged:
        // New request selected — reset scroll to top and rebuild.
        r.req, r.scope, r.lastEnv = req, scope, envName
        r.width, r.height = width, height
        r.offset, r.cursor, r.count = 0, 0, 0
        r.rebuild()
    case envChanged || dimChanged:
        // Same request, different env or size — rebuild but keep scroll position.
        r.scope, r.lastEnv = scope, envName
        r.width, r.height = width, height
        r.rebuild()
        r.clamp()
    }
    // render from r.lines ...
}
```

## Why not compare scope pointers?

`resolveRequest(c, env, r, vars)` is called on every `activeRequestAndScope()`
invocation (i.e. every frame). It returns a new `*interp.VarScope` pointer each
time, so pointer equality always signals "changed" — triggering a rebuild every
frame and resetting scroll on every tick.

**Compare by env name string instead.** Env names change rarely (only on explicit
env switches), so rebuilds are triggered only when content actually changes.

## Reset scroll only on request change

- **Request changes** (`reqChanged = true`): reset `offset` and `cursor` to 0.
  The user selected a different endpoint — start from the top.
- **Env or resize** (`envChanged || dimChanged`): keep scroll position.
  The user is reading the same request; just reformat it.

## Caller side (layout.go)

Pass all three inputs on every frame — the model handles the diffing:

```go
req, scope := m.activeRequestAndScope()
reqView = m.request.View(req, scope, m.activeEnvName(), rightW-4, reqHeight-2, focused)
```

No explicit `SetRequest()` needed anywhere in the app.
