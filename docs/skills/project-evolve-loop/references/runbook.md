# Runbook

## Minimal Flow

```bash
# 1) run evolve loop
ocx evolve run --goal "fix parser edge case" --max-iterations 3

# 2) inspect result files from printed run_dir
cat <run_dir>/evolve-report.md
cat <run_dir>/pr/pr-title.txt
cat <run_dir>/pr/pr-body.md

# 3) optional manual draft PR
BRANCH=$(git rev-parse --abbrev-ref HEAD)
gh pr create --draft --title "$(cat <run_dir>/pr/pr-title.txt)" --body-file <run_dir>/pr/pr-body.md --head "$BRANCH"
```

## Guardrails

- Keep goal narrow and testable.
- Never skip verifier command.
- Keep auto-commit scoped to the evolve branch.
