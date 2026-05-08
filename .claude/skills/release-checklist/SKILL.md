---
name: release-checklist
description: Guide through the brio release process — verify commit messages, check release-please status, confirm GoReleaser artifacts, and validate Homebrew formula update.
disable-model-invocation: true
---

Walk through the brio release checklist in order:

## 1. Verify commits since last release

Run:
```sh
git log $(git describe --tags --abbrev=0)..HEAD --oneline
```

Check each commit title follows Conventional Commits format. Flag any that:
- Don't have a valid type prefix (`feat:`, `fix:`, `chore:`, etc.)
- Have uppercase subject or trailing period
- Are ambiguous about semver impact

## 2. Check release-please PR status

Run:
```sh
gh pr list --label "autorelease: pending" --repo luca-trifilio/brio
```

If a Release PR is open: show its title and the version it will bump to. Ask the user to confirm the semver bump is correct before merging.

If no Release PR exists: explain that release-please only opens one after a `feat` or `fix` commit lands on main. If the user expected a release, check if the commits use non-releasing types (chore, docs, etc.).

## 3. Post-merge verification (run after Release PR is merged)

Check that the tag was created:
```sh
git fetch --tags && git tag --sort=-v:refname | head -5
```

Check that the release workflow triggered:
```sh
gh run list --workflow=release.yml --repo luca-trifilio/brio --limit 3
```

## 4. Confirm release artifacts

Once the release workflow completes:
```sh
gh release view --repo luca-trifilio/brio
```

Verify:
- All 4 platform binaries present (darwin/amd64, darwin/arm64, linux/amd64, linux/arm64)
- `checksums.txt` and `checksums.txt.sig` present (cosign signing)
- SBOM file present (`brio_<version>_sbom.json`)

## 5. Homebrew formula update

Check the tap was updated:
```sh
gh api repos/luca-trifilio/homebrew-tap/commits?per_page=3 --jq '.[].commit.message'
```

Confirm the latest commit updates the brio formula to the new version.

---

If any step fails, report what's missing and the likely cause. The full release pipeline is automated — manual intervention should only be needed for CI failures or cosign signing issues.
