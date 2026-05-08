---
name: verify
description: Run full QA check (make check) and report results. Use after making changes to validate code quality before committing.
---

Run `make check` in the project root. This executes fmt + vet + lint + test in sequence.

Report each stage outcome:
- If all pass: confirm everything is clean and the code is ready to commit.
- If any stage fails: show the exact error output, identify which files/lines are affected, and fix the issues before re-running.

After a clean run, remind the user to check if the change warrants a Conventional Commit message of type `feat`, `fix`, or another type — since this determines whether release-please will trigger a version bump.
