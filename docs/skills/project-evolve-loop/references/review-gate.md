# Human Review Gate

Before merge:

- [ ] Verify loop reached qualified state.
- [ ] Read verifier output in run artifacts.
- [ ] Confirm file diff matches intended scope.
- [ ] Confirm PR body includes risk/rollback notes.
- [ ] Re-run local checks if high risk.

Recommended commands:

```bash
go vet ./...
go test ./...
go build ./cmd/ocx
git show --stat
gh pr checks <n>
```
