# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

On-demand Minecraft server infrastructure on AWS. Web app controls EC2 instances that auto-shutdown when idle.

## Build Commands

```bash
make control|details|worlds  # Build individual lambda -> dist/<name>.zip
make lambdas                 # Build all lambdas
make ui                      # Build minecraft-ui/apps/manager
make deploy                  # Deploy lambdas + UI to AWS (manual)
```

**UI dev**: `cd minecraft-ui && pnpm install && pnpm dev:manager`

**minecraftctl CLI**: See `minecraftctl/README.md` for Go build commands.

**Packer AMIs**: See `packer/readme.rst` for AMI build commands.

## Testing & Validation

| Subproject | Command |
|------------|---------|
| minecraftctl | `cd minecraftctl && go test ./...` |
| lambda/* | `cd lambda/<name> && uv run pytest tests/ -v` |
| minecraft-ui | `cd minecraft-ui && pnpm -r test` |
| packer | `cd packer && bats tests/*.bats` |

## CI/CD Deployments

Lambdas and UI apps auto-deploy on push to `master`. See `docs/github-actions.md` for details.

| Component | Workflow | Trigger |
|-----------|----------|---------|
| Lambdas | `lambdas-deploy.yml` | Auto on `lambda/` changes |
| Worlds App | `worlds-deploy.yml` | Auto on `minecraft-ui/` changes |
| Manager App | `manager-deploy.yml` | Manual only (legacy) |

## Architecture

- `lambda/` - Python Lambda functions:
  - `control/` - EC2 start/stop/status (FastAPI/Mangum)
  - `details/` - Minecraft server ping
  - `worlds/` - Map manifest API (reads from S3, enriches with preview URLs)
- `minecraft-ui/` - pnpm monorepo with Svelte 5 apps (`@minecraft/*` packages)
- `minecraftctl/` - Go CLI for server-side world/map management (see `minecraftctl/CLAUDE.md`)
- `packer/` - AMI builds: base.pkr.hcl (foundation) → minecraft.pkr.hcl (server)
- `infra/` - Terraform infrastructure (see `infra/README.md`)
  - `test/` and `prod/` environments with separate VPCs
  - `modules/` - Shared: networking, s3_buckets, ec2_role, mc_stack, api_lambda, web_ui, acm_certificate, dns_records

## Agent Usage

Use subagents to keep context focused:

**Explore agent** - Use for:
- "How does X work?" questions
- Finding where functionality is implemented
- Understanding cross-component interactions (Lambda ↔ UI ↔ minecraftctl)

**Plan agent** - Use for:
- Changes spanning multiple subprojects
- New Lambda endpoints that need UI changes
- Infrastructure changes (Terraform + Packer)

**Bash agent** - Use for:
- Running test suites across subprojects
- Build operations
- Git operations
