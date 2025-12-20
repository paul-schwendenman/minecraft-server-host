# minecraftctl CLI

Go CLI for managing Minecraft server worlds, maps, JARs, and RCON.

## Build

```bash
go build -o minecraftctl ./cmd/minecraftctl
```

## Test & Lint

```bash
go test ./...                    # Run all tests
go test -race ./...              # Run with race detection
go fmt ./...                     # Format code
go vet ./...                     # Static analysis
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

## Maps System

**Maps vs Dimensions:** A "map" is a rendering configuration, not the same as a Minecraft dimension. You can have multiple maps for the same dimension (e.g., a surface map and a cave map both for the overworld, with different Y-level/rendering settings).

**Map config:** Each world has a `map-config.yml` defining its maps:
```yaml
defaults:
  zoomout: 6
  imageformat: jpeg
maps:
  - name: overworld      # Map name (used in URLs/paths)
    dimension: overworld # Minecraft dimension to render
    output_subdir: overworld
  - name: caves
    dimension: overworld # Same dimension, different settings
    options:
      topY: 64
```

**Output structure:** Maps are rendered to `/srv/minecraft-server/maps/` (symlinked from `/var/www/map/`):
```
maps/
├── world_manifest.json     # Aggregate of all worlds
├── index.html              # HTML index page
└── {world}/
    ├── manifest.json       # World metadata + list of maps
    ├── preview.png         # World preview (copied from overworld map)
    └── {map-name}/
        ├── manifest.json   # Map manifest
        ├── preview.png     # Map preview (64-block radius around spawn)
        └── [tile files]    # uNmINeD web render output
```

**Data flow:** CLI generates manifests → S3 backup → Lambda API enriches with URLs → UI displays

**Preview generation:** Uses uNmINeD to render a 64-block radius around spawn. World-level preview is copied from the first overworld map.
