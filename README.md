<div align="center">

# brio

**A vim-style TUI for [Bruno](https://www.usebruno.com/) API collections**

[![Latest Release](https://img.shields.io/github/v/release/luca-trifilio/brio?style=flat-square&color=a6da95)](https://github.com/luca-trifilio/brio/releases/latest)
[![Go](https://img.shields.io/badge/go-1.22+-00ADD8?style=flat-square&logo=go&logoColor=white)](https://go.dev/)
[![CI](https://img.shields.io/github/actions/workflow/status/luca-trifilio/brio/release.yml?style=flat-square&label=CI&color=8aadf4)](https://github.com/luca-trifilio/brio/actions)
[![License: MIT](https://img.shields.io/badge/license-MIT-c6a0f6?style=flat-square)](LICENSE)
[![Homebrew](https://img.shields.io/badge/homebrew-luca--trifilio%2Ftap-f5a97f?style=flat-square&logo=homebrew&logoColor=white)](https://github.com/luca-trifilio/homebrew-tap)

</div>

---

brio reads your `.bru` files directly from disk, executes HTTP requests with full variable interpolation and AWS SigV4 signing, and supports configurable credential-refresh hooks — all from the terminal.

> **Read-only by design.** brio never writes to your `.bru` files.

---

## Features

- **Vim motions** — Normal / Insert / Command modes, `j/k/gg/G`, count prefixes, leap (flash.nvim-style jump)
- **Multi-collection** — open several Bruno collections side by side in one session, fuzzy-pick the active one with `gc`
- **Pluggable loaders** — internal `CollectionLoader` interface decouples the TUI from any single source format (Bruno today; Postman/Insomnia targeted as plugins). Format auto-detected from the directory or set explicitly via the `format = "..."` config field
- **jq response filter** — press `|` to filter the JSON response with a [jq](https://jqlang.github.io/jq/) query; malformed queries surface as a non-fatal error, never a crash
- **Diagnostics overlay** — non-fatal loader issues are aggregated and shown via `gd`; the status bar surfaces a `⚠ N` badge when any are present
- **Full variable interpolation** — layered scope: collection → environment → folder chain → request → runtime overrides, with cycle detection
- **AWS SigV4 signing** — complete auth inheritance chain (request → folder → collection), credentials resolved through variable interpolation
- **Credential hooks** — when a response matches a configured trigger (status code, body regex, env tier), brio runs a refresh script (interactively or in the background), injects the returned credentials as runtime vars, and retries the request automatically
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
brew install luca-trifilio/tap/brio
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
| `gs` | open settings |
| `gc` | fuzzy-pick a collection |
| `gd` | open diagnostics overlay |
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
| `\|` | filter response body with [jq](https://jqlang.github.io/jq/) (Enter applies, Esc clears) |

### Environment pane

| Key | Action |
|-----|--------|
| `j` / `k` | move down / up |
| `Enter` | select environment |
| `e` | open env file in `$EDITOR` |

---

## Configuration

`~/.config/brio/config.toml` carries collection paths and credential hooks. Press `gs` inside brio, then `e`, to open it in `$EDITOR`.

```toml
[[collections]]
path = "/path/to/my-bruno-collection"
format = "bruno"           # optional — auto-detected when omitted

[[collections]]
path = "~/projects/payments"

# active_collection = "my-bruno-collection"  # optional, otherwise first wins
# theme = ""                                  # placeholder for v2.2 theme system
```

The legacy flat-string form `collections = ["..."]` is still accepted at load time for backward compatibility; saving (e.g. via the import modal or `gs → c → s`) rewrites the file using the table form above. **Note:** comments in your hand-edited `config.toml` are not preserved across writes.

### Managing collections from the TUI

Press `I` (uppercase) at any time to open the collection-management modal. From there you can:

- browse, edit, or remove the configured collections
- add a new collection by entering a path or running plugin **autodetect** (Bruno scans `preferences.json` and the current working directory)
- multi-select discovered candidates before saving

When `brio` starts with no collections registered (no CLI args, no `[[collections]]` entries, and no Bruno preferences) the import modal opens automatically — no need to quit and re-edit the config file.

The format of each collection is auto-detected (Bruno today; Postman/Insomnia targeted as future loaders). When detection is ambiguous, supply an explicit format via the `format` field on a `[[collections]]` entry.

## Credential hooks

Configure hooks in `~/.config/brio/config.toml` (press `gs` inside brio, then `e` to open the file in `$EDITOR`):

```toml
[[hooks]]
name = "aws-token-refresh"

[hooks.trigger]
status = [401, 403]                          # HTTP status codes that fire the hook
body   = "ExpiredToken|InvalidClientTokenId" # optional regex matched on response body
tier   = "danger"                            # optional: "safe" | "caution" | "danger"

[hooks.script]
path = "~/bin/my-refresh.sh"                 # ~ and $ENV vars are expanded
[hooks.script.env]
AWS_DEFAULT_REGION = "eu-west-1"             # extra env vars passed to the script

[hooks.output]
type = "stdout"                              # "stdout" (non-interactive) | "file" (interactive)

[hooks.vars]
ACCESS_KEY    = "aws_access_key_id"          # output key → brio runtime variable
SECRET_KEY    = "aws_secret_access_key"
SESSION_TOKEN = "aws_session_token"
```

**How it works:**
1. A request returns a matching status code (and optionally a matching body)
2. brio runs the hook script — either capturing `KEY=VALUE` stdout (non-interactive) or suspending the TUI so the user can interact (interactive)
3. The returned credentials are injected as runtime variables
4. The original request is retried automatically

Supported output formats for `type = "file"`: `dotenv`, `json`, `yaml`, `bruno-env`.

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
