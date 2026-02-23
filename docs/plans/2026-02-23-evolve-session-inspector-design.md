# Evolve Session Inspector Design

Date: 2026-02-23
Status: Approved design

## Objective

Add a hybrid verification role to the evolve loop:
- strict qualification verdict from an inspector
- optional patch hints for next iteration when not qualified

This enables self-observation experiments where one agent executes and another inspects the produced session artifact.

## Scope

In scope:
- Session Inspector role in evolve loop
- New artifact protocol for inspector input/output
- Judge merge logic: verifier + inspector
- Carry-forward patch hints into next iteration
- Human PR gate integration with inspector evidence

Out of scope:
- Multi-inspector voting
- Remote orchestration platform
- Non-file transport protocols

## Role Model

1. Context Designer
2. Agent Launcher
3. Implementer
4. Verifier
5. Session Inspector (new)
6. Judge

## Data Flow

1. Evolve iteration starts and writes inbox artifacts.
2. Implementer produces code/session outputs.
3. Verifier runs deterministic checks and writes `verify.log`.
4. Session Inspector receives:
   - session file path
   - goal
   - acceptance criteria
   - verifier log path
5. Session Inspector writes `inspector.json` and optional logs.
6. Judge merges verifier + inspector result:
   - qualified only if verifier passes and inspector verdict is `QUALIFIED`
   - otherwise fail and persist patch hints for next iteration.

## File Protocol

Per iteration `iter-N`:

Input:
- `inbox/session-ref.json`
- `inbox/goal.md`
- `inbox/constraints.md`
- `inbox/context-pack.md`

Output:
- `outbox/verify.log`
- `outbox/inspector.log`
- `outbox/inspector.json`
- `inbox/next-step-hints.md` (generated when fail)
- `summary.md`

Suggested `inspector.json` schema:

```json
{
  "verdict": "QUALIFIED",
  "reasons": ["acceptance criteria met"],
  "patch_hints": ["optional next-step hint"],
  "confidence": 0.92
}
```

## Merge Logic

Inputs:
- verifier exit code
- inspector verdict

Decision:
- PASS if verifier exit code == 0 AND inspector verdict == `QUALIFIED`
- FAIL otherwise

Failure behavior:
- Aggregate reasons
- Save patch hints to `inbox/next-step-hints.md`
- Continue until max iterations

## CLI Changes

Add evolve/lab options:
- `--inspector <cmd>`
- `--session-ref <path>` (optional override)
- `--inspector-required` (default true)

Inspector env vars:
- `OCX_LAB_SESSION_REF_FILE`
- `OCX_LAB_VERIFY_LOG_FILE`
- `OCX_LAB_INSPECTOR_JSON_FILE`
- `OCX_LAB_INSPECTOR_LOG_FILE`

## Human-in-Loop PR Gate

Before PR ready:
- verifier passed in final iteration
- inspector verdict qualified in final iteration
- PR body includes inspector summary and failure->hint history
- evolve report artifacts linked

## Testing Plan

Unit tests:
- parse/validate `inspector.json`
- decision merge logic
- hints carry-forward behavior

Integration tests:
- inspector returns fail then pass across iterations
- verify artifact presence and iteration transitions

E2E smoke:
- temp git repo
- real evolve run with inspector command
- ensure final PR artifacts include inspector evidence

## Risks and Mitigations

Risk: inspector command output drift
- Mitigation: strict schema validation with clear failure reason

Risk: false negatives block progress
- Mitigation: optional manual override flag (future) and transparent logs

Risk: noisy hints
- Mitigation: cap hints and dedupe by hash per iteration

## Rollout

1. Add protocol + flags + merge logic
2. Add tests and sample templates
3. Update docs and project skill references
4. Validate with sub-agent E2E run
