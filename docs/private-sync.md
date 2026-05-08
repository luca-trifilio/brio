# Private Sync Setup

This project uses two git remotes to keep the public OSS repo clean while syncing private files (skills, tasks, roadmap, binaries) across machines.

## Remotes

| Remote | URL | Purpose |
|--------|-----|---------|
| `origin` | `git@github-personal:luca-trifilio/brio.git` | Public OSS repo |
| `private` | `git@github-personal:luca-trifilio/brio-private.git` | Private sync across machines |

The `private` remote has a `sync` branch with a relaxed `.gitignore` that includes everything excluded from `origin/main`.

## What lives where

**Public (`origin/main`):**
- All Go source code
- `docs/adr/` — architectural decisions
- `docs/req-collection-config.md`
- `README.md`, `CLAUDE.md`

**Private (`private/sync`) only:**
- `.claude/skills/` — Claude Code skills
- `.claude/tasks/` — task tracking
- `.claude/settings.json`
- `docs/roadmap.md`
- `docs/prd-v2-architecture.md`
- `docs/bubbletea-architecture.md`
- `docs/github-project-management.md`
- compiled `brio` binary

## Daily workflow

```sh
git push private    # sync everything (run often)
git push origin     # publish to OSS (when ready)
```

## New machine setup

```sh
git clone git@github-personal:luca-trifilio/brio-private.git brio
cd brio
git checkout sync
git remote add origin git@github-personal:luca-trifilio/brio.git
git remote -v   # verify both remotes
```

## Adding new private files

Add the file path to `.gitignore` on `main`/feature branches so it never leaks to `origin`. Do NOT add it to the `.gitignore` on the `sync` branch so it gets picked up by `private`.

## Adding new public files

Normal feature-branch workflow, then push to `origin`. Keep `sync` up to date by merging the feature branch into it:

```sh
git checkout sync
git merge feat/my-feature
git push private
```
