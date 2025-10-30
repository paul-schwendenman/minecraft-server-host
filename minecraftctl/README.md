# minecraftctl

A CLI tool for managing Minecraft worlds, maps, and RCON commands. Built with Go, Cobra, Viper, and Zerolog.

## Features

- **World Management**: List and inspect Minecraft worlds
- **Map Building**: Build static maps using uNmINeD based on per-world `map-config.yml` files
- **RCON Integration**: Send commands to Minecraft servers via RCON
- **Configurable**: Support for global config, environment variables, and per-world settings

## Installation

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
- `MINECRAFT_RCON_HOST` - RCON host (default: `127.0.0.1`)
- `MINECRAFT_RCON_PORT` - RCON port (default: `25575`)
- `MINECRAFT_RCON_PASSWORD` - RCON password

### Example Config File

```yaml
worlds_dir: /srv/minecraft-server
maps_dir: /srv/minecraft-server/maps
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

### Build Maps

```bash
# Build all maps for a world
minecraftctl map build <world-name>

# Build a specific map
minecraftctl map build <world-name> --map overworld

# Force rebuild
minecraftctl map build <world-name> --force
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

## Map Configuration

Each world can have a `map-config.yml` file that defines how maps are rendered. See `docs/map-build-config.rst` in the parent project for details.

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
```

## License

Same as parent project.

