# minecraftctl Enhancement Plan

## Overview

This document outlines enhancements to `minecraftctl` to add missing features identified during the packer script migration review. These enhancements will allow `minecraftctl` to fully replace bash wrapper scripts.

## Current Gaps Identified

1. **Lock file management** - No protection against concurrent map builds
2. **Glob/wildcard support** - Cannot process multiple worlds via patterns
3. **Preview generation in manifest step** - Must be called separately
4. **Non-blocking mode** - No graceful handling when lock is held
5. **Batch operations** - Limited support for processing multiple worlds
6. **Aggregate manifest & HTML index** - Missing global aggregate manifest and HTML index page generation

## Enhancement Specifications

### Enhancement 1: Lock File Management

**Goal**: Prevent concurrent map builds from conflicting.

**Requirements**:
- Use file locking mechanism (similar to `flock`)
- Default lock file: `/tmp/minecraft-map-build.lock`
- Configurable lock file path
- Automatic cleanup on exit (signal handling)
- Thread-safe implementation

**Design**:
```go
// New package: pkg/lock/lock.go
package lock

type FileLock struct {
    path string
    file *os.File
}

func NewFileLock(path string) *FileLock
func (fl *FileLock) TryLock(timeout time.Duration) error
func (fl *FileLock) Lock() error  // Blocks until acquired
func (fl *FileLock) Unlock() error
```

**Implementation Steps**:
1. Create `pkg/lock/lock.go` with file locking implementation
   - Use `syscall.Flock` (Unix) or equivalent
   - Handle signal cleanup (SIGINT, SIGTERM)
2. Add lock file path to config:
   ```yaml
   lock_file: /tmp/minecraft-map-build.lock  # default
   ```
3. Integrate into `pkg/maps/builder.go`:
   - Acquire lock at start of `Build()`
   - Release lock on completion/error
   - Add `--lock-timeout` flag for timeout behavior

**Commands Affected**:
- `minecraftctl map build` - Add locking
- Add `--lock-timeout` flag (default: block indefinitely)
- Add `--no-lock` flag for testing/bypass

**API Changes**:
```go
type BuildOptions struct {
    WorldName  string
    MapName    string
    Force      bool
    Clean      bool
    LockFile   string          // NEW
    LockTimeout time.Duration  // NEW (0 = block forever)
    NoLock     bool            // NEW
}
```

**Files to Create/Modify**:
- `pkg/lock/lock.go` (new)
- `pkg/lock/lock_test.go` (new)
- `pkg/maps/build.go` (modify)
- `pkg/config/config.go` (add lock_file config)
- `cmd/minecraftctl/map.go` (add flags)

---

### Enhancement 2: Glob/Wildcard Support

**Goal**: Support processing multiple worlds via glob patterns.

**Requirements**:
- Support `world*` style patterns in world name
- Expand patterns to matching world directories
- Process each match sequentially (or in parallel as future enhancement)
- Maintain current behavior when no glob characters present

**Design**:
```go
// In pkg/worlds/worlds.go
func ExpandWorldPattern(pattern string) ([]string, error) {
    // Check if pattern contains glob characters
    // Use filepath.Glob or filepath.Match
    // Return list of matching world names
}
```

**Implementation Steps**:
1. Add `ExpandWorldPattern()` function to `pkg/worlds/worlds.go`
   - Detect glob characters (`*`, `?`, `[`)
   - Use `filepath.Glob` for expansion
   - Return list of world names (basenames of matching directories)
2. Update `map build` command:
   - Check if world name contains glob characters
   - Expand to list of worlds
   - Process each world sequentially
3. Add `--parallel` flag (future enhancement) for concurrent processing

**Commands Affected**:
- `minecraftctl map build <pattern>` - Support globs
- `minecraftctl map manifest <pattern>` - Support globs
- Example: `minecraftctl map build world*` processes all worlds starting with "world"

**API Changes**:
```go
// In cmd/minecraftctl/map.go
mapBuildCmd.Flags().Bool("parallel", false, "Process multiple worlds in parallel")
```

**Files to Create/Modify**:
- `pkg/worlds/worlds.go` (add ExpandWorldPattern)
- `pkg/worlds/worlds_test.go` (add tests)
- `cmd/minecraftctl/map.go` (use pattern expansion)

**Example Usage**:
```bash
# Build maps for all worlds matching pattern
minecraftctl map build world*

# Build maps for all worlds
minecraftctl map build "*"

# Build specific world (no change)
minecraftctl map build world1
```

---

### Enhancement 3: Preview Generation in Manifest Step

**Goal**: Automatically generate preview images when building manifests.

**Requirements**:
- Generate previews for each map during manifest creation
- Make it optional (add `--no-preview` flag)
- Maintain backward compatibility
- Use existing `GeneratePreview()` functionality

**Design**:
Modify `ManifestBuilder.BuildManifests()` to:
1. Build per-map manifests
2. Generate preview for each map (unless `--no-preview`)
3. Build world-level manifest

**Implementation Steps**:
1. Add `GeneratePreviews` option to `ManifestBuilder`:
   ```go
   type ManifestBuilder struct {
       worldsDir       string
       mapsDir         string
       generatePreviews bool  // NEW
   }
   ```
2. Modify `BuildManifests()`:
   - After building map manifest, call `GeneratePreview()` if enabled
   - Handle errors gracefully (log warning, continue)
3. Add `--no-preview` flag to `map manifest` command
4. Add `--preview-only` flag to generate only previews (useful for testing)

**API Changes**:
```go
type ManifestOptions struct {
    WorldName       string
    GeneratePreviews bool  // NEW (default: true)
    PreviewOnly     bool   // NEW (default: false)
}

func NewManifestBuilder() *ManifestBuilder {
    // ...
    return &ManifestBuilder{
        generatePreviews: true,  // Default to generating previews
    }
}
```

**Commands Affected**:
- `minecraftctl map manifest <world>` - Generate previews by default
- Add `--no-preview` flag
- Add `--preview-only` flag (alternative command)

**Files to Create/Modify**:
- `pkg/maps/manifest_builder.go` (modify)
- `cmd/minecraftctl/map.go` (add flags, update command)

**Example Usage**:
```bash
# Generate manifests with previews (default)
minecraftctl map manifest world1

# Generate manifests without previews
minecraftctl map manifest world1 --no-preview

# Generate only previews (no manifest updates)
minecraftctl map manifest world1 --preview-only
```

---

### Enhancement 4: Non-Blocking Lock Mode

**Goal**: Allow graceful handling when lock cannot be acquired.

**Requirements**:
- Support non-blocking lock acquisition
- Exit gracefully with appropriate message when lock is held
- Use with timeout for flexible behavior
- Return appropriate exit codes

**Design**:
Integrate with Enhancement 1 (Lock File Management):
- `--non-blocking` flag: Try lock, exit immediately if held
- `--lock-timeout <duration>`: Wait up to duration, then exit
- Default: Block indefinitely (current behavior)

**Implementation Steps**:
1. Extend `pkg/lock/lock.go`:
   ```go
   type LockOptions struct {
       Timeout      time.Duration  // 0 = block forever
       NonBlocking  bool           // Try once and fail
   }
   
   func (fl *FileLock) TryLockWithOptions(opts LockOptions) error
   ```
2. Update `BuildOptions`:
   ```go
   type BuildOptions struct {
       // ... existing fields
       NonBlocking bool  // NEW
   }
   ```
3. Update `cmd/minecraftctl/map.go`:
   - Add `--non-blocking` flag
   - Add `--lock-timeout <seconds>` flag
   - Handle lock acquisition failures appropriately

**Commands Affected**:
- `minecraftctl map build` - Add non-blocking options
- Exit code 0 when skipping due to lock (to match script behavior)

**Files to Create/Modify**:
- `pkg/lock/lock.go` (modify)
- `pkg/maps/build.go` (modify)
- `cmd/minecraftctl/map.go` (add flags)

**Example Usage**:
```bash
# Non-blocking: exit immediately if lock held
minecraftctl map build world1 --non-blocking

# Timeout: wait up to 60 seconds, then exit
minecraftctl map build world1 --lock-timeout 60s

# Default: block until lock available
minecraftctl map build world1
```

---

### Enhancement 5: Batch Operations

**Goal**: Support processing multiple worlds efficiently.

**Requirements**:
- Process multiple worlds in parallel or sequentially
- Aggregate results appropriately
- Support `--parallel` flag for concurrent execution
- Respect locking across parallel operations

**Design**:
```go
type BatchOptions struct {
    WorldPatterns []string
    Parallel      bool
    MaxWorkers    int  // Limit concurrent workers
}

func (b *Builder) BuildBatch(opts BatchOptions) error {
    // Expand all patterns
    // Process sequentially or in parallel
    // Aggregate errors
}
```

**Implementation Steps**:
1. Add batch processing to `pkg/maps/builder.go`:
   - Function to process multiple worlds
   - Use goroutines for parallel execution
   - Use worker pool pattern
2. Add `--parallel` and `--max-workers` flags
3. Update commands to support multiple world arguments:
   - `minecraftctl map build world1 world2 world3`
   - Or use glob: `minecraftctl map build world* --parallel`

**Commands Affected**:
- `minecraftctl map build` - Support multiple worlds/parallel
- `minecraftctl map manifest` - Support multiple worlds/parallel

**Files to Create/Modify**:
- `pkg/maps/build.go` (add batch processing)
- `cmd/minecraftctl/map.go` (accept multiple args, add flags)

**Example Usage**:
```bash
# Process multiple worlds sequentially
minecraftctl map build world1 world2 world3

# Process all matching worlds in parallel (max 4 workers)
minecraftctl map build world* --parallel --max-workers 4
```

---

## Implementation Priority

### Phase 1: Core Functionality (High Priority)
1. ✅ **Enhancement 1: Lock File Management**
   - Critical for preventing conflicts
   - Blocks other enhancements if not done first
   - Estimated: 2-3 days

2. ✅ **Enhancement 3: Preview in Manifest Step**
   - Simplifies usage significantly
   - Users won't need wrapper scripts
   - Estimated: 1-2 days

### Phase 2: Usability (Medium Priority)
3. ✅ **Enhancement 4: Non-Blocking Mode**
   - Depends on Enhancement 1
   - Important for systemd timer integration
   - Estimated: 1 day

4. ✅ **Enhancement 2: Glob Support**
   - Nice-to-have for batch operations
   - Can use multiple args as alternative
   - Estimated: 1-2 days

### Phase 3: Advanced Features (Low Priority)
5. ⚠️ **Enhancement 5: Batch Operations**
   - Performance optimization
   - Can be added later if needed
   - Estimated: 2-3 days

6. ⚠️ **Enhancement 6: Aggregate Manifest & HTML Index**
   - Generate `world_manifest.json` aggregating all world manifests
   - Generate `index.html` for web browsing
   - Can be added as separate command or flag
   - Estimated: 1-2 days

---

## Detailed Implementation Guide

### Step-by-Step: Enhancement 1 (Lock File Management)

#### 1.1 Create Lock Package
```go
// pkg/lock/lock.go
package lock

import (
    "fmt"
    "os"
    "syscall"
    "time"
)

type FileLock struct {
    path string
    file *os.File
}

func NewFileLock(path string) *FileLock {
    return &FileLock{path: path}
}

func (fl *FileLock) Lock() error {
    return fl.lockWithTimeout(0) // Block forever
}

func (fl *FileLock) TryLock(timeout time.Duration) error {
    return fl.lockWithTimeout(timeout)
}

func (fl *FileLock) lockWithTimeout(timeout time.Duration) error {
    // Open/create lock file
    file, err := os.OpenFile(fl.path, os.O_CREATE|os.O_RDWR, 0644)
    if err != nil {
        return fmt.Errorf("failed to open lock file: %w", err)
    }
    fl.file = file

    // Attempt to acquire exclusive lock
    start := time.Now()
    for {
        err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
        if err == nil {
            // Lock acquired
            return nil
        }

        if timeout > 0 && time.Since(start) >= timeout {
            file.Close()
            return fmt.Errorf("timeout waiting for lock: %w", err)
        }

        time.Sleep(100 * time.Millisecond)
    }
}

func (fl *FileLock) Unlock() error {
    if fl.file == nil {
        return nil
    }
    
    err := syscall.Flock(int(fl.file.Fd()), syscall.LOCK_UN)
    closeErr := fl.file.Close()
    fl.file = nil

    if err != nil {
        return fmt.Errorf("failed to unlock: %w", err)
    }
    return closeErr
}
```

#### 1.2 Update Config
```go
// pkg/config/config.go
type GlobalConfig struct {
    WorldsDir string
    MapsDir   string
    LockFile  string  // NEW
    Rcon      RconConfig
}

// In Init():
viper.SetDefault("lock_file", "/tmp/minecraft-map-build.lock")
globalConfig.LockFile = viper.GetString("lock_file")
```

#### 1.3 Integrate into Builder
```go
// pkg/maps/build.go
func (b *Builder) Build(opts BuildOptions) error {
    // Acquire lock if not disabled
    var fileLock *lock.FileLock
    if !opts.NoLock {
        lockPath := opts.LockFile
        if lockPath == "" {
            lockPath = config.Get().LockFile
        }
        
        fileLock = lock.NewFileLock(lockPath)
        timeout := opts.LockTimeout
        if opts.NonBlocking {
            timeout = 0 // Try immediately
        }
        
        if err := fileLock.TryLock(timeout); err != nil {
            if opts.NonBlocking {
                log.Info().Msg("Another map build is running, skipping (non-blocking)")
                return nil
            }
            return fmt.Errorf("failed to acquire lock: %w", err)
        }
        
        defer func() {
            if err := fileLock.Unlock(); err != nil {
                log.Error().Err(err).Msg("failed to release lock")
            }
        }()
    }

    // Rest of build logic...
}
```

#### 1.4 Add CLI Flags
```go
// cmd/minecraftctl/map.go
mapBuildCmd.Flags().String("lock-file", "", "Path to lock file")
mapBuildCmd.Flags().Duration("lock-timeout", 0, "Maximum time to wait for lock (0 = block forever)")
mapBuildCmd.Flags().Bool("no-lock", false, "Disable file locking")
mapBuildCmd.Flags().Bool("non-blocking", false, "Exit immediately if lock is held")
```

---

### Step-by-Step: Enhancement 3 (Preview in Manifest)

#### 3.1 Update ManifestBuilder
```go
// pkg/maps/manifest_builder.go
type ManifestBuilder struct {
    worldsDir        string
    mapsDir          string
    generatePreviews bool
    builder          *Builder  // For preview generation
}

func NewManifestBuilder() *ManifestBuilder {
    cfg := config.Get()
    return &ManifestBuilder{
        worldsDir:        cfg.WorldsDir,
        mapsDir:          cfg.MapsDir,
        generatePreviews: true,  // Default to true
        builder:          NewBuilder(),
    }
}

func (mb *ManifestBuilder) BuildManifests(worldName string, opts ManifestOptions) error {
    // ... existing manifest building code ...

    // Generate previews if enabled
    if opts.GeneratePreviews && !opts.PreviewOnly {
        for _, mapDef := range mapConfig.Maps {
            log.Info().Str("map", mapDef.Name).Msg("generating preview")
            if err := mb.builder.GeneratePreview(worldName, mapDef.Name); err != nil {
                log.Warn().Err(err).Str("map", mapDef.Name).Msg("failed to generate preview, continuing")
                // Don't fail entire operation on preview error
            }
        }
    }

    // Build world manifest...
}
```

#### 3.2 Update Map Command
```go
// cmd/minecraftctl/map.go
var mapManifestCmd = &cobra.Command{
    Use:   "manifest <world>",
    Short: "Build manifests for all maps in a world",
    Args:  cobra.ExactArgs(1),
    RunE: func(cmd *cobra.Command, args []string) error {
        worldName := args[0]
        noPreview, _ := cmd.Flags().GetBool("no-preview")
        previewOnly, _ := cmd.Flags().GetBool("preview-only")

        builder := maps.NewManifestBuilder()
        opts := maps.ManifestOptions{
            WorldName:       worldName,
            GeneratePreviews: !noPreview,
            PreviewOnly:     previewOnly,
        }
        return builder.BuildManifests(worldName, opts)
    },
}

mapManifestCmd.Flags().Bool("no-preview", false, "Skip preview generation")
mapManifestCmd.Flags().Bool("preview-only", false, "Only generate previews, skip manifest")
```

---

## Testing Strategy

### Unit Tests
- `pkg/lock/lock_test.go`: Test lock acquisition, timeout, cleanup
- `pkg/worlds/worlds_test.go`: Test glob pattern expansion
- `pkg/maps/manifest_builder_test.go`: Test preview generation in manifest step

### Integration Tests
- Test concurrent map builds (lock prevents conflicts)
- Test glob expansion with various patterns
- Test manifest generation with/without previews

### Manual Testing
- Test with systemd timers (non-blocking mode)
- Test lock cleanup on SIGTERM/SIGINT
- Test pattern matching in real environment

---

## Migration Path

After implementing enhancements:

1. **Update Migration Plan**: Remove need for wrapper scripts
2. **Update Systemd Services**: Use minecraftctl directly:
   ```ini
   ExecStart=/usr/local/bin/minecraftctl map build %i --non-blocking
   ExecStartPost=/usr/local/bin/minecraftctl map manifest %i
   ```
3. **Remove Wrapper Scripts**: Delete `rebuild-map.sh` and `build-map-manifests-wrapper.sh`
4. **Update Documentation**: Update README with new features

---

## Success Criteria

- ✅ No wrapper scripts needed for basic functionality
- ✅ Lock file prevents concurrent builds
- ✅ Preview generation happens automatically
- ✅ Glob patterns work as expected
- ✅ Non-blocking mode works with systemd timers
- ✅ Backward compatible (existing commands work unchanged)

---

## Estimated Timeline

- **Phase 1** (Core): 3-5 days
- **Phase 2** (Usability): 2-3 days  
- **Phase 3** (Advanced): 2-3 days (optional)
- **Total (Phases 1-2)**: ~1 week
- **Total (All Phases)**: ~2 weeks

---

## Dependencies

### New Go Packages (if needed)
- None required (all can use standard library)
- Consider `github.com/gofrs/flock` as alternative lock implementation (but standard library should suffice)

### External Tools
- No new dependencies

---

## Backward Compatibility

All enhancements maintain backward compatibility:
- Default behavior unchanged (locking blocks, previews generated)
- Flags are optional with sensible defaults
- Existing scripts/commands continue to work

---

## Open Questions

1. **Lock File Location**: Should it be configurable via env var or only config file?
   - **Decision**: Both (config file default, env var override)

2. **Preview Generation Errors**: Should manifest step fail if preview fails?
   - **Decision**: Log warning, continue (preview failure shouldn't block manifest)

3. **Parallel Processing**: Should we limit by default (e.g., max 4 workers)?
   - **Decision**: Yes, default to CPU count, allow override

4. **Lock Cleanup**: Should we remove lock file on cleanup, or leave it?
   - **Decision**: Leave file, just release lock (standard practice)

---

### Enhancement 6: Aggregate Manifest & HTML Index

**Goal**: Generate aggregate manifest file and HTML index page matching script behavior.

**Requirements**:
- Generate `/srv/minecraft-server/maps/world_manifest.json` - JSON array of all world manifests
- Generate `/srv/minecraft-server/maps/index.html` - Simple HTML listing of worlds and maps
- Should be called after manifest generation (or as separate command)
- Match existing script output format

**Design**:
```go
// Add to ManifestBuilder or new command
func (mb *ManifestBuilder) BuildAggregateIndex() error {
    // Collect all world manifest.json files
    // Aggregate into single JSON array
    // Generate HTML index
}
```

**Implementation Steps**:
1. Add `BuildAggregateIndex()` method to `ManifestBuilder`
2. Add new command `minecraftctl map index` or flag `--update-index` to manifest command
3. Generate `world_manifest.json` combining all world manifests
4. Generate `index.html` with simple HTML structure matching script output

**Commands Affected**:
- `minecraftctl map manifest --update-index` - Update index after manifest
- `minecraftctl map index` - Standalone index update command

**Files to Create/Modify**:
- `pkg/maps/manifest_builder.go` (add BuildAggregateIndex)
- `cmd/minecraftctl/map.go` (add index command/flag)

**Example Usage**:
```bash
# Update index after manifest generation
minecraftctl map manifest world1 --update-index

# Update index standalone
minecraftctl map index
```

---

## Related Documentation Updates

After implementation, update:
1. `minecraftctl/README.md` - Add new flags and features
2. `docs/minecraftctl-migration-plan.md` - Update with enhanced capabilities
3. `docs/map-build-config.rst` - If any config changes needed

