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
  --context-designer "printf '# Context Pack\n- fixed acceptance\n' > \"$OCX_LAB_CONTEXT_PACK_FILE\"" \
  --launcher "echo 'launch agent with context pack' > \"$OCX_LAB_ITER_DIR/outbox/launcher.log\"" \
  --verify "go vet ./... && go test ./... && go build ./cmd/ocx" \
  --inspector "cat > \"$OCX_LAB_INSPECTOR_JSON_FILE\" <<'JSON'
{\"verdict\":\"QUALIFIED\",\"reasons\":[\"all checks passed\"],\"patch_hints\":[],\"confidence\":0.95}
JSON" \
  --auto-commit \
  --open-draft-pr=false
```

`--inspector` is required. Qualification only happens when verification exits `0` and inspector verdict is `QUALIFIED`.

## PR Handoff

On success, inspect generated artifacts:
- `.../evolve-report.md`
- `.../pr/pr-title.txt`
- `.../pr/pr-body.md`
- `.../iter-*/outbox/inspector.json`
- `.../iter-*/inbox/session-ref.json`

Human-in-loop review then decides:
- request more loop iterations, or
- open/review PR, or
- reject and rollback.

## References

- `docs/skills/project-evolve-loop/references/runbook.md`
- `docs/skills/project-evolve-loop/references/review-gate.md`
