# Evolve Session Inspector Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add a Session Inspector role to the evolve loop so verifier + inspector jointly decide qualification and failed iterations produce actionable hints for the next loop.

**Architecture:** Extend the existing `lab` orchestration to run an inspector command after verifier, persist inspector JSON artifacts, and merge results in a deterministic judge decision. Wire flags through `ocx lab run` and `ocx evolve run`, then update docs and tests. Keep all protocol file-based for reproducibility.

**Tech Stack:** Go (`cobra`, stdlib JSON/file I/O), existing `internal/lab` and `internal/evolve` packages, repo docs/skills markdown.

---

### Task 1: Add Inspector Data Contract and Merge Logic

**Files:**
- Modify: `internal/lab/lab.go`
- Test: `internal/lab/lab_test.go`

**Step 1: Write failing tests for inspector merge behavior**

Add tests in `internal/lab/lab_test.go` that expect:
- loop qualifies only when verifier passes and inspector verdict is `QUALIFIED`
- loop fails when inspector verdict is `NOT_QUALIFIED` even if verifier passes
- hints are written to next iteration input on fail

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/lab -v`
Expected: FAIL on missing inspector merge behavior.

**Step 3: Implement minimal inspector schema + parser**

Add in `internal/lab/lab.go`:
- struct for inspector output: `verdict`, `reasons`, `patch_hints`, `confidence`
- helper to load/validate `outbox/inspector.json`
- invalid/missing schema should mark iteration as not qualified with clear reason

**Step 4: Implement merge decision**

Change iteration decision:
- qualified only if verifier exit code == 0 and inspector verdict == `QUALIFIED`
- otherwise not qualified
- if not qualified and hints exist, write `inbox/next-step-hints.md` for next iteration

**Step 5: Re-run tests**

Run: `go test ./internal/lab -v`
Expected: PASS for new inspector tests.

**Step 6: Commit**

```bash
git add internal/lab/lab.go internal/lab/lab_test.go
git commit -m "feat(lab): add inspector merge gate and hint carry-forward"
```

### Task 2: Add Inspector Command Execution and Artifact Paths

**Files:**
- Modify: `internal/lab/lab.go`
- Test: `internal/lab/lab_test.go`

**Step 1: Write failing test for inspector artifact generation**

Add a test asserting iteration includes:
- `outbox/inspector.log`
- `outbox/inspector.json`
- summary includes inspector verdict and exit code

**Step 2: Run targeted failing test**

Run: `go test ./internal/lab -run Inspector -v`
Expected: FAIL due to missing files/fields.

**Step 3: Implement inspector role execution**

In `internal/lab/lab.go`:
- new config field `InspectorCommand`
- run inspector command after verifier
- expose env vars:
  - `OCX_LAB_INSPECTOR_JSON_FILE`
  - `OCX_LAB_INSPECTOR_LOG_FILE`
  - `OCX_LAB_VERIFY_LOG_FILE`
  - `OCX_LAB_SESSION_REF_FILE`
- persist inspector logs and JSON

**Step 4: Re-run tests**

Run: `go test ./internal/lab -v`
Expected: PASS.

**Step 5: Commit**

```bash
git add internal/lab/lab.go internal/lab/lab_test.go
git commit -m "feat(lab): execute inspector role with file-based artifacts"
```

### Task 3: Wire Flags Through CLI (`lab` and `evolve`)

**Files:**
- Modify: `internal/cli/lab.go`
- Modify: `internal/cli/evolve.go`
- Modify: `internal/evolve/evolve.go`
- Test: `internal/cli/root_test.go`
- Test: `internal/evolve/evolve_test.go`

**Step 1: Add failing CLI tests for new flags passthrough**

Add tests to verify new flags are accepted and mapped:
- `--inspector`
- propagate to `lab.Config` via evolve flow

**Step 2: Run failing tests**

Run: `go test ./internal/cli ./internal/evolve -v`
Expected: FAIL on missing flag plumbing.

**Step 3: Implement flag plumbing**

- `internal/cli/lab.go`: add `--inspector`
- `internal/cli/evolve.go`: add `--inspector`
- `internal/evolve/evolve.go`: pass inspector command into `lab.Run`

**Step 4: Re-run tests**

Run: `go test ./internal/cli ./internal/evolve -v`
Expected: PASS.

**Step 5: Commit**

```bash
git add internal/cli/lab.go internal/cli/evolve.go internal/evolve/evolve.go internal/cli/root_test.go internal/evolve/evolve_test.go
git commit -m "feat(cli): add inspector flag support for lab and evolve"
```

### Task 4: Add Session Reference Protocol for Inspector Inputs

**Files:**
- Modify: `internal/lab/lab.go`
- Test: `internal/lab/lab_test.go`

**Step 1: Write failing test for session-ref input artifact**

Expect `inbox/session-ref.json` exists each iteration with:
- workspace
- run id
- iteration
- optional session path placeholder

**Step 2: Run failing test**

Run: `go test ./internal/lab -run SessionRef -v`
Expected: FAIL.

**Step 3: Implement session-ref artifact generation**

In `internal/lab/lab.go`:
- write `inbox/session-ref.json` before inspector runs
- include deterministic fields inspector can consume

**Step 4: Re-run tests**

Run: `go test ./internal/lab -v`
Expected: PASS.

**Step 5: Commit**

```bash
git add internal/lab/lab.go internal/lab/lab_test.go
git commit -m "feat(protocol): add session-ref artifact for inspector role"
```

### Task 5: Docs and Skill References

**Files:**
- Modify: `README.md`
- Modify: `docs/skills/project-evolve-loop/SKILL.md`
- Modify: `docs/skills/project-evolve-loop/references/runbook.md`
- Modify: `docs/skills/project-evolve-loop/references/review-gate.md`
- Modify: `docs/templates/lab-config.example.json`

**Step 1: Add doc examples with inspector role**

Update command examples to include:
- `--inspector`
- expected inspector JSON schema
- explicit human review gate items that include inspector evidence

**Step 2: Validate docs are coherent**

Run: `rg -n "inspector|session-ref|QUALIFIED" README.md docs/skills docs/templates`
Expected: references present and consistent.

**Step 3: Commit**

```bash
git add README.md docs/skills/project-evolve-loop/SKILL.md docs/skills/project-evolve-loop/references/runbook.md docs/skills/project-evolve-loop/references/review-gate.md docs/templates/lab-config.example.json
git commit -m "docs: add inspector role protocol and review gate guidance"
```

### Task 6: End-to-End Verification and CI Safety

**Files:**
- Modify: `internal/lab/lab_test.go`
- Modify: `internal/evolve/evolve_test.go`
- Optional: `.github/workflows/ci.yml` (only if needed for test runtime env)

**Step 1: Add E2E-style tests for inspector path**

Add temp-repo test that validates:
- inspector command writes JSON
- loop marks qualified
- PR handoff artifacts include inspector evidence path

**Step 2: Run full validation**

Run:
```bash
go vet ./...
go test ./...
go build ./cmd/ocx
```
Expected: all pass.

**Step 3: Commit**

```bash
git add internal/lab/lab_test.go internal/evolve/evolve_test.go
git commit -m "test: add inspector-path e2e verification coverage"
```

### Task 7: Final Integration Commit (if needed)

**Files:**
- Any remaining staged files

**Step 1: Ensure clean feature state**

Run: `git status --short`
Expected: no unintended files.

**Step 2: Create final integration commit**

```bash
git add -A
git commit -m "feat(evolve): add session inspector verification role"
```

**Step 3: Push branch and prepare PR**

```bash
git push -u origin <branch>
```

