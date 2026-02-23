# Runbook

## Minimal Flow

```bash
# 1) run evolve loop
ocx evolve run --goal "fix parser edge case" --max-iterations 3 --inspector 'cat > "$OCX_LAB_INSPECTOR_JSON_FILE" <<'\''JSON'\''
{"verdict":"QUALIFIED","reasons":["all checks passed"],"patch_hints":[],"confidence":0.95}
JSON'

# 2) inspect result files from printed run_dir
cat <run_dir>/evolve-report.md
cat <run_dir>/pr/pr-title.txt
cat <run_dir>/pr/pr-body.md
cat <run_dir>/iter-001/outbox/inspector.json
cat <run_dir>/iter-001/inbox/session-ref.json

# 3) optional manual draft PR
BRANCH=$(git rev-parse --abbrev-ref HEAD)
gh pr create --draft --title "$(cat <run_dir>/pr/pr-title.txt)" --body-file <run_dir>/pr/pr-body.md --head "$BRANCH"
```

## Inspector Protocol

Inspector command is required and must write JSON to `$OCX_LAB_INSPECTOR_JSON_FILE`.

Schema:

```json
{
  "verdict": "QUALIFIED",
  "reasons": ["all checks passed"],
  "patch_hints": [],
  "confidence": 0.95
}
```

Rules:
- `verdict` must be `QUALIFIED` or `NOT_QUALIFIED`
- `reasons` is required and must be an array
- `patch_hints` is required and must be an array
- `confidence` is required and must be a number between `0` and `1`

Inspector input artifact:
- `inbox/session-ref.json` is generated each iteration and exposed via `$OCX_LAB_SESSION_REF_FILE`

## Guardrails

- Keep goal narrow and testable.
- Never skip verifier command.
- Never skip inspector command (`--inspector` is required).
- Keep auto-commit scoped to the evolve branch.
