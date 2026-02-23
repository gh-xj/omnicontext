# OmniContext

Local-first context memory + verification-loop tooling for human and AI collaboration.

## Why

OmniContext is designed around one principle: keep all evidence on disk so humans can verify fast.

- ingest session data from local tools (Claude/Codex)
- query/share context from a local SQLite store
- run a local verification loop (`lab` / `evolve`)
- hand off a PR with reproducible artifacts for human review

## First 5 Minutes (User-First)

```bash
go build -o bin/ocx ./cmd/ocx

./bin/ocx init
./bin/ocx ingest auto --dry-run --json
./bin/ocx ingest auto --max-sessions 20 --since 2026-02-01

./bin/ocx context list
./bin/ocx context stats default
./bin/ocx session search --query timeout --limit 20
```

If this works, you already have a usable local context system.

## Filesystem Mental Model

### 1) Data root

Default data dir is `~/.ocx` (override with `--data-dir`).

```bash
./bin/ocx --data-dir /tmp/ocx init
```

### 2) Loop artifacts

`ocx lab run` and `ocx evolve run` write run artifacts under:

- `~/.ocx/lab/runs/<run-id>/...`
- `~/.ocx/evolve/runs/<run-id>/...`

Each iteration has deterministic structure:

- `iter-001/inbox/goal.md`
- `iter-001/inbox/constraints.md`
- `iter-001/inbox/context-pack.md`
- `iter-001/inbox/session-ref.json`
- `iter-001/outbox/verify.log`
- `iter-001/outbox/inspector.log`
- `iter-001/outbox/inspector.json`
- `iter-001/summary.md`

Run-level review entrypoints:

- `report.json`
- `report.md`
- `review-checklist.md` (start here for human review)

### 3) PR handoff artifacts (`evolve`)

When qualified, `evolve` writes:

- `pr/pr-title.txt`
- `pr/pr-body.md`
- `evolve-report.json`
- `evolve-report.md`

## Command Surface

### Context and sessions

- `ocx init`
- `ocx import claude --path <dir>`
- `ocx import codex --path <dir>`
- `ocx ingest auto [--dry-run] [--json] [--max-sessions N] [--since YYYY-MM-DD]`
- `ocx context list|show|stats|export csv`
- `ocx session list|show|search|export csv`
- `ocx share export|import`
- `ocx doctor`
- `ocx dashboard`

### Verification loop

- `ocx lab init`
- `ocx lab run --config docs/templates/lab-config.example.json`
- `ocx evolve run --goal "fix parser edge case" --max-iterations 3 --inspector '<cmd>'`

## Lean Human Review Flow

1. Open `<run-dir>/review-checklist.md`.
2. Validate last iteration `summary.md`.
3. Check `outbox/verify.log` and `outbox/inspector.json`.
4. If evolve run: review `pr/pr-title.txt` and `pr/pr-body.md`.
5. Open PR only after evidence is coherent.

Inspector output contract (`outbox/inspector.json`):

- `verdict`: `QUALIFIED` or `NOT_QUALIFIED`
- `reasons`: `[]string`
- `patch_hints`: `[]string`
- `confidence`: number in `[0,1]`

## Contributor Onboarding

```bash
go mod tidy
go test ./...
go vet ./...
go build ./cmd/ocx
```

Primary docs:

- `CONTRIBUTING.md`
- `docs/skills/project-evolve-loop/SKILL.md`
- `docs/skills/project-evolve-loop/references/runbook.md`
- `docs/skills/project-evolve-loop/references/review-gate.md`
- `docs/templates/lab-config.example.json`

PR templates:

- `docs/templates/ai-pr-template.md`
- `docs/templates/ai-pr-checklist.md`
- `docs/templates/issue-first-proposal.md`

## Release

- release notes template: `docs/templates/release-notes-template.md`
