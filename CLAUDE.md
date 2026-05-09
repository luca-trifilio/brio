# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Skills & Docs

Plugin: **`brio`** (luca-marketplace). Skills: dev-workflow, architecture, release-checklist, verify.  
Private docs (roadmap, PRD, ADRs) live in the Obsidian vault at `60 - Progetti/brio/` — symlinked into `docs/` (gitignored).

## Project Overview

**brio** is a vim-style TUI for API collections. It loads collections via a plugin interface (`CollectionLoader`), executes HTTP requests with variable interpolation and AWS SigV4 signing. **brio is read-only by design — it never writes collection files.**

## Build & Test Commands

```sh
make setup      # Install go tools (goimports, golangci-lint) + lefthook hooks (goreleaser/cosign/gh via brew)
make build      # Compile binary → ./brio
make run ARGS="<args>"  # Build and run with arguments
make check      # Full QA: fmt + vet + lint + test (run before committing)
make test       # Tests with race detector + coverage
make test-short # Fast iteration tests (skips slow tests)
make lint       # golangci-lint
make fmt        # gofmt + goimports
make snapshot   # Build all platforms locally (no publish)
```

Run `make check` before every commit. The pre-commit hook (lefthook) also runs fmt + lint automatically.

## Code Style

- **goimports** with local prefix `github.com/luca-trifilio/brio` — internal imports grouped separately
- **godot**: exported top-level comments must end with a period
- Test files get relaxed rules (errcheck, noctx, godot are excluded)
- Run `make fmt` to apply gofmt + goimports in one step

## Testing

```sh
make test         # -race -coverprofile=coverage.txt -covermode=atomic
make test-short   # -short flag, no race detector
make cover        # Open coverage HTML in browser
```

Use table-driven tests. The `-short` flag skips slow/integration tests — guard slow tests with `testing.Short()`.

## Branch & PR Conventions

**Branch naming:** `type/short-description` mirroring Conventional Commit types  
Examples: `feat/history-search`, `fix/parser-bug`, `chore/update-deps`

**PR workflow:**
- Currently solo; designed for community contributions
- PR title is the commit message on main (squash-merge)
- PR title must follow Conventional Commits (validated by `semantic-pr.yml`)
- `release-please` reads commit history to auto-bump version and generate changelog

## Conventional Commits (Critical)

Release automation depends on this. The PR title becomes the commit on main.

```
<type>[optional scope]: <short description>
```

| Type | Semver effect |
|------|--------------|
| `feat` | minor bump |
| `fix` | patch bump |
| `feat!` / `BREAKING CHANGE:` | major bump |
| `perf`, `refactor`, `docs`, `test`, `chore`, `ci` | no release |

Rules:
- Lowercase subject: `feat: add X` ✓ `feat: Add X` ✗
- No trailing period
- Imperative mood: "add", "fix", "remove"
- Max 50 chars subject line

## Architecture

- **TUI:** Bubble Tea (Elm-inspired state machine). Multi-pane: tree, request, response, env, vars, history, help, settings. Vim-style keybindings (normal/insert/command modes).
- **Plugin loader:** `CollectionLoader` interface in `internal/plugins`. Bruno loader wraps the existing parser; Postman loader in progress. `AutodetectLoader` opt-in for scanning known config paths.
- **Canonical model:** `internal/canonical` — format-agnostic `Collection`, `Folder`, `Request`, `Scripts`, `AuthBlock`, `Diagnostic`. All loaders map to this model.
- **Variable interpolation:** Layered scoping — collection → environment → folder chain → request → runtime overrides. Supports cycle detection.
- **AWS SigV4:** Full inheritance chain (request → folder → collection).
- **Credential hooks:** Trigger-based refresh (status code / regex body / env tier). Supports dotenv, json, yaml, bruno-env output formats.

## Gotchas

- **Read-only by design:** post-response scripts can inject runtime vars but never write `.bru` files
- **CGO_ENABLED=0:** Static binaries — no C deps
- **Squash-merge:** PR title = commit on main. Don't manually bump versions; release-please is automatic
- **Lefthook hooks:** pre-commit runs fmt+lint, pre-push runs full tests. Install with `make setup`
- **Keyless cosign signing:** uses GitHub OIDC tokens (no long-lived keys)

## Release Pipeline

1. Merge PR with Conventional Commit title → main
2. `release-please` opens Release PR (bumps CHANGELOG, version)
3. Merge Release PR → git tag created automatically
4. `release.yml` triggers GoReleaser: multi-platform builds, cosign signing, SBOM, Homebrew formula update

Do **not** push tags manually or edit version numbers — let release-please handle it.
