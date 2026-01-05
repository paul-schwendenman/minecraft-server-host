package maps

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"time"

	"github.com/paul/minecraftctl/pkg/config"
	"github.com/paul/minecraftctl/pkg/lock"
	"github.com/rs/zerolog/log"
)

const (
	DefaultUnminedPath = "/opt/unmined/unmined-cli"
)

// Builder handles building maps using uNmINeD
type Builder struct {
	unminedPath string
	worldsDir   string
	mapsDir     string
}

// NewBuilder creates a new map builder
func NewBuilder() *Builder {
	cfg := config.Get()
	return &Builder{
		unminedPath: DefaultUnminedPath,
		worldsDir:   cfg.WorldsDir,
		mapsDir:     cfg.MapsDir,
	}
}

// BuildOptions control map building behavior
type BuildOptions struct {
	WorldName   string
	MapName     string // If empty, build all maps
	Force       bool
	Clean       bool
	LockFile    string        // Lock file path (empty = use config default)
	LockTimeout time.Duration // Lock timeout (0 = block forever)
	NoLock      bool          // Disable file locking
	NonBlocking bool          // Exit immediately if lock is held
	LogLevel    string        // unmined log level (verbose, debug, information, warning, error, fatal)
}

// Build builds maps for a world according to its map-config.yml
func (b *Builder) Build(opts BuildOptions) error {
	// Acquire lock if not disabled
	var fileLock *lock.FileLock
	if !opts.NoLock {
		lockPath := opts.LockFile
		if lockPath == "" {
			lockPath = config.Get().LockFile
		}

		fileLock = lock.NewFileLock(lockPath)
		lockOpts := lock.LockOptions{
			Timeout:     opts.LockTimeout,
			NonBlocking: opts.NonBlocking,
		}

		if err := fileLock.LockWithOptions(lockOpts); err != nil {
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

	worldPath := filepath.Join(b.worldsDir, opts.WorldName)
	mapConfig, err := config.LoadMapConfig(worldPath)
	if err != nil {
		return fmt.Errorf("failed to load map config: %w", err)
	}

	worldDir := filepath.Join(worldPath, "world")
	levelDatPath := filepath.Join(worldDir, "level.dat")

	// Get world mtime for up-to-date checks
	worldStat, err := os.Stat(levelDatPath)
	if err != nil {
		return fmt.Errorf("failed to stat level.dat: %w", err)
	}
	worldMtime := worldStat.ModTime().Unix()

	worldMapsDir := filepath.Join(b.mapsDir, opts.WorldName)
	if err := os.MkdirAll(worldMapsDir, 0755); err != nil {
		return fmt.Errorf("failed to create maps directory: %w", err)
	}

	// Default log level to warning if not specified
	if opts.LogLevel == "" {
		opts.LogLevel = "warning"
	}

	// Build each map
	for _, mapDef := range mapConfig.Maps {
		if opts.MapName != "" && mapDef.Name != opts.MapName {
			continue
		}

		if err := b.buildMap(mapDef, mapConfig.Defaults, worldDir, worldMapsDir, worldMtime, opts); err != nil {
			log.Error().Err(err).Str("map", mapDef.Name).Msg("failed to build map")
			continue
		}
	}

	return nil
}

func (b *Builder) buildMap(
	mapDef config.MapDefinition,
	defaults config.MapDefaults,
	worldDir string,
	worldMapsDir string,
	worldMtime int64,
	opts BuildOptions,
) error {
	// Check if rebuild is needed
	if !opts.Force {
		outputSubdir := mapDef.OutputSubdir
		if outputSubdir == "" {
			outputSubdir = mapDef.Name
		}
		mapOutput := filepath.Join(worldMapsDir, outputSubdir)
		manifestPath := filepath.Join(mapOutput, "manifest.json")

		if manifest, err := readManifest(manifestPath); err == nil {
			if manifest.LastRenderedEpoch >= worldMtime {
				log.Info().Str("map", mapDef.Name).Msg("map is up to date, skipping")
				return nil
			}
		}
	}

	// Verify dimension exists
	dimDir := worldDir
	switch mapDef.Dimension {
	case "nether", "-1":
		dimDir = filepath.Join(worldDir, "DIM-1")
	case "end", "1":
		dimDir = filepath.Join(worldDir, "DIM1")
	}

	regionDir := filepath.Join(dimDir, "region")
	if _, err := os.Stat(regionDir); os.IsNotExist(err) {
		return fmt.Errorf("dimension %s has no region data", mapDef.Dimension)
	}

	// Determine zoom levels
	zoomout := defaults.Zoomout
	if mapDef.Zoomout != nil {
		zoomout = *mapDef.Zoomout
	}
	zoomin := defaults.Zoomin
	if mapDef.Zoomin != nil {
		zoomin = *mapDef.Zoomin
	}

	// Build base command
	outputSubdir := mapDef.OutputSubdir
	if outputSubdir == "" {
		outputSubdir = mapDef.Name
	}
	mapOutput := filepath.Join(worldMapsDir, outputSubdir)

	baseArgs := []string{
		"web", "render",
		"--world", worldDir,
		"--dimension", mapDef.Dimension,
		"--output", mapOutput,
		"--imageformat", defaults.ImageFormat,
		"--chunkprocessors", strconv.Itoa(defaults.ChunkProcessors),
		"--log-level", opts.LogLevel,
		"--zoomout", strconv.Itoa(zoomout),
		"--zoomin", strconv.Itoa(zoomin),
	}

	// Add optional map options
	baseArgs = b.addMapOptions(baseArgs, mapDef.Options)

	// Build ranges FIRST so their properties don't overwrite the base map bounds
	// The base map render must come last to set the correct full-world bounds
	// in unmined.map.properties.js
	//
	// Track max zoomin across all ranges so we can patch the properties file
	maxZoomin := zoomin
	for _, r := range mapDef.Ranges {
		rangeZoomin := zoomin
		if r.Zoomin != nil {
			rangeZoomin = *r.Zoomin
		}
		if rangeZoomin > maxZoomin {
			maxZoomin = rangeZoomin
		}
		if err := b.buildRange(r, mapDef, defaults, worldDir, mapOutput, zoomout, zoomin, opts.LogLevel); err != nil {
			log.Error().Err(err).Str("range", r.Name).Msg("failed to build range")
			continue
		}
	}

	log.Info().Str("map", mapDef.Name).Str("dimension", mapDef.Dimension).
		Strs("args", baseArgs).Msg("rendering base map")

	// Build base map LAST so it sets the correct full-world bounds
	cmd := exec.Command(b.unminedPath, baseArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to render base map: %w", err)
	}

	// Patch maxZoom in properties file if ranges have higher zoom levels
	if maxZoomin > zoomin {
		if err := b.patchPropertiesMaxZoom(mapOutput, maxZoomin); err != nil {
			log.Warn().Err(err).Msg("failed to patch maxZoom in properties file")
		}
	}

	// Update manifest
	return b.updateManifest(mapOutput, opts.WorldName, mapDef.Name, mapDef.Dimension)
}

func (b *Builder) buildRange(
	r config.MapRange,
	mapDef config.MapDefinition,
	defaults config.MapDefaults,
	worldDir string,
	mapOutput string,
	defaultZoomout int,
	defaultZoomin int,
	logLevel string,
) error {
	// Calculate area bounds
	x1 := r.Center[0] - r.Radius
	z1 := r.Center[1] - r.Radius
	x2 := r.Center[0] + r.Radius
	z2 := r.Center[1] + r.Radius

	if x1 >= x2 || z1 >= z2 {
		return fmt.Errorf("invalid range bounds")
	}

	areaArg := fmt.Sprintf("--area=b((%d,%d),(%d,%d))", x1, z1, x2, z2)

	zoomout := defaultZoomout
	if r.Zoomout != nil {
		zoomout = *r.Zoomout
	}
	zoomin := defaultZoomin
	if r.Zoomin != nil {
		zoomin = *r.Zoomin
	}

	args := []string{
		"web", "render",
		"--world", worldDir,
		"--dimension", mapDef.Dimension,
		"--output", mapOutput,
		"--imageformat", defaults.ImageFormat,
		"--chunkprocessors", strconv.Itoa(defaults.ChunkProcessors),
		"--log-level", logLevel,
		"--zoomout", strconv.Itoa(zoomout),
		"--zoomin", strconv.Itoa(zoomin),
		areaArg,
	}

	// Apply map-specific options (gndxray, topY, bottomY, shadows, etc.)
	args = b.addMapOptions(args, mapDef.Options)

	log.Info().Str("range", r.Name).Strs("args", args).Msg("rendering range")

	cmd := exec.Command(b.unminedPath, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (b *Builder) addMapOptions(args []string, opts config.MapOptions) []string {
	if opts.TopY != nil {
		args = append(args, "--topY", strconv.Itoa(*opts.TopY))
	}
	if opts.BottomY != nil {
		args = append(args, "--bottomY", strconv.Itoa(*opts.BottomY))
	}
	if opts.GndXray != nil && *opts.GndXray {
		args = append(args, "--gndxray", "true")
	}
	if opts.Night != nil && *opts.Night {
		args = append(args, "--night", "true")
	}
	if opts.Shadows != nil {
		var shadowArg string
		switch v := opts.Shadows.(type) {
		case bool:
			if v {
				shadowArg = "true"
			} else {
				shadowArg = "false"
			}
		case string:
			shadowArg = v
		}
		if shadowArg != "" {
			args = append(args, "--shadows", shadowArg)
		}
	}
	return args
}

// patchPropertiesMaxZoom updates the maxZoom value in unmined.map.properties.js
// This is needed because ranges may have higher zoom levels than the base map,
// but the base map render (which runs last) overwrites the properties file.
func (b *Builder) patchPropertiesMaxZoom(mapOutput string, maxZoom int) error {
	propsPath := filepath.Join(mapOutput, "unmined.map.properties.js")

	data, err := os.ReadFile(propsPath)
	if err != nil {
		return fmt.Errorf("failed to read properties file: %w", err)
	}

	// Replace maxZoom value
	re := regexp.MustCompile(`maxZoom:\s*\d+`)
	newData := re.ReplaceAll(data, []byte(fmt.Sprintf("maxZoom: %d", maxZoom)))

	if err := os.WriteFile(propsPath, newData, 0644); err != nil {
		return fmt.Errorf("failed to write properties file: %w", err)
	}

	log.Info().Int("maxZoom", maxZoom).Msg("patched maxZoom in properties file")
	return nil
}
