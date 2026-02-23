# Contributing to OmniContext

Thanks for contributing.

## Policy

- External PRs are welcome.
- For large behavior or architecture changes, use issue-first:
  - open an issue with scope, rationale, risk, and test plan
  - wait for maintainer alignment before large implementation

## PR Convention

- Keep PR scope narrow and reviewable.
- Include a concise summary of what/why/how.
- Include exact verification commands and outputs.
- Note compatibility impact and rollback/mitigation for risky changes.

## Required Local Checks

```bash
go vet ./...
go test ./...
go build ./cmd/ocx
```

## AI Agent Guidance

Use templates in `docs/templates/`:
- `docs/templates/ai-pr-template.md`
- `docs/templates/ai-pr-checklist.md`
- `docs/templates/issue-first-proposal.md`

These templates are designed for agent-authored PRs to keep tone concise, policy-aware, and reviewer-friendly.
