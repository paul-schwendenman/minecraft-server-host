# Minecraft Worlds Viewer

SvelteKit app for viewing Minecraft world maps. Uses static adapter for S3/CloudFront deployment.

## Development

```bash
pnpm install
pnpm dev
```

## Deployment

**CI/CD (recommended):** Push changes to `master` branch. The `worlds-deploy.yml` workflow automatically builds and deploys to S3/CloudFront.

**Manual:**

```bash
pnpm build
aws s3 sync build/ s3://minecraft-test-webapp --delete
aws cloudfront create-invalidation --distribution-id E35JG9QWEEVI98 --paths "/*"
```

## Building

```bash
pnpm build      # Output: build/
pnpm preview    # Preview production build locally
```
