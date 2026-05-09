# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Skills & Docs

Plugin: **`brio`** (luca-marketplace). Skills: dev-workflow, architecture, code-style, testing, release-checklist, verify.  
Private docs (roadmap, PRD, ADRs) live in the Obsidian vault at `60 - Progetti/brio/` — symlinked into `docs/` (gitignored).

## Project Overview

**brio** is a vim-style TUI for API collections. It loads collections via a plugin interface (`CollectionLoader`), executes HTTP requests with variable interpolation and AWS SigV4 signing. Read-only by design — never writes collection files.

See `CONTRIBUTING.md` for build commands, branching, conventional commits, and release pipeline.
