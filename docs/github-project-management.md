# GitHub Project Management Guide

How brio manages its open source presence on GitHub — labels, triage, templates, and community health.

---

## Label Taxonomy

### Area labels (auto-applied via `.github/labeler.yml`)
| Label | Scope |
|-------|-------|
| `area: parser` | `.bru` / format parsing |
| `area: tui` | Bubble Tea UI layer |
| `area: http` | Request execution, SigV4, auth |
| `area: cli` | CLI entrypoint, flags |
| `area: model` | Canonical collection model |
| `area: interp` | Variable interpolation, scripting |
| `area: config` | `config.toml`, settings |
| `area: history` | History store |
| `area: plugins` | Format integration plugins |
| `area: themes` | Theme plugins |
| `area: ci` | GitHub Actions, release pipeline |
| `area: docs` | Documentation |
| `area: deps` | Dependency updates (Dependabot) |

### Status labels (applied manually during triage)
| Label | Meaning |
|-------|---------|
| `status: needs-triage` | New — not yet reviewed by maintainer |
| `status: investigating` | Maintainer is actively looking into it |
| `status: blocked` | Waiting on an external decision or dependency |
| `status: help-wanted` | Community contribution explicitly invited |
| `good first issue` | Appropriate for first-time contributors |

### Type labels
| Label | Meaning |
|-------|---------|
| `type: bug` | Something is broken |
| `type: feature` | New functionality request |
| `type: docs` | Documentation gap or error |
| `type: question` | Should be redirected to Discussions |
| `type: regression` | Worked in a prior version, now broken |

### Priority labels (applied sparingly)
| Label | Meaning |
|-------|---------|
| `priority: critical` | Data loss, crash, security issue |
| `priority: high` | Blocking common workflows |
| `priority: low` | Nice to have |

---

## Triage Flow

1. New issue opens → auto-labeled `status: needs-triage` (via Actions).
2. Maintainer reviews: apply `type:*` and `area:*` (area is usually auto-applied).
3. If it's a question → close, link to Discussions, remove `needs-triage`.
4. If it's a duplicate → close as duplicate, link to original.
5. If it's actionable → apply `priority:*`, optionally `good first issue` or `status: help-wanted`, remove `needs-triage`.
6. If it needs more info → comment requesting details, apply `status: investigating`.

Target: no issue sits at `status: needs-triage` for more than 7 days.

---

## Issue Templates

### Bug Report (`.github/ISSUE_TEMPLATE/bug.yml`)
Required fields:
- **Description** — what happened vs what was expected
- **Steps to reproduce** — minimal, numbered
- **brio version** (`brio --version` output)
- **OS and terminal emulator** — critical for TUI rendering bugs (iTerm2, Ghostty, Windows Terminal, etc.)
- **Collection format** — Bruno / Postman / Insomnia
- **Additional context** — screen recordings welcome; redact secrets from env file snippets

### Feature Request (`.github/ISSUE_TEMPLATE/feature.yml`)
Fields:
- **Problem / use case** — what workflow is blocked or painful
- **Proposed solution** — optional, but encouraged
- **Alternatives considered** — have you looked at credential hooks? curl export? etc.

### Config
```yaml
# .github/ISSUE_TEMPLATE/config.yml
blank_issues_enabled: false
contact_links:
  - name: Question or discussion
    url: https://github.com/luca-trifilio/brio/discussions
    about: Ask questions and propose ideas in Discussions, not Issues.
```

---

## PR Template (`.github/PULL_REQUEST_TEMPLATE.md`)

```markdown
## Summary
<!-- What does this PR do and why? -->

## Checklist
- [ ] PR title follows Conventional Commits (`feat:`, `fix:`, `chore:`, etc.)
- [ ] `make check` passes locally (fmt + vet + lint + test)
- [ ] Tests added or updated for changed behavior
- [ ] For TUI changes: tested on at least one terminal emulator locally
- [ ] Docs updated if this changes user-facing behavior
```

No CHANGELOG entry needed — release-please generates it from the commit.

---

## Branch Protection (main)

- Require status checks to pass before merge: `lint`, `test`, `build`, `semantic-pr`
- Require at least 1 approving review (relax to 0 for solo-maintainer convenience if needed)
- Dismiss stale reviews when new commits are pushed
- Require branches to be up to date before merging
- Squash merge only — PR title becomes the commit on main
- Auto-delete head branches after merge

---

## Release Pipeline (current — do not change)

1. Merge PR with Conventional Commit title → main
2. `release-please` opens/updates a Release PR (bumps CHANGELOG, version file)
3. Merge Release PR → git tag created automatically
4. `release.yml` triggers GoReleaser: multi-platform builds, cosign signing, SBOM, Homebrew formula update

**Never push tags manually or edit version numbers.** Let release-please handle it.

Semver mapping:
| Commit type | Bump |
|-------------|------|
| `feat` | minor |
| `fix`, `perf` | patch |
| `feat!` / `BREAKING CHANGE:` | major |
| everything else | no release |

---

## Discussions

Use GitHub Discussions as the first-line venue for questions, ideas, and RFCs.

Recommended categories:
- **Q&A** — questions that don't belong in Issues
- **Ideas / RFC** — feature proposals before they become Issues
- **Show and Tell** — community workflows, integrations, themes

When a Discussion matures into a vetted feature request, open an Issue with `type: feature` and link back to the Discussion thread.

---

## Community Health Files

| File | Status | Notes |
|------|--------|-------|
| `CODE_OF_CONDUCT.md` | ✅ | Contributor Covenant v2.1 |
| `CONTRIBUTING.md` | ✅ | Covers build, test, branch naming, Conventional Commits |
| `SECURITY.md` | ✅ | Responsible disclosure, response timelines |
| `MAINTAINERS.md` | ✅ | Expand as plugin/theme contributors join |
| `FUNDING.yml` | ➕ consider | GitHub Sponsors / Ko-fi if seeking community support |

As plugin contributors join, add them to `MAINTAINERS.md` with their area (`plugins/postman`, `themes/dracula`, etc.). This gives clear ownership without requiring repo write access.

---

## Large Feature Contribution Gate

For large contributions (new format plugins, new panes, breaking model changes): require an Issue or Discussion first. The contributor should explain their approach and get maintainer acknowledgement before writing code. This prevents wasted effort on PRs that won't merge.

Add to CONTRIBUTING.md:
> For significant changes (new plugins, new panes, breaking changes), please open an issue or Discussion describing your approach before writing code. This saves everyone time.

---

## What to Skip (not worth the overhead at this scale)

- **GitHub Projects** — overkill for a solo-maintained project
- **Milestones** — only useful when coordinating multi-issue releases; use labels instead
- **CODEOWNERS** — not needed until multiple maintainers own different areas
- **Auto-assignment** — manual assignment is fine at this scale
