# Infrastructure (Terraform)

Terraform configuration for the Minecraft server infrastructure on AWS.

## Environments

| Environment | Directory | Purpose |
|-------------|-----------|---------|
| `prod/` | Production | Live server |
| `test/` | Testing | Test server |
| `minimal/` | Starter | Minimal config for new deployments |

## Domain Structure

### Production
| Domain | Service | Notes |
|--------|---------|-------|
| `{zone}` | EC2 (game server) | A/AAAA records |
| `www.{zone}` | CloudFront | Main web UI |

### Test
| Domain | Service | Notes |
|--------|---------|-------|
| `testmc.{zone}` | EC2 (game server) | A record |
| (CloudFront default) | CloudFront | No custom domain by default |

### CloudFront Routes
- `/` → Landing page (webapp S3 bucket)
- `/api/*` → API Gateway (Lambda functions)
- `/maps/*` → Maps S3 bucket (uNmINeD exports)
- `/worlds/*` → Worlds app (webapp S3 bucket)

## Modules

| Module | Purpose |
|--------|---------|
| `networking/` | VPC, subnets, internet gateway, route tables |
| `s3_buckets/` | Three buckets: webapp, maps, backups |
| `ec2_role/` | IAM role for EC2 (S3 + Route53 access) |
| `mc_stack/` | EC2 instance, security groups, EBS volumes |
| `api_lambda/` | Lambda functions + API Gateway + Route53 DNS |
| `web_ui/` | CloudFront distribution + S3 bucket policies |
| `acm_certificate/` | ACM wildcard certificate with DNS validation |
| `dns_records/` | Route53 A/AAAA records |

## Module Dependencies

```
networking
    ↓
s3_buckets ──────────────────┐
    ↓                        ↓
ec2_role ───→ mc_stack ───→ api_lambda
                               ↓
                            web_ui ←── acm_certificate (prod only)
                               ↓
                          dns_records (prod only)
```

## Key Differences: Test vs Prod

| Aspect | Test | Prod |
|--------|------|------|
| Instance type | t3.small | t3.medium |
| VPC CIDR | 10.0.0.0/16 | 10.1.0.0/16 |
| Custom domain | No | Yes (www.{zone}) |
| ACM certificate | No | Yes (*.{zone}) |
| Geo restrictions | US, CA, MX only | None |
| Route53 zone | References existing | Creates/manages zone |

## Usage

```bash
# Initialize
cd infra/test  # or prod
terraform init

# Plan changes
terraform plan

# Apply changes
terraform apply

# View outputs
terraform output
```

## S3 Buckets

Each environment creates three buckets:

| Bucket | Purpose | Features |
|--------|---------|----------|
| `{name}-webapp` | Svelte UI static files | Website config, CloudFront access |
| `{name}-maps` | uNmINeD map exports | Versioning, encryption, CloudFront access |
| `{name}-backups` | World backups | Versioning, encryption, lifecycle (Glacier 30d, delete 90d) |

## CloudFront

Single distribution per environment with multiple origins:
1. **webapp-origin** → S3 webapp bucket (OAC access)
2. **api-origin** → API Gateway (HTTPS only)
3. **maps-origin** → S3 maps bucket (OAC access)

CloudFront function `maps-append-index.js` handles directory-style requests under `/maps/*`.
