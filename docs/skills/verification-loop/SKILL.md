---
name: verification-loop
description: Reusable project-level skill for iterative implement-verify-inspect loops using filesystem inbox/outbox artifacts. Use for auto bug fixes, scoped feature updates, and quality-gated refinement until qualified.
---

# Verification Loop Skill

## Bootstrap

This skill assumes repository bootstrap and contribution policy setup are complete.
Use repository `CONTRIBUTING.md` for policy and PR conventions.

## Key Paths

| Item | Path |
|---|---|
| Skill root | `docs/skills/verification-loop/` |
| Commands reference | `docs/skills/verification-loop/references/commands.md` |
| File protocol | `docs/skills/verification-loop/references/file-protocol.md` |
| PR gate | `docs/skills/verification-loop/references/pr-gate.md` |
| Example lab config | `docs/templates/lab-config.example.json` |

## Workflow

1. Define a testable goal and max iterations.
2. Provide deterministic verifier command.
3. Provide inspector command that writes `outbox/inspector.json` with a `QUALIFIED`/`NOT_QUALIFIED` verdict.
4. Run loop and inspect run artifacts.
5. If qualified, prepare PR using AI templates/checklist.

## Canonical Commands

```bash
# initialize lab workspace artifacts
ocx lab init

# run loop with explicit commands
ocx lab run \
  --goal "fix flaky test and keep behavior compatible" \
  --max-iterations 3 \
  --planner 'echo "- inspect\n- patch\n- verify" > "$OCX_LAB_PLAN_FILE"' \
  --implementer 'echo "implement step" > "$OCX_LAB_IMPL_FILE"' \
  --verify 'go test ./...' \
  --inspector 'cat > "$OCX_LAB_INSPECTOR_JSON_FILE" <<'\''JSON'\''
{"verdict":"QUALIFIED","reasons":["all checks passed"],"patch_hints":[],"confidence":0.95}
JSON'
```

## References

- `docs/skills/verification-loop/references/commands.md`
- `docs/skills/verification-loop/references/file-protocol.md`
- `docs/skills/verification-loop/references/pr-gate.md`

## Boundaries

- Do not skip verification.
- Do not claim completion without verifier/inspector evidence.
- Keep loop scope narrow per run.
- For large architecture changes, use issue-first alignment before deep implementation.
