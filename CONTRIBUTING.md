# Contributing to brio

Thank you for your interest in contributing! This document explains how the
project works, how to set up a dev environment, and — most importantly — **how
commits must be written**, because the entire release pipeline is automated
from them.

---

## Code of Conduct

By participating you agree to the [Code of Conduct](CODE_OF_CONDUCT.md).

---

## Prerequisites

| Tool | Minimum version | Install |
|---|---|---|
| Go | see `go.mod` | https://go.dev/dl |
| `make` | any | system |
| `golangci-lint` | latest | `make setup` |
| `lefthook` | latest | `brew install lefthook` |
| `goreleaser` | v2 | `brew install goreleaser` |
| `gh` | latest | `brew install gh` |

Run `make setup` to install tools and wire up git hooks (pre-commit: fmt/vet/lint, pre-push: tests).

---

## Development workflow

```sh
git clone https://github.com/luca-trifilio/brio.git
cd brio
make build      # compile
make test       # run tests with race detector
make check      # fmt + vet + lint + test — mirrors CI
make snapshot   # build all platforms locally (no publish)
```

---

## Conventional Commits — read this carefully

**The entire release pipeline is automated from commit messages.**
[release-please](https://github.com/googleapis/release-please) reads every
commit that lands on `main` and uses it to decide:

- whether a release is needed
- what the new version number should be (semver bump)
- what goes in the CHANGELOG

That only works if commits follow the
[Conventional Commits](https://www.conventionalcommits.org/) spec.

### Format

```
<type>[optional scope]: <short description>

[optional body]

[optional footer(s)]
```

### Types and their effect on versioning

| Type | Semver bump | Example |
|---|---|---|
| `feat` | **minor** | `feat: add history search` |
| `fix` | patch | `fix: correct env variable scoping` |
| `feat!` or `BREAKING CHANGE:` footer | **major** | `feat!: remove --legacy flag` |
| `perf` | patch | `perf: cache parsed .bru files` |
| `refactor` | — (no release) | `refactor: extract auth logic` |
| `docs` | — | `docs: update keybindings table` |
| `test` | — | `test: add parser corpus cases` |
| `chore` | — | `chore: update dependencies` |
| `ci` | — | `ci: pin actions versions` |

### Rules

- **Subject starts lowercase** — `feat: add X` ✓ · `feat: Add X` ✗
- **No period at the end** — `fix: correct path` ✓ · `fix: correct path.` ✗
- **Imperative mood** — "add", "fix", "remove", not "added", "fixed"
- **50 chars max** for the subject line

### Where the check runs

The **PR title** is checked automatically (see `semantic-pr.yml`). Because we
use **squash merges**, the PR title becomes the commit message that lands on
`main` — that is the commit release-please reads.

---

## Branching

- Branch from `main`: `git checkout -b feat/your-feature`
- Keep branches short-lived
- Rebase on `main` before opening a PR: `git rebase origin/main`

---

## Pull Requests

1. Open a PR against `main`
2. Give the PR a conventional commit title — that title **is** the commit
3. Fill in the PR template
4. All CI checks must pass (lint, test, build matrix)
5. At least one approval is required once the project grows past solo stage
6. Use **squash merge** — the PR title becomes the commit on main

---

## How releases work

You don't need to do anything to release. Here is what happens automatically:

```
your PR merged to main (conventional commit title)
        ↓
release-please analyses commits since the last release
        ↓
opens / updates a "Release PR" — bumps CHANGELOG.md
        ↓
maintainer reviews and merges the Release PR
        ↓
release-please pushes the git tag (e.g. v0.2.0)
        ↓
release.yml triggers → GoReleaser builds all platforms,
signs the checksum with cosign, generates an SBOM,
publishes the GitHub Release, and updates the Homebrew formula
```

**As a contributor**: write good conventional commits. That's it.
Changelogs, version numbers, and release artefacts are fully automated.

---

## Questions?

Open a [Discussion](https://github.com/luca-trifilio/brio/discussions) rather
than an issue. Issues are reserved for confirmed bugs and accepted feature
requests.
