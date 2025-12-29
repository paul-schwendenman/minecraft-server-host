# GitHub Actions Workflows

This document describes all GitHub Actions workflows in this repository.

## Overview

| Workflow | Trigger | Purpose |
|----------|---------|---------|
| [Packer Build](#packer-build) | Push to master, manual | Builds AMIs for EC2 instances |
| [Deploy Lambdas](#deploy-lambdas) | Push to master, manual | Builds and deploys Lambda functions |
| [Deploy Worlds App](#deploy-worlds-app) | Push to master, manual | Builds and deploys Worlds UI to S3/CloudFront |
| [Deploy Manager App](#deploy-manager-app) | Manual only | Builds and deploys Manager UI (legacy) |
| [AMI Cleanup](#ami-cleanup) | Weekly, manual | Removes old AMIs to reduce costs |
| [Packer Validation](#packer-validation) | Pull request | Validates Packer configuration |
| [Lambda Validation](#lambda-validation) | Pull request | Validates Python Lambda functions |
| [Web Apps Validation](#web-apps-validation) | Pull request | Validates Svelte UI apps |
| [Terraform Validation](#terraform-validation) | Pull request | Validates Terraform configurations |
| [Go Validation](#go-validation) | Pull request | Validates minecraftctl Go code |
| [minecraftctl Release](#minecraftctl-release) | Tag push, manual | Builds and releases minecraftctl binaries |

## Build Workflows

### Packer Build

**File:** `packer-build.yml`

Builds AMIs for the Minecraft server infrastructure. Uses a two-stage pipeline:

```
minecraft-base-* AMI (foundation)
        ↓
minecraft-ubuntu-* AMI (full server)
```

**Triggers:**
- Push to `master` when files in `packer/` change
- Manual via `workflow_dispatch`

**Change Detection:**
The workflow detects which AMIs need rebuilding:
- `packer/base.pkr.hcl` or `packer/scripts/base/` → rebuilds base AMI
- `packer/minecraft.pkr.hcl` or `packer/scripts/minecraft/` → rebuilds minecraft AMI
- `packer/scripts/shared/` or `minecraft_jars.auto.pkrvars.hcl` → rebuilds both

**Manual Trigger Options:**
| Input | Default | Description |
|-------|---------|-------------|
| `build_base` | `false` | Build the base AMI |
| `build_minecraft` | `true` | Build the minecraft AMI |

**AWS Authentication:** Uses OIDC. See [github-actions-aws-oidc.md](github-actions-aws-oidc.md).

### Deploy Lambdas

**File:** `lambdas-deploy.yml`

Builds and deploys Python Lambda functions to AWS. Only deploys lambdas that have changed.

**Triggers:**
- Push to `master` when files in `lambda/` change
- Manual via `workflow_dispatch`

**Change Detection:**
The workflow detects which lambdas need rebuilding/deploying:
- `lambda/control/` → deploys control lambda
- `lambda/details/` → deploys details lambda
- `lambda/worlds/` → deploys worlds lambda

**Manual Trigger Options:**
| Input | Default | Description |
|-------|---------|-------------|
| `deploy_control` | `false` | Deploy the control lambda |
| `deploy_details` | `false` | Deploy the details lambda |
| `deploy_worlds` | `false` | Deploy the worlds lambda |

**Target Resources (Test Environment):**
| Lambda | Function Name | Python |
|--------|--------------|--------|
| control | `minecraft-test-control` | 3.13 |
| details | `minecraft-test-details` | 3.12 |
| worlds | `minecraft-test-worlds` | 3.13 |

**AWS Authentication:** Uses OIDC. See [github-actions-aws-oidc.md](github-actions-aws-oidc.md).

### Deploy Worlds App

**File:** `worlds-deploy.yml`

Builds and deploys the Worlds SvelteKit app (map viewer) to S3 and CloudFront at `maps.{zone}`.

**Triggers:**
- Push to `master` when files in `minecraft-ui/apps/worlds/` or `minecraft-ui/libs/` change
- Manual via `workflow_dispatch`

**Environment Variables (per GitHub environment):**
| Variable | Description |
|----------|-------------|
| `S3_BUCKET_MAPS` | S3 bucket for worlds app (e.g., `minecraft-test-webapp-maps`) |
| `CLOUDFRONT_MAPS` | CloudFront distribution ID for maps.{zone} |

**AWS Authentication:** Uses OIDC. See [github-actions-aws-oidc.md](github-actions-aws-oidc.md).

### Deploy Manager App

**File:** `manager-deploy.yml`

Builds and deploys the Manager Vite/Svelte app (EC2 control panel) to S3 and CloudFront at `www.{zone}`.

**Triggers:**
- Push to `master` when files in `minecraft-ui/apps/manager/` or `minecraft-ui/libs/` change
- Manual via `workflow_dispatch`

**Environment Variables (per GitHub environment):**
| Variable | Description |
|----------|-------------|
| `S3_BUCKET_WWW` | S3 bucket for manager app (e.g., `minecraft-test-webapp`) |
| `CLOUDFRONT_WWW` | CloudFront distribution ID for www.{zone} |

**AWS Authentication:** Uses OIDC. See [github-actions-aws-oidc.md](github-actions-aws-oidc.md).

### AMI Cleanup

**File:** `ami-cleanup.yml`

Removes old AMIs to reduce EBS snapshot storage costs. Keeps the most recent AMIs for rollback capability.

**Triggers:**
- Weekly on Sundays at 2:00 AM UTC
- Manual via `workflow_dispatch`

**Behavior:**
- Cleans both `minecraft-base-*` and `minecraft-ubuntu-*` AMIs
- Deletes AMI and associated EBS snapshots
- Scheduled runs perform actual deletion
- Manual runs default to dry-run mode

**Manual Trigger Options:**
| Input | Default | Description |
|-------|---------|-------------|
| `dry_run` | `true` | Preview what would be deleted |
| `keep_count` | `3` | Number of recent AMIs to retain per type |

### minecraftctl Release

**File:** `minecraftctl.yml`

Builds and releases the minecraftctl CLI tool for multiple platforms.

**Triggers:**
- Push tags matching `minecraftctl-v*`
- Manual via `workflow_dispatch`

**Outputs:**
- `minecraftctl-linux-amd64`
- `minecraftctl-darwin-amd64`
- `minecraftctl-darwin-arm64`
- `checksums.txt`

Creates a GitHub Release with the built binaries.

## Validation Workflows

These workflows run on pull requests to validate code changes before merging.

### Packer Validation

**File:** `packer.yml`
**Paths:** `packer/**`

Validates all Packer configuration files:
- Installs required plugins
- Runs `packer validate` on all `.pkr.hcl` and `.hcl` files
- Special handling for `ssh.pkr.hcl` which requires secrets

### Lambda Validation

**File:** `lambdas.yml`
**Paths:** `lambda/**`

Validates Python Lambda functions using a matrix strategy:

| Lambda | Python Version |
|--------|---------------|
| control | 3.13 |
| details | 3.12 |
| worlds | 3.13 |

**Checks:**
- Dependency validation with `uv sync`
- Python syntax checking with `py_compile`
- Import validation to catch missing dependencies

### Web Apps Validation

**File:** `web-apps.yml`
**Paths:** `minecraft-ui/**`

Validates the Svelte UI monorepo:
- Installs dependencies with `pnpm install --frozen-lockfile`
- Installs Playwright browsers for testing
- Checks formatting with Prettier
- Runs linting
- Builds all projects
- Runs type checking
- Runs tests

### Terraform Validation

**File:** `terraform.yml`
**Paths:** `infra/**`

Validates Terraform configurations:
- Checks formatting with `terraform fmt -check`
- Validates all modules in `infra/modules/`
- Validates environments: `minimal`, `test`, `prod`

### Go Validation

**File:** `go.yml`
**Paths:** `minecraftctl/**`

Validates the minecraftctl Go code:
- Checks formatting with `gofmt`
- Builds the project
- Runs `go vet`
- Runs tests

## How AMIs Flow Through the System

1. **Build:** Packer Build workflow creates timestamped AMIs
2. **Use:** Terraform finds the latest AMI via `data "aws_ami"` with `most_recent = true`
3. **Cleanup:** AMI Cleanup removes old AMIs, keeping recent ones for rollback

```
packer-build.yml                    Terraform
      │                                 │
      ▼                                 ▼
minecraft-base-1702847123    data "aws_ami" "minecraft" {
minecraft-base-1703452789      most_recent = true
minecraft-ubuntu-1702847456    filter { name = "minecraft-ubuntu-*" }
minecraft-ubuntu-1703452912  }
      │                                 │
      ▼                                 ▼
ami-cleanup.yml              Uses: minecraft-ubuntu-1703452912
(keeps 3 most recent)
```

## Running Workflows Manually

All workflows with `workflow_dispatch` can be triggered from the GitHub Actions UI:

1. Go to **Actions** tab
2. Select the workflow
3. Click **Run workflow**
4. Fill in any inputs
5. Click **Run workflow**

## Secrets Required

| Secret | Used By | Description |
|--------|---------|-------------|
| `AWS_ROLE_ARN` | packer-build, ami-cleanup, lambdas-deploy, worlds-deploy, manager-deploy | IAM role ARN for AWS OIDC |
| `PACKER_TEST_HOST` | packer (validation) | Optional: Host for ssh.pkr.hcl validation |
| `PACKER_TEST_PRIVATE_KEY` | packer (validation) | Optional: SSH key for ssh.pkr.hcl validation |
