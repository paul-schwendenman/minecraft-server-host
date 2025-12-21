# Packer AMI Builds

AMI configuration and provisioning scripts for the Minecraft server. See `readme.rst` for full architecture details.

## Build

```bash
AWS_PROFILE=minecraft packer build -var-file=minecraft_jars.auto.pkrvars.hcl base.pkr.hcl
AWS_PROFILE=minecraft packer build -var-file=minecraft_jars.auto.pkrvars.hcl minecraft.pkr.hcl
```

## Test

```bash
bats tests/*.bats                # Run all bats tests
bats tests/shellcheck.bats       # Run shellcheck on all scripts
```

## Structure

- `base.pkr.hcl` - Foundation AMI (Java, dependencies)
- `minecraft.pkr.hcl` - Minecraft-specific AMI (scripts, services)
- `scripts/base/` - Base AMI provisioning scripts
- `scripts/minecraft/` - Minecraft AMI provisioning scripts
- `scripts/shared/` - Scripts used by both AMIs
- `tests/` - Bats tests for shell scripts
