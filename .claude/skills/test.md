# Test

Run tests for a specific component or auto-detect from context.

## Usage

`/test [component]`

## Components

| Component | Command |
|-----------|---------|
| minecraftctl | `cd minecraftctl && go test ./...` |
| control | `cd lambda/control && uv run pytest tests/ -v` |
| details | `cd lambda/details && uv run pytest tests/ -v` |
| worlds | `cd lambda/worlds && uv run pytest tests/ -v` |
| ui | `cd minecraft-ui && pnpm -r test` |
| packer | `cd packer && bats tests/*.bats` |

## Behavior

1. If component specified, run that component's tests
2. If no component, detect from current working directory
3. If at repo root, ask which component to test or run all
4. Report test results clearly with pass/fail summary
