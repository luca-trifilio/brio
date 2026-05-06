# bruno-tui

A vim-style TUI for [Bruno API client](https://www.usebruno.com/) collections, written in Go (Bubble Tea).

Reads `.bru` files directly from disk, executes HTTP requests with full variable interpolation, AWS SigV4 signing, environment switching, and history. MVP is execute-only — `.bru` files are read but never written.

## Install

```sh
brew install luca-trifilio/bruno-tui/bruno-tui
```

## Usage

```sh
bruno-tui /path/to/bruno-collection [/path/to/another-collection ...]
```

## Status

Early development. See `.claude/tasks/bruno-tui-rust/plan.md` for the implementation roadmap.
