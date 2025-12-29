# Plan: Separate CloudFront Distributions for Manager and Worlds Apps

## Summary

1. Create two webapp S3 buckets per environment (one for manager, one for worlds)
2. Deploy `web_ui` module twice - once for `www.*` (manager) and once for `maps.*` (worlds)
3. Rename test domains to `*.test.{zone}` pattern
4. Update GitHub Actions to deploy each app to its respective bucket

---

## Domain Structure After Changes

| Environment | EC2 Server | Manager App (www) | Worlds App (maps) |
|-------------|------------|-------------------|-------------------|
| **Prod** | `{zone}` | `www.{zone}` | `maps.{zone}` |
| **Test** | `test.{zone}` | `www.test.{zone}` | `maps.test.{zone}` |

### CloudFront Routes

**www.* (manager app)**
- `/` → manager app (webapp-www bucket)
- `/api/*` → API Gateway (Lambdas)

**maps.* (worlds app)**
- `/` → worlds app (webapp-maps bucket)
- `/api/*` → API Gateway (Lambdas)
- `/maps/*` → maps bucket (uNmINeD exports)

---

## Implementation Steps

### Step 1: Update `s3_buckets` Module

**File:** `infra/modules/s3_buckets/main.tf`

Add second webapp bucket:
- `{name}-webapp-www` → manager app
- `{name}-webapp-maps` → worlds app
- Keep existing `{name}-maps` and `{name}-backups`

**File:** `infra/modules/s3_buckets/outputs.tf`

Add outputs for both webapp buckets.

### Step 2: Parameterize `web_ui` Module

**File:** `infra/modules/web_ui/variables.tf`

Add variable to control maps functionality:
```hcl
variable "include_maps" {
  description = "Include maps S3 bucket origin and /maps/* route"
  type        = bool
  default     = false
}

variable "map_bucket_name" {
  description = "Maps S3 bucket name (required if include_maps = true)"
  type        = string
  default     = ""
}

variable "map_bucket_domain_name" {
  description = "Maps S3 bucket domain (required if include_maps = true)"
  type        = string
  default     = ""
}
```

**File:** `infra/modules/web_ui/main.tf`

- Make maps origin conditional on `var.include_maps`
- Make `/maps/*` cache behavior conditional
- Remove `/worlds/*` route (no longer needed - app is at root)

### Step 3: Update Test Environment

**File:** `infra/test/main.tf`

1. Add `us_east_1` provider for ACM
2. Add ACM certificate module for `*.test.{zone}`
3. Update `api_lambda` dns_name: `test.{zone}` (was `testmc.{zone}`)
4. Update `dns_records` dns_name: `test.{zone}`
5. Deploy `web_ui` twice:

```hcl
# Manager app (www.test.*)
module "web_ui_www" {
  source = "../modules/web_ui"
  name   = "minecraft-test-www"

  webapp_bucket_name        = module.s3_buckets.webapp_www_bucket_name
  webapp_bucket_domain_name = module.s3_buckets.webapp_www_bucket_domain_name
  api_endpoint              = module.api_lambda.api_endpoint

  custom_domain       = "www.test.{zone}"
  acm_certificate_arn = module.acm_certificate.certificate_arn
  zone_id             = data.aws_route53_zone.prod.zone_id

  include_maps = false
  geo_whitelist = ["US", "CA", "MX"]
}

# Worlds app (maps.test.*)
module "web_ui_maps" {
  source = "../modules/web_ui"
  name   = "minecraft-test-maps"

  webapp_bucket_name        = module.s3_buckets.webapp_maps_bucket_name
  webapp_bucket_domain_name = module.s3_buckets.webapp_maps_bucket_domain_name
  api_endpoint              = module.api_lambda.api_endpoint
  map_bucket_name           = module.s3_buckets.map_bucket_name
  map_bucket_domain_name    = module.s3_buckets.map_bucket_domain_name

  custom_domain       = "maps.test.{zone}"
  acm_certificate_arn = module.acm_certificate.certificate_arn
  zone_id             = data.aws_route53_zone.prod.zone_id

  include_maps = true
  geo_whitelist = ["US", "CA", "MX"]
}
```

### Step 4: Update Production Environment

**File:** `infra/prod/main.tf`

Same pattern - deploy `web_ui` twice:
- `web_ui_www` for `www.{zone}` (manager app, no maps)
- `web_ui_maps` for `maps.{zone}` (worlds app, with maps)

### Step 5: Update GitHub Actions

**File:** `.github/workflows/manager-deploy.yml` (update existing)
- Auto-deploy on push to master when `minecraft-ui/apps/manager/` changes
- Deploy to `webapp-www` bucket
- Invalidate www CloudFront distribution

**File:** `.github/workflows/worlds-deploy.yml` (update existing)
- Auto-deploy to test, manual for prod
- Deploy to `webapp-maps` bucket
- Invalidate maps CloudFront distribution

**GitHub Environment Variables:**
| Environment | Variable | Value |
|-------------|----------|-------|
| test | `S3_BUCKET_WWW` | `minecraft-test-webapp-www` |
| test | `S3_BUCKET_MAPS` | `minecraft-test-webapp-maps` |
| test | `CLOUDFRONT_WWW` | (from terraform output) |
| test | `CLOUDFRONT_MAPS` | (from terraform output) |
| prod | (same pattern) | |

---

## Files to Modify

| File | Action |
|------|--------|
| `infra/modules/s3_buckets/main.tf` | Add second webapp bucket |
| `infra/modules/s3_buckets/outputs.tf` | Add outputs for both buckets |
| `infra/modules/web_ui/variables.tf` | Add `include_maps` variable |
| `infra/modules/web_ui/main.tf` | Make maps origin/routes conditional, remove `/worlds/*` |
| `infra/test/main.tf` | ACM cert, domain renames, two web_ui instances |
| `infra/prod/main.tf` | Two web_ui instances |
| `.github/workflows/manager-deploy.yml` | Update for auto-deploy to www bucket |
| `.github/workflows/worlds-deploy.yml` | Update bucket/distribution variables |

---

## Execution Order

1. Update `s3_buckets` module (add second webapp bucket)
2. Update `web_ui` module (parameterize maps functionality)
3. Apply test environment (`terraform apply` - creates new buckets, ACM cert ~10 mins)
4. Apply prod environment (`terraform apply`)
5. Update GitHub workflows
6. Configure GitHub environment variables with new CloudFront distribution IDs
7. Deploy apps to new buckets

---

## Migration Notes

- Existing `{name}-webapp` bucket can be renamed or kept alongside new buckets
- May need to migrate existing deployed files to new bucket structure
- Old `/worlds/*` path will 404 after change - ensure no external links depend on it
