# Minecraft Server Management UI

Legacy control panel for starting/stopping the Minecraft server.

## Development

```bash
pnpm install
pnpm dev
```

## Deployment

**CI/CD:** Manually trigger `manager-deploy.yml` workflow from GitHub Actions (this is a legacy app, not auto-deployed).

**Manual:**

```bash
pnpm build
aws s3 sync dist/ s3://minecraft-test-webapp --delete
aws cloudfront create-invalidation --distribution-id E35JG9QWEEVI98 --paths "/*"
```

Or use `make deploy-ui` from repo root.
