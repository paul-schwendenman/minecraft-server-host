# Minecraft UI Monorepo

pnpm monorepo with Svelte 5 apps and shared libraries.

## Structure

```
apps/
  manager/    # Legacy control panel (Vite + Svelte 5)
  worlds/     # Map viewer (SvelteKit + static adapter)
libs/
  ui/         # Shared UI components
  data/       # Shared data types/utilities
```

## Development

```bash
pnpm install
pnpm dev:manager    # Run manager app
pnpm dev:worlds     # Run worlds app (from apps/worlds: pnpm dev)
```

## Build

```bash
# From monorepo root
pnpm -r build       # Build all

# Individual apps
cd apps/manager && pnpm build   # Output: dist/
cd apps/worlds && pnpm build    # Output: build/
```

## Test & Lint

```bash
pnpm -r test                     # Run all tests
pnpm -r lint                     # Run linting
pnpm -r check                    # Type checking (svelte-check)
pnpm -r format                   # Format code (prettier)
```

## Deploy

**CI/CD (recommended):**
- Worlds app: Push to `master` triggers `worlds-deploy.yml` (auto)
- Manager app: Manual trigger via `manager-deploy.yml` (legacy)

**Manual:** `make deploy-ui` (requires AWS credentials)

## Test Environment

- S3 Bucket: `minecraft-test-webapp`
- CloudFront: `E35JG9QWEEVI98`
