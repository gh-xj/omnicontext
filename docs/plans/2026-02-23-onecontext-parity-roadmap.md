# OneContext Parity Roadmap (Open-Source)

Date: 2026-02-23
Scope: close the highest-value functional gaps between OneContext workflow behavior and OmniContext while preserving local-first simplicity.

## Source Baseline

Reference experiment report:
- `/Users/xiangjun/Downloads/onecontext-full-workflow-20260222-152012/report.md`

Observed OneContext capabilities to mirror:
- doctor/upgrade and auto-repair workflows
- Claude/Codex hook integration + compatibility checks
- watcher/backlog catch-up behavior
- richer operational telemetry (`jobs/events`)
- dashboard-led ingestion modes

## Product Principle

Do not clone internal complexity.
Build an open, auditable, file-first system where each automation has:
1. deterministic inputs
2. deterministic artifacts
3. human-readable review entrypoint

## Phase 1: Integration Reliability (High ROI, Low Risk)

Goal:
- make OmniContext self-healing for local integrations and onboarding

Deliverables:
1. `ocx doctor --fix`
- validates environment, db schema, path permissions, CLI dependencies
- checks known Claude/Codex config failure modes
- applies safe auto-fixes with backup files

2. `ocx hooks install|status|repair`
- manages Claude/Codex hook entries in user config
- writes idempotent markers and supports rollback
- emits machine-readable status (`--json`)

3. compatibility matrix
- codex/claude version detection + warning rules
- explicit guidance when a hook method is unsupported

Success criteria:
- first-time setup to healthy state in under 2 minutes
- repeated `doctor --fix` is idempotent
- no destructive config edits without backup

## Phase 2: Continuous Capture + Operational Telemetry

Goal:
- move from manual ingest to robust continuous capture with observability

Deliverables:
1. `ocx watcher` service mode
- scans/increments sessions from configured sources
- supports `dashboard_only`-style and `always_on` modes
- catch-up limit config (`max_catchup_sessions` equivalent)

2. operational tables + APIs
- add minimal telemetry entities: `jobs`, `events`, `event_sessions`
- expose `ocx jobs list|show` and `ocx events tail`

3. failure recovery
- retry/backoff policies for parse/import failures
- dead-letter queue folder for malformed inputs

Success criteria:
- watcher recovers from restart without data loss
- ingestion lag visible via CLI (`ocx jobs`)
- parse failures are inspectable and replayable

## Phase 3: Collaboration + Share Workflow

Goal:
- add optional collaboration flow without compromising local-first default

Deliverables:
1. hardened share flow
- signed/encrypted `.ocxpack` option
- secret-redaction policy profiles before export

2. optional cloud backend adapter (plugin boundary)
- token-based login/logout
- remote publish/pull for shared context packs
- keep local-only mode fully supported

3. review workflow UX
- one-command PR handoff bundle from evolve runs
- reviewer packet includes diff summary + verification evidence pointers

Success criteria:
- secure sharing available but optional
- local-only users see zero cloud dependency
- PR handoff remains auditable end-to-end

## Cross-Phase Engineering Guardrails

1. every automation writes evidence artifacts under run directories
2. every state mutation is reversible or backed up
3. every command has `--json` for machine workflows
4. avoid hidden daemons: explicit start/stop/status controls
5. maintain low cognitive load: one canonical human review file per run

## Suggested Execution Order (90-Day)

- Weeks 1-3: Phase 1 (`doctor --fix`, hooks manager, compatibility checks)
- Weeks 4-8: Phase 2 (watcher MVP, jobs/events visibility, replay path)
- Weeks 9-12: Phase 3 (share hardening, optional cloud adapter boundary)

## Immediate Next Tickets

1. Implement `ocx hooks status` with dry-run diff of config edits.
2. Implement `ocx doctor --fix` with backup + restore manifest.
3. Design watcher config schema (`mode`, sources, catch-up limit).
4. Add `jobs/events` migration and read-only CLI listing.
5. Add end-to-end harness test for setup -> ingest -> verify -> report.
