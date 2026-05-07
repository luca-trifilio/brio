<div align="center">

# brio

**A vim-style TUI for [Bruno](https://www.usebruno.com/) API collections**

[![Latest Release](https://img.shields.io/github/v/release/luca-trifilio/brio?style=flat-square&color=a6da95)](https://github.com/luca-trifilio/brio/releases/latest)
[![Go](https://img.shields.io/badge/go-1.22+-00ADD8?style=flat-square&logo=go&logoColor=white)](https://go.dev/)
[![CI](https://img.shields.io/github/actions/workflow/status/luca-trifilio/brio/release.yml?style=flat-square&label=CI&color=8aadf4)](https://github.com/luca-trifilio/brio/actions)
[![License: MIT](https://img.shields.io/badge/license-MIT-c6a0f6?style=flat-square)](LICENSE)
[![Homebrew](https://img.shields.io/badge/homebrew-luca--trifilio%2Fbrio-f5a97f?style=flat-square&logo=homebrew&logoColor=white)](https://github.com/luca-trifilio/homebrew-brio)

</div>

---

brio reads your `.bru` files directly from disk, executes HTTP requests with full variable interpolation and AWS SigV4 signing, and handles Satispay breakglass credential refresh automatically — all from the terminal.

> **Read-only by design.** brio never writes to your `.bru` files.

---

## Features

- **Vim motions** — Normal / Insert / Command modes, `j/k/gg/G`, count prefixes, leap (flash.nvim-style jump)
- **Multi-collection** — open several Bruno collections side by side in one session
- **Full variable interpolation** — layered scope: collection → environment → folder chain → request → runtime overrides, with cycle detection
- **AWS SigV4 signing** — complete auth inheritance chain (request → folder → collection), credentials resolved through variable interpolation
- **Auto-breakglass** — when a PROD request returns an invalid or expired AWS token, brio suspends the TUI, runs `~/bin/breakglass.sh` interactively, injects the fresh credentials, and retries automatically
- **Environment safety tiers** — `●` safe · `▲` caution · `⚠` danger, with mutating methods (POST/PUT/PATCH) blocked in production
- **Post-response scripts** — `bru.setVar` + `res.body` path extraction stores values as runtime vars for chained requests
- **Pre-request scripts** — UUID v4 generation via `require('uuid').v4()`
- **History** — persisted request log with replay
- **Response pane** — syntax highlighting, vim scroll, in-pane search (`/`), visual selection + yank, leap jumps
- **Copy as curl** — `yc` yanks the selected request as a `curl` command to the clipboard
- **Catppuccin Macchiato** theme throughout

---

## Install

### Homebrew

```sh
brew install luca-trifilio/brio/brio
```

### Go

```sh
go install github.com/luca-trifilio/brio@latest
```

### From source

```sh
git clone https://github.com/luca-trifilio/brio.git
cd brio
go build -o brio .
```

---

## Usage

```sh
brio /path/to/collection [/path/to/another-collection ...]
```

brio discovers `bruno.json`, `collection.bru`, `environments/`, and all `.bru` request files automatically. Pass as many collection roots as you need.

---

## Key bindings

### Global

| Key | Action |
|-----|--------|
| `Tab` / `Shift+Tab` | cycle panes |
| `j` / `k` | move down / up |
| `]` / `[` | cycle environment forward / backward |
| `:env <name>` | switch environment by name |
| `yc` | copy selected request as curl |
| `V` | toggle runtime vars panel |
| `H` | open history |
| `?` | keybinding help |
| `q` / `Ctrl+C` | quit |

### Collections tree

| Key | Action |
|-----|--------|
| `l` / `h` | expand / collapse |
| `gg` / `G` | jump to top / bottom |
| `d` / `u` | half page down / up |
| `Enter` | execute request |

### Response & Request panes

| Key | Action |
|-----|--------|
| `j` / `k` | line down / up |
| `d` / `u` / `f` / `b` | half / full page |
| `gg` / `G` | top / bottom |
| `/` / `?` | search forward / backward |
| `n` / `N` | next / previous match |
| `s` | leap jump (flash.nvim style) |
| `v` | visual linewise selection |
| `y` | yank selection to clipboard |

### Environment pane

| Key | Action |
|-----|--------|
| `j` / `k` | move down / up |
| `Enter` | select environment |
| `e` | open env file in `$EDITOR` |

---

## Breakglass (Satispay PROD)

Production APIs sit behind an AWS API Gateway requiring SigV4 with a short-lived role (`api-developer--services-prod`). brio handles this transparently:

1. You execute a PROD request — credentials are missing or expired
2. brio detects the `403` + `"security token … invalid/expired"` response
3. The TUI suspends and `~/bin/breakglass.sh` runs interactively in the terminal (SSO login, Step Function execution, approval wait)
4. Once the script exits, brio reads the fresh credentials from the Bruno environment YAML written by the script, injects them as runtime variables, and retries the original request — no manual copy-paste required

Credentials persist in the session for subsequent requests until the token expires again.

---

## Bruno compatibility

brio supports the subset of the Bruno file format used in day-to-day API work:

| Feature | Status |
|---------|--------|
| All HTTP methods | ✅ |
| Variable interpolation (`{{var}}`) | ✅ |
| Auth inheritance (request → folder → collection) | ✅ |
| `auth:awsv4` | ✅ |
| `auth:bearer`, `auth:basic` | parsing only |
| `body:json`, `body:text`, `body:xml` | ✅ |
| `script:post-response` (`bru.setVar` + `res.body`) | ✅ |
| `script:pre-request` (UUID generation) | ✅ |
| `params:query`, `params:path` | ✅ |
| `settings` (timeout, encodeUrl) | ✅ |
| Multiple environments | ✅ |
| Writing `.bru` files | ✗ (by design) |

---

## License

[MIT](LICENSE)
