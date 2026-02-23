# Human Review Gate

Before merge:

- [ ] Verify loop reached qualified state.
- [ ] Read verifier output in run artifacts.
- [ ] Read inspector evidence (`outbox/inspector.json` and `outbox/inspector.log`) and confirm verdict is `QUALIFIED`.
- [ ] Confirm inspector schema fields exist: `verdict`, `reasons`, `patch_hints`, `confidence`.
- [ ] Confirm inspector consumed the iteration reference (`inbox/session-ref.json`).
- [ ] Confirm file diff matches intended scope.
- [ ] Confirm PR body includes risk/rollback notes.
- [ ] Re-run local checks if high risk.

Recommended commands:

```bash
go vet ./...
go test ./...
go build ./cmd/ocx
cat <run_dir>/iter-001/outbox/inspector.json
cat <run_dir>/iter-001/inbox/session-ref.json
git show --stat
gh pr checks <n>
```
