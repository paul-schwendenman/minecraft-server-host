# minecraftctl

A CLI tool for managing Minecraft worlds, maps, and RCON commands. Built with Go, Cobra, Viper, and Zerolog.

## Features

- **World Management**: List, inspect, and create Minecraft worlds
- **Map Building**: Build static maps using uNmINeD based on per-world `map-config.yml` files
- **RCON Integration**: Send commands to Minecraft servers via RCON
- **JAR Management**: Download, list, and verify Minecraft server JAR files with checksum support
- **Configurable**: Support for global config, environment variables, and per-world settings

## Installation

### From GitHub Releases (Recommended)

Install the latest release with a single command:

```bash
curl -fsSL https://raw.githubusercontent.com/paul-schwendenman/minecraft-server-host/master/minecraftctl/install.sh | bash
```

To also install man pages and shell completions:

```bash
curl -fsSL https://raw.githubusercontent.com/paul-schwendenman/minecraft-server-host/master/minecraftctl/install.sh | INSTALL_MAN=1 INSTALL_COMPLETIONS=1 bash
```

### From Source

```bash
make build
sudo make install
```

Or build manually:

```bash
go build -ldflags "-X github.com/paul/minecraftctl/internal/version.Version=$(git describe --tags --always --dirty)" -o minecraftctl ./cmd/minecraftctl
```

## Configuration

Configuration can be provided via:

1. CLI flags (highest priority)
2. Environment variables (e.g., `MINECRAFT_WORLDS_DIR`, `MINECRAFT_RCON_HOST`)
3. Config file (`/etc/minecraftctl.yml` or `~/.config/minecraftctl.yml`)
4. Defaults

### Environment Variables

- `MINECRAFT_WORLDS_DIR` - Directory containing Minecraft worlds (default: `/srv/minecraft-server`)
- `MINECRAFT_MAPS_DIR` - Directory for map output (default: `/srv/minecraft-server/maps`)
- `MINECRAFT_JARS_DIR` - Directory containing Minecraft server JARs (default: `/opt/minecraft/jars`)
- `MINECRAFT_RCON_HOST` - RCON host (default: `127.0.0.1`)
- `MINECRAFT_RCON_PORT` - RCON port (default: `25575`)
- `MINECRAFT_RCON_PASSWORD` - RCON password

### Example Config File

```yaml
worlds_dir: /srv/minecraft-server
maps_dir: /srv/minecraft-server/maps
jars_dir: /opt/minecraft/jars
rcon:
  host: 127.0.0.1
  port: 25575
  password: ${RCON_PASSWORD}
```

## Usage

### List Worlds

```bash
minecraftctl world list
```

### Show World Info

```bash
minecraftctl world info <world-name>
```

### Create World

```bash
# Create a new world with required version
minecraftctl world create <world-name> --version <version>

# Create a world with a seed
minecraftctl world create <world-name> --version <version> --seed <seed>

# Create a world without map-config.yml
minecraftctl world create <world-name> --version <version> --no-map-config

# Create a world without enabling systemd service
minecraftctl world create <world-name> --version <version> --no-systemd

# Example: Create a world with version 1.21.1 and seed
minecraftctl world create vanilla-121 --version 1.21.1 --seed 8675309
```

**Note**: The `world create` command requires:
- The Minecraft server jar to be installed at `/opt/minecraft/jars/minecraft_server_<version>.jar`
- Proper permissions (may require `sudo` for systemd operations)
- RCON configuration (from `/etc/minecraft.env` or config file)

### Register World

```bash
# Register an existing world with systemd services
minecraftctl world register <world-name>
```

The `world register` command is used to "reattach" an existing world to a new server instance. This is particularly useful when booting from an existing EBS world volume. It:

- Enables and starts `minecraft@<world>.service`
- Enables `minecraft-map-build@<world>.timer`
- Enables `minecraft-world-backup@<world>.timer`
- Enables `minecraft-map-backup@<world>.timer`
- Reloads systemd daemon

**Note**: The `world register` command does NOT modify any world files (eula.txt, server.properties, map-config.yml, etc.). It only sets up systemd services and timers for an existing world.

### Build Maps

```bash
# Build all maps for a world immediately
minecraftctl map build now <world-name>

# Build a specific map
minecraftctl map build now <world-name> --map overworld

# Force rebuild
minecraftctl map build now <world-name> --force

# Check status of map build timer/service
minecraftctl map build status <world-name>

# Enable/disable automatic map builds
minecraftctl map build enable <world-name>
minecraftctl map build disable <world-name>
```

### RCON Commands

```bash
# Check server status
minecraftctl rcon status

# Send a command
minecraftctl rcon send "say Hello World"

# Execute commands from file
minecraftctl rcon exec commands.txt
```

### JAR Management

```bash
# List installed JARs
minecraftctl jar list

# Download a JAR (with checksum verification)
minecraftctl jar download 1.21.11 --url https://piston-data.mojang.com/v1/objects/.../server.jar --sha256 <checksum>

# Download a JAR (checksum from checksums.txt if available)
minecraftctl jar download 1.21.11 --url https://piston-data.mojang.com/v1/objects/.../server.jar

# Verify a JAR's checksum
minecraftctl jar verify 1.21.11

# Show JAR details
minecraftctl jar info 1.21.11
```

**Note**: The `jar download` command:
- Downloads JARs to `/opt/minecraft/jars/` (or `MINECRAFT_JARS_DIR`)
- Verifies checksums against provided `--sha256` flag or `checksums.txt` file
- Updates `checksums.txt` after successful download
- Uses `sha256sum`-compatible format for checksums

## Map Configuration

Each world can have a `map-config.yml` file that defines how maps are rendered. See `docs/map-build-config.rst` in the parent project for details.

## Cross-Compilation

To build for Linux amd64 (for EC2 instances):

```bash
make build-linux-amd64
```

This creates `minecraftctl-linux-amd64` which can be copied to your Linux server:

```bash
# Copy to remote server
scp minecraftctl-linux-amd64 user@your-server:/tmp/

# On the server, install it
sudo install -m 755 /tmp/minecraftctl-linux-amd64 /usr/local/bin/minecraftctl
```

To build for multiple platforms:

```bash
make build-all
```

This creates binaries in the `dist/` directory for:
- Linux amd64
- macOS amd64
- macOS arm64

## Documentation

### Man Pages

Man pages are auto-generated from the command definitions. After installation with `INSTALL_MAN=1`, view them with:

```bash
man minecraftctl
man minecraftctl-world-create
man minecraftctl-map-build
```

To generate man pages locally:

```bash
make docs
man ./man/man1/minecraftctl.1
```

### Shell Completions

Shell completions are available for bash, zsh, and fish. Install with `INSTALL_COMPLETIONS=1` or generate locally:

```bash
make docs
```

This creates:
- `completions/minecraftctl.bash` - Bash completion
- `completions/minecraftctl.zsh` - Zsh completion
- `completions/minecraftctl.fish` - Fish completion

To manually install completions:

```bash
# Bash
sudo install -m 644 completions/minecraftctl.bash /etc/bash_completion.d/minecraftctl

# Zsh (add to fpath)
sudo install -m 644 completions/minecraftctl.zsh /usr/local/share/zsh/site-functions/_minecraftctl

# Fish
install -m 644 completions/minecraftctl.fish ~/.config/fish/completions/
```

## Development

```bash
# Build
make build

# Format code
make fmt

# Run linter
make lint

# Run tests
make test

# Generate man pages and completions
make docs

# Install man pages to system
sudo make install-man
```

## License

Same as parent project.

