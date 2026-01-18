# Go Expert

Specialized guidance for working in `minecraftctl/` - Go CLI for server management.

## Project Structure

```
minecraftctl/
├── cmd/minecraftctl/
│   ├── main.go           # Entry point, command registration
│   ├── root/root.go      # Root command with config init
│   ├── world.go          # World commands (list, info, create, etc.)
│   ├── map.go            # Map commands (build, preview, manifest)
│   ├── jar.go            # JAR management
│   ├── rcon.go           # RCON client
│   └── backup.go         # Restic backup commands
├── pkg/
│   ├── worlds/           # World management, symlinks, systemd
│   ├── maps/             # Map building with uNmINeD
│   ├── jars/             # JAR downloads, checksums
│   ├── rcon/             # RCON client
│   ├── config/           # Viper-based config
│   ├── systemd/          # systemctl/journalctl wrappers
│   ├── nbt/              # NBT level.dat reader
│   └── backup/           # Restic integration
└── internal/commands/    # Shared command definitions
```

## Conventions

### Cobra Commands
```go
// Package-level command variable
var worldListCmd = &cobra.Command{
    Use:   "list",
    Short: "List all worlds",
    Long:  `Detailed description...`,
    RunE: func(cmd *cobra.Command, args []string) error {
        // Return error, don't call os.Exit
        return listWorlds()
    },
}

// Flags in init()
var listFull bool

func init() {
    worldListCmd.Flags().BoolVarP(&listFull, "full", "f", false, "Show full details")
    WorldCmd.AddCommand(worldListCmd)
}
```

### Error Handling
```go
// Always wrap errors with context
if err != nil {
    return fmt.Errorf("failed to read world %s: %w", name, err)
}

// Graceful degradation for non-critical failures
if err := enableTimer(world); err != nil {
    log.Warn().Err(err).Msg("failed to enable timer, continuing")
}
```

### Config Pattern
```go
// pkg/config/config.go - Viper singleton
func Get() *GlobalConfig {
    return &GlobalConfig{
        MinecraftDir: viper.GetString("minecraft_dir"),
        // ...
    }
}

// Priority: CLI flags > env vars > config file > defaults
```

### Options Structs
```go
// For functions with many parameters
type CreateWorldOptions struct {
    Name       string
    Version    string
    EnableMaps bool
}

func CreateWorld(opts CreateWorldOptions) error {
    // ...
}
```

## Guidelines

1. **Return errors** - Use `RunE`, never panic or `os.Exit` in commands
2. **Wrap errors** - Always `fmt.Errorf("context: %w", err)`
3. **Zerolog** - Use `log.Info().Str("world", name).Msg("created")` for structured logs
4. **filepath.Join** - Never concatenate paths with `/`
5. **Table-driven tests** - `tests := []struct{...}` pattern
6. **Flag cleanup** - Reset flag values in tests to avoid state bleed

## Commands

```bash
cd minecraftctl
make build                  # Build for current OS
make build-linux-amd64      # Cross-compile for prod server
make test                   # Run tests
go test -race ./...         # Race detection
make fmt                    # Format code
make vet                    # Static analysis
```
