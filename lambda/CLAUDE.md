# Lambda Functions

Three Python Lambda functions for the Minecraft server control plane.

## Functions

| Lambda | Python | Purpose |
|--------|--------|---------|
| `control/` | 3.13 | EC2 start/stop/status, Route53 DNS sync (FastAPI + Mangum) |
| `details/` | 3.12 | Minecraft server ping via mcstatus |
| `worlds/` | 3.13 | Map manifest API, S3 integration |

## Build

```bash
make control    # Build dist/control.zip
make details    # Build dist/details.zip
make worlds     # Build dist/worlds.zip
make lambdas    # Build all
```

## Deploy

**CI/CD (recommended):** Push to `master` triggers `lambdas-deploy.yml` - only changed lambdas deploy.

**Manual:** `make deploy-lambdas` (requires AWS credentials)

## Test Environment

- `minecraft-test-control`
- `minecraft-test-details`
- `minecraft-test-worlds`
- Region: `us-east-2`
