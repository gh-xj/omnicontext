# OmniContext

<p align="center">
  <img src="./assets/logo/omnicontext-logo.svg" alt="OmniContext logo" width="760" />
</p>

Local-first context memory and session tooling built with Go + SQLite.

## Install

### Go install (recommended)

```bash
go install github.com/gh-xj/omnicontext/cmd/ocx@latest
```

### Homebrew (tap)

```bash
brew tap gh-xj/homebrew-tap
brew install ocx
```

### From source (this repo)

```bash
go build -o bin/ocx ./cmd/ocx
```

## Install Verification

```bash
which ocx
ocx version
ocx --help
```

Expected:

- `which ocx` prints a valid binary path
- `ocx version` prints release/build version
- `ocx --help` prints command usage

## First 60 Seconds

```bash
ocx init
ocx ingest auto --max-sessions 20 --since 2026-02-01
ocx context stats default
```

If this works, your local context system is usable.

## Filesystem Guide

Default data directory:

- `~/.ocx`

Use a custom data directory:

```bash
ocx --data-dir /tmp/ocx init
```

Run artifacts are deterministic on disk:

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

For evolve runs:

- `pr/pr-title.txt`
- `pr/pr-body.md`
- `evolve-report.json`
- `evolve-report.md`

## Common Commands

```bash
# local data
ocx init
ocx import claude --path ~/.claude/projects
ocx import codex --path ~/.codex/sessions
ocx ingest auto --dry-run --json

# query context/session
ocx context list
ocx context stats default
ocx session list --limit 50
ocx session search --query timeout --limit 50

# export/share
ocx context export csv default --out ./default-context.csv
ocx session export csv --out ./sessions.csv --limit 200
ocx share export default --out ./default.ocxpack
ocx share import ./default.ocxpack

# loop tooling
ocx lab init
ocx lab run --config ./docs/templates/lab-config.example.json
ocx evolve run --goal "fix parser edge case" --max-iterations 3 --inspector 'cat > "$OCX_LAB_INSPECTOR_JSON_FILE" <<'\''JSON'\''
{"verdict":"QUALIFIED","reasons":["all checks passed"],"patch_hints":[],"confidence":0.95}
JSON'

# health
ocx doctor
ocx dashboard
```

## Inspector Contract

`outbox/inspector.json` must contain:

- `verdict`: `QUALIFIED` or `NOT_QUALIFIED`
- `reasons`: array of strings
- `patch_hints`: array of strings
- `confidence`: number in `[0,1]`
