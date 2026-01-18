# Packer Expert

Specialized guidance for working in `packer/` - AMI builds with Packer.

## Project Structure

```
packer/
├── base.pkr.hcl              # Foundation AMI (Ubuntu + system deps)
├── minecraft.pkr.hcl         # Application AMI (layered on base)
├── ssh.pkr.hcl               # Null builder for live testing
├── docker.pkr.hcl            # Docker builder for local dev
├── minecraft_jars.auto.pkrvars.hcl  # JAR versions with SHA256
├── scripts/
│   ├── base/                 # Base AMI provisioning
│   ├── minecraft/            # Minecraft installation scripts
│   │   ├── install_*.sh      # One script per feature
│   │   ├── autoshutdown/     # Autoshutdown service files
│   │   ├── create-world/     # World creation scripts
│   │   └── maps/             # Map generation scripts
│   └── shared/               # Used by multiple builds
└── tests/
    ├── test_helper.bash      # Bats test utilities
    ├── shellcheck.bats       # Script linting
    └── *.bats                # Feature tests
```

## AMI Layering

```
Ubuntu 22.04 LTS (Canonical)
    └── base AMI (minecraft-base-*)
        - OpenJDK 21, AWS CLI v2, Python tools
        └── minecraft AMI (minecraft-ubuntu-*)
            - Minecraft services, minecraftctl, maps
```

## Conventions

### Provisioner Pattern
```hcl
build {
  sources = ["source.amazon-ebs.minecraft"]

  # 1. Create temp directory
  provisioner "shell" {
    inline = ["mkdir -p /tmp/scripts"]
  }

  # 2. Copy scripts
  provisioner "file" {
    source      = "scripts/minecraft/"
    destination = "/tmp/scripts/"
  }

  # 3. Run installation scripts (in order)
  provisioner "shell" {
    script = "scripts/minecraft/install_autoshutdown.sh"
  }

  # 4. Pass variables via environment
  provisioner "shell" {
    environment_vars = [
      "MINECRAFT_JARS_JSON=${jsonencode(var.minecraft_jars)}"
    ]
    script = "scripts/shared/install_jars.sh"
  }

  # 5. Cleanup
  provisioner "shell" {
    inline = [
      "sudo systemctl daemon-reload",
      "sudo apt-get clean",
      "rm -rf /tmp/scripts"
    ]
  }
}
```

### Variable Pattern
```hcl
variable "minecraft_jars" {
  type = list(object({
    version = string
    url     = string
    sha256  = string
  }))
  default     = []
  description = "Minecraft server JARs to install"
}
```

### Script Conventions
```bash
#!/bin/bash
set -euxo pipefail  # Always use strict mode

# Read JSON from environment
JARS_JSON="${MINECRAFT_JARS_JSON:-}"
if [ -z "$JARS_JSON" ]; then
    echo "No JARs specified, skipping"
    exit 0
fi
```

## Testing with Bats

```bash
# tests/create-world.bats
load test_helper

setup() {
    setup_test_fixtures
    SCRIPT=$(wrap_script "../scripts/minecraft/create-world/create-world.sh")
}

teardown() {
    teardown_test_fixtures
}

@test "validates world name argument" {
    run bash "$SCRIPT"
    [ "$status" -ne 0 ]
    [[ "$output" == *"Usage:"* ]]
}

@test "calls minecraftctl with correct args" {
    create_mock minecraftctl 0 ""
    run bash "$SCRIPT" myworld 1.21.4
    assert_mock_called_with "minecraftctl world create myworld"
}
```

## Guidelines

1. **SHA256 verification** - Always verify downloaded files
2. **Strict mode** - All scripts use `set -euxo pipefail`
3. **Modular scripts** - One install_*.sh per feature
4. **Minimal installers** - Copy files, reload systemd, that's it
5. **JSON for complex vars** - Use `jsonencode()` for lists/objects
6. **Test with mocks** - Avoid network/filesystem deps in tests

## Commands

```bash
cd packer

# Validate
packer validate base.pkr.hcl
packer validate minecraft.pkr.hcl

# Build
packer build base.pkr.hcl
packer build minecraft.pkr.hcl

# Test
bats tests/*.bats
```
