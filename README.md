# OmniContext

Local-first OSS MVP inspired by OneContext, built with Go + SQLite + Bubble Tea.

## Features (MVP)
- `ocx init` to initialize local store
- `ocx import claude --path ...` (real Claude JSONL adapter)
- `ocx import codex --path ...` (real Codex JSONL adapter)
- `ocx ingest auto` (scan defaults: `~/.claude/projects`, `~/.codex/sessions`)
- `ocx ingest auto --dry-run` (preview only)
- `ocx ingest auto --json` (machine-readable output)
- `ocx ingest auto --max-sessions N --since YYYY-MM-DD` (incremental control)
- `ocx context list`
- `ocx context show <id>`
- `ocx context stats <id>` (deterministic aggregation by source/workspace/turns)
- `ocx context export csv <id> --out ./context.csv`
- `ocx session list --limit 50`
- `ocx session show <session-id> --turn-limit 10`
- `ocx session search --query timeout --limit 50`
- `ocx session export csv --out ./sessions.csv --query timeout --limit 1000`
- `ocx lab init` + `ocx lab run` (multi-agent verification loop with file-based inbox/outbox)
- `ocx evolve run` (self-evolution harness: local loop -> auto-fix -> PR handoff)
- `ocx share export <context-id> --out ./x.ocxpack`
- `ocx share import ./x.ocxpack`
- `ocx doctor`
- `ocx dashboard` (minimal Bubble Tea UI)

## Build
```bash
go mod tidy
go build -o bin/ocx ./cmd/ocx
go test ./...
```

## Quickstart
```bash
./bin/ocx init
./bin/ocx import claude --path ~/.claude/projects
./bin/ocx ingest auto
./bin/ocx ingest auto --dry-run --json --max-sessions 20 --since 2026-02-01
./bin/ocx context list
./bin/ocx context stats default
./bin/ocx session list --limit 20
./bin/ocx session search --query codex --limit 20
./bin/ocx session export csv --out ./sessions.csv --limit 100
./bin/ocx context export csv default --out ./default-context.csv
./bin/ocx lab init
./bin/ocx lab run --config ./docs/templates/lab-config.example.json
./bin/ocx evolve run --goal \"fix parser edge case\" --max-iterations 3
./bin/ocx share export default --out ./default.ocxpack
./bin/ocx share import ./default.ocxpack
./bin/ocx doctor
```

## AI Feedback Loop (Harness Engineering)

Expected workflow:
1. Trigger project skill: `docs/skills/project-evolve-loop/SKILL.md`
2. Run local loop:
```bash
./bin/ocx evolve run \
  --goal "fix bug X with backward compatibility" \
  --max-iterations 3 \
  --verify "go vet ./... && go test ./... && go build ./cmd/ocx" \
  --auto-commit
```
3. Inspect artifacts generated under `<data-dir>/evolve/runs/<run-id>/`
4. Human reviews `pr-title.txt` and `pr-body.md`, then opens/reviews PR.

## Data Dir
Default: `~/.ocx`

Use custom path:
```bash
./bin/ocx --data-dir /tmp/ocx init
```

## Contributing

- Contributor guide: `CONTRIBUTING.md`
- AI PR templates:
  - `docs/templates/ai-pr-template.md`
  - `docs/templates/ai-pr-checklist.md`
  - `docs/templates/issue-first-proposal.md`
- Verification loop skill template:
  - `docs/skills/verification-loop/SKILL.md`
  - `docs/skills/verification-loop/references/commands.md`
  - `docs/skills/verification-loop/references/file-protocol.md`
  - `docs/skills/verification-loop/references/pr-gate.md`
  - `docs/skills/verification-loop-generic.md` (high-level design)
  - `docs/templates/lab-config.example.json`
- Project evolve loop skill:
  - `docs/skills/project-evolve-loop/SKILL.md`
  - `docs/skills/project-evolve-loop/references/runbook.md`
  - `docs/skills/project-evolve-loop/references/review-gate.md`

## Release Notes

- Template: `docs/templates/release-notes-template.md`
