# Validate

Run full validation suite before creating a PR. Checks all subprojects.

## Usage

`/validate`

## Checks Performed

### minecraftctl (Go)
```bash
cd minecraftctl
go test ./...
go vet ./...
go fmt ./... # Check for unformatted files
```

### Lambda Functions (Python)
```bash
cd lambda/control && uv run pytest tests/ -v
cd lambda/details && uv run pytest tests/ -v
cd lambda/worlds && uv run pytest tests/ -v
```

### UI (Svelte/TypeScript)
```bash
cd minecraft-ui
pnpm -r test
pnpm -r lint
pnpm -r check
```

### Packer (Bats)
```bash
cd packer && bats tests/*.bats
```

## Behavior

1. Run all checks across all subprojects
2. Collect failures and report at the end
3. Exit with summary: X passed, Y failed
4. If all pass, confirm ready for PR
