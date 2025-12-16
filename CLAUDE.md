# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

On-demand Minecraft server infrastructure on AWS. Web app controls EC2 instances that auto-shutdown when idle.

## Build Commands

```bash
make control|details|worlds  # Build individual lambda -> dist/<name>.zip
make lambdas                 # Build all lambdas
make ui                      # Build minecraft-ui/apps/manager
make deploy                  # Deploy lambdas + UI to AWS
```

**UI dev**: `cd minecraft-ui && pnpm install && pnpm dev:manager`

**minecraftctl CLI**: See `minecraftctl/README.md` for Go build commands.

**Packer AMIs**: See `packer/readme.rst` for AMI build commands.

## Architecture

- `lambda/` - Python Lambda functions:
  - `control/` - EC2 start/stop/status (FastAPI/Mangum)
  - `details/` - Minecraft server ping
  - `worlds/` - Map manifest API (reads from S3, enriches with preview URLs)
- `minecraft-ui/` - pnpm monorepo with Svelte 5 apps (`@minecraft/*` packages)
- `minecraftctl/` - Go CLI for server-side world/map management (see `minecraftctl/CLAUDE.md`)
- `packer/` - AMI builds: base.pkr.hcl (foundation) → minecraft.pkr.hcl (server)
- `infra/` - Terraform modules (test/ and prod/ environments)
