---
name: project-evolve-loop
description: Trigger local self-evolution loop to auto-fix bugs/features with deterministic verification and human PR review handoff.
---

# Project Evolve Loop

## Objective

User-triggered workflow:
1. Trigger this skill.
2. Local loop starts and iterates.
3. Auto-fix bugs / feature adjustments.
4. Stop only when qualified.
5. Human reviews PR artifacts (or draft PR) before merge.

## Canonical Command

```bash
ocx evolve run \
  --goal "<specific bug/feature target>" \
  --max-iterations 3 \
  --verify "go vet ./... && go test ./... && go build ./cmd/ocx" \
  --auto-commit \
  --open-draft-pr=false
```

## PR Handoff

On success, inspect generated artifacts:
- `.../evolve-report.md`
- `.../pr/pr-title.txt`
- `.../pr/pr-body.md`

Human-in-loop review then decides:
- request more loop iterations, or
- open/review PR, or
- reject and rollback.

## References

- `docs/skills/project-evolve-loop/references/runbook.md`
- `docs/skills/project-evolve-loop/references/review-gate.md`
