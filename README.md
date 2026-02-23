# OmniContext

Local-first OSS MVP inspired by OneContext, built with Go + SQLite + Bubble Tea.

## Features (MVP)
- `ocx init` to initialize local store
- `ocx import claude --path ...` (real Claude JSONL adapter)
- `ocx import codex --path ...` (real Codex JSONL adapter)
- `ocx ingest auto` (scan defaults: `~/.claude/projects`, `~/.codex/sessions`)
- `ocx context list`
- `ocx context show <id>`
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
./bin/ocx context list
./bin/ocx share export default --out ./default.ocxpack
./bin/ocx share import ./default.ocxpack
./bin/ocx doctor
```

## Data Dir
Default: `~/.ocx`

Use custom path:
```bash
./bin/ocx --data-dir /tmp/ocx init
```
