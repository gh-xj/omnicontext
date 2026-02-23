# Commands Reference

## Lab Commands

```bash
ocx lab init
ocx lab run --config docs/templates/lab-config.example.json
ocx lab run --goal "..." --verify "go test ./..." --inspector "<command>"
ocx lab run --json ...
```

## Inspector Contract Example

```bash
cat > "$OCX_LAB_INSPECTOR_JSON_FILE" <<'JSON'
{"verdict":"QUALIFIED","reasons":["all checks passed"],"patch_hints":[],"confidence":0.95}
JSON
```

## Recommended Verifier Commands

```bash
go vet ./...
go test ./...
go build ./cmd/ocx
```

## Ingest + Inspect Support

```bash
ocx ingest auto --dry-run --json
ocx context stats default
ocx session list --limit 50
ocx session show <session-id>
```
