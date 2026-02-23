# OmniContext

<p align="center">
  <img src="./assets/logo/omnicontext-logo.svg" alt="OmniContext logo" width="760" />
</p>

Local-first context memory and session tooling built with Go + SQLite.

## First 5 Minutes

```bash
go build -o bin/ocx ./cmd/ocx

./bin/ocx init
./bin/ocx ingest auto --dry-run --json
./bin/ocx ingest auto --max-sessions 20 --since 2026-02-01

./bin/ocx context list
./bin/ocx context stats default
./bin/ocx session search --query timeout --limit 20
```

If this works, your local context system is ready.

## Filesystem Guide

Default data directory:

- `~/.ocx`

Use a custom data directory:

```bash
./bin/ocx --data-dir /tmp/ocx init
```

Loop runs write deterministic artifacts under:

- `~/.ocx/lab/runs/<run-id>/...`
- `~/.ocx/evolve/runs/<run-id>/...`

Typical iteration files:

- `iter-001/inbox/goal.md`
- `iter-001/inbox/constraints.md`
- `iter-001/inbox/context-pack.md`
- `iter-001/inbox/session-ref.json`
- `iter-001/outbox/verify.log`
- `iter-001/outbox/inspector.log`
- `iter-001/outbox/inspector.json`
- `iter-001/summary.md`

Run-level files:

- `report.json`
- `report.md`
- `review-checklist.md`

If using `evolve`, you also get PR handoff files:

- `pr/pr-title.txt`
- `pr/pr-body.md`
- `evolve-report.json`
- `evolve-report.md`

## Common Commands

```bash
# local data
./bin/ocx init
./bin/ocx import claude --path ~/.claude/projects
./bin/ocx import codex --path ~/.codex/sessions
./bin/ocx ingest auto --max-sessions 50 --since 2026-02-01

# query context/session
./bin/ocx context list
./bin/ocx context show default
./bin/ocx context stats default
./bin/ocx session list --limit 50
./bin/ocx session show <session-id> --turn-limit 20
./bin/ocx session search --query timeout --limit 50

# export/share
./bin/ocx context export csv default --out ./default-context.csv
./bin/ocx session export csv --out ./sessions.csv --limit 200
./bin/ocx share export default --out ./default.ocxpack
./bin/ocx share import ./default.ocxpack

# loop tooling
./bin/ocx lab init
./bin/ocx lab run --config ./docs/templates/lab-config.example.json
./bin/ocx evolve run --goal "fix parser edge case" --max-iterations 3 --inspector '<command>'

# health
./bin/ocx doctor
./bin/ocx dashboard
```

## Inspector Contract

`outbox/inspector.json` must contain:

- `verdict`: `QUALIFIED` or `NOT_QUALIFIED`
- `reasons`: array of strings
- `patch_hints`: array of strings
- `confidence`: number in `[0,1]`
