# Verification Loop Skill (Generic Project-Level Design)

This document defines a reusable project-level skill for AI agents to perform iterative implementation with a verification and judge loop.

## Purpose

Use this when an agent should:
- auto-fix bugs
- implement a scoped feature
- keep polishing until a judge says `QUALIFIED`

## Core Contract

1. **Implementer** proposes/edits code.
2. **Verifier** runs deterministic checks (tests/lint/build).
3. **Judge** decides `QUALIFIED` or `NOT_QUALIFIED` from artifacts.
4. Repeat until qualified or max iterations reached.

## Filesystem Protocol

For each run:

- `run-dir/<run-id>/iter-001/inbox/goal.md`
- `run-dir/<run-id>/iter-001/inbox/constraints.md`
- `run-dir/<run-id>/iter-001/outbox/planner.log`
- `run-dir/<run-id>/iter-001/outbox/implementer.log`
- `run-dir/<run-id>/iter-001/outbox/verify.log`
- `run-dir/<run-id>/iter-001/outbox/judge.log`
- `run-dir/<run-id>/iter-001/summary.md`
- `run-dir/<run-id>/report.json`
- `run-dir/<run-id>/report.md`

Environment variables available to agent commands:
- `OCX_LAB_RUN_DIR`
- `OCX_LAB_ITER_DIR`
- `OCX_LAB_WORKSPACE`
- `OCX_LAB_GOAL_FILE`
- `OCX_LAB_PLAN_FILE`
- `OCX_LAB_IMPL_FILE`
- `OCX_LAB_JUDGE_FILE`

## Checklist

- [ ] Goal is explicit and testable
- [ ] Verification command is deterministic
- [ ] Judge criteria is explicit (`QUALIFIED` token)
- [ ] Max iterations is set
- [ ] Run artifacts are preserved on disk
- [ ] Final report includes reason and iteration history

## Recommended Defaults

- `max_iterations = 3`
- verifier command: project CI core checks
- judge command: fail if verifier log contains failures

## OmniContext Mapping

- Initialize: `ocx lab init`
- Execute: `ocx lab run --config sample-config.json`
- JSON mode: `ocx lab run ... --json`

## Notes for PR Authors

When opening PRs from this flow, attach:
- report path
- verifier output summary
- explicit risk and rollback notes
