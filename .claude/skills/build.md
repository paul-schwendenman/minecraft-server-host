# Build

Build lambdas, UI, or CLI components.

## Usage

`/build [target]`

## Targets

| Target | Command |
|--------|---------|
| lambdas | `make lambdas` |
| control | `make control` |
| details | `make details` |
| worlds | `make worlds` |
| ui | `cd minecraft-ui && pnpm -r build` |
| minecraftctl | `cd minecraftctl && go build -o minecraftctl ./cmd/minecraftctl` |
| all | Build lambdas + ui + minecraftctl |

## Behavior

1. If target specified, build that target
2. If no target, detect from current working directory
3. Report build success/failure with any errors
4. For lambdas, output goes to `dist/<name>.zip`
