# AI PR Checklist

## Policy and Strategy

- [ ] Confirm contribution policy in `CONTRIBUTING.md`
- [ ] For large changes, align via issue/discussion first

## Readiness

- [ ] Scope is narrow and intentional
- [ ] No unrelated files included
- [ ] Commands pass locally:
  - [ ] `go vet ./...`
  - [ ] `go test ./...`
  - [ ] `go build ./cmd/ocx`

## Review-Friendliness

- [ ] PR explains what/why/how in plain language
- [ ] Compatibility notes are explicit
- [ ] Risk + rollback/mitigation is explicit
- [ ] Tone is concise and respectful

## Feedback Loop

- [ ] Check comments: `gh pr view <n> --comments`
- [ ] Check reviews: `gh api repos/gh-xj/omnicontext/pulls/<n>/reviews`
- [ ] Check CI status: `gh pr checks <n>`

## First-Time Contributor Gate

- [ ] Keep initial PR conservative
- [ ] If complete, mark ready: `gh pr ready <n>`
- [ ] Ask maintainers before broad follow-up roadmap PRs
