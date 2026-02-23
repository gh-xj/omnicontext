# Verification Loop Skill (Generic Project-Level Design)

This document defines a reusable project-level skill for AI agents to perform iterative implementation with a verification and inspector loop.

## Purpose

Use this when an agent should:
- auto-fix bugs
- implement a scoped feature
- keep polishing until verifier + inspector evidence says `QUALIFIED`

## Core Contract

1. **Implementer** proposes/edits code.
2. **Verifier** runs deterministic checks (tests/lint/build).
3. **Inspector** writes `outbox/inspector.json` with `QUALIFIED` or `NOT_QUALIFIED` plus reasons/hints.
4. Repeat until qualified or max iterations reached.

## Filesystem Protocol

For each run:

- `run-dir/<run-id>/iter-001/inbox/goal.md`
- `run-dir/<run-id>/iter-001/inbox/constraints.md`
- `run-dir/<run-id>/iter-001/inbox/context-pack.md`
- `run-dir/<run-id>/iter-001/inbox/session-ref.json`
- `run-dir/<run-id>/iter-001/outbox/planner.log`
- `run-dir/<run-id>/iter-001/outbox/implementer.log`
- `run-dir/<run-id>/iter-001/outbox/verify.log`
- `run-dir/<run-id>/iter-001/outbox/inspector.log`
- `run-dir/<run-id>/iter-001/outbox/inspector.json`
- `run-dir/<run-id>/iter-001/summary.md`
- `run-dir/<run-id>/report.json`
- `run-dir/<run-id>/report.md`
- `run-dir/<run-id>/review-checklist.md`

Environment variables available to agent commands:
- `OCX_LAB_RUN_DIR`
- `OCX_LAB_ITER_DIR`
- `OCX_LAB_WORKSPACE`
- `OCX_LAB_GOAL_FILE`
- `OCX_LAB_CONSTRAINTS_FILE`
- `OCX_LAB_CONTEXT_PACK_FILE`
- `OCX_LAB_CONTEXT_PACK_SHA256`
- `OCX_LAB_PLAN_FILE`
- `OCX_LAB_IMPL_FILE`
- `OCX_LAB_VERIFY_LOG_FILE`
- `OCX_LAB_INSPECTOR_LOG_FILE`
- `OCX_LAB_INSPECTOR_JSON_FILE`
- `OCX_LAB_SESSION_REF_FILE`
- `OCX_LAB_SESSION_PATH_FILE`

## Checklist

- [ ] Goal is explicit and testable
- [ ] Verification command is deterministic
- [ ] Inspector JSON contract is explicit (`verdict/reasons/patch_hints/confidence`)
- [ ] Max iterations is set
- [ ] Run artifacts are preserved on disk
- [ ] Final report includes reason and iteration history

## Recommended Defaults

- `max_iterations = 3`
- verifier command: project CI core checks
- inspector command: emit JSON contract to `$OCX_LAB_INSPECTOR_JSON_FILE`

## OmniContext Mapping

- Initialize: `ocx lab init`
- Execute: `ocx lab run --config sample-config.json`
- JSON mode: `ocx lab run ... --json`

## Notes for PR Authors

When opening PRs from this flow, attach:
- report path
- verifier output summary
- explicit risk and rollback notes
