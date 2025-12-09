# Minecraft Server Management UI

## Get started

Install the dependencies...

```bash
pnpm install
```

```bash
pnpm dev
```

## Building and running in production mode

To create an optimized version of the app:

```bash
pnpm build
aws s3 sync dist/ s3://your-bucket-name --delete
aws cloudfront create-invalidation --distribution-id XYZ --paths "/*"
```
