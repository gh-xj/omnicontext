# PR Gate (After Qualification)

## Required Before Review

- Verifier logs show all required checks passed.
- Judge result is `QUALIFIED`.
- Scope is narrow and intentional.
- PR body includes what/why/how and compatibility notes.

## Suggested Artifacts in PR

- run id and report path
- verification summary
- risk + rollback note

## Useful Commands

```bash
gh pr checks <n>
gh pr view <n> --comments
gh api repos/gh-xj/omnicontext/pulls/<n>/reviews
```
