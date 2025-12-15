# minecraftctl CLI

Go CLI for managing Minecraft server worlds, maps, JARs, and RCON.

## Build

```bash
go build -o minecraftctl ./cmd/minecraftctl
```

## Architecture

Uses [Cobra](https://github.com/spf13/cobra) for commands. Entry point: `cmd/minecraftctl/main.go`

### Commands

| Command | File | Description |
|---------|------|-------------|
| `world` | `cmd/minecraftctl/world.go` | list, info, create, register, upgrade |
| `map` | `cmd/minecraftctl/map.go` | build, preview, manifest, index, config |
| `jar` | `cmd/minecraftctl/jar.go` | list, download, verify, info |
| `rcon` | `cmd/minecraftctl/rcon.go` | status, send, exec |

### Packages

| Package | Purpose |
|---------|---------|
| `pkg/worlds` | World management, symlinks, systemd services |
| `pkg/jars` | JAR downloads, checksums |
| `pkg/maps` | Map building with uNmINeD |
| `pkg/rcon` | RCON client |
| `pkg/config` | Global config via Viper |
| `pkg/nbt` | NBT level.dat reader |

## Key Concepts

**World structure:** `/srv/minecraft-server/{world}/`
- `server.jar` - symlink to `/opt/minecraft/jars/minecraft_server_{version}.jar`
- `world/level.dat` - NBT file with world metadata (version, spawn, etc.)
- `map-config.yml` - per-world map rendering config

**Systemd services:** `minecraft@{world}.service` for each world

**Version tracking:** The JAR version comes from the `server.jar` symlink target filename, not the NBT level.dat (which is updated by Minecraft itself when the server runs).
