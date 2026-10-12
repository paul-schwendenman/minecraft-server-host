package maps

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/paul/minecraftctl/pkg/config"
	"github.com/paul/minecraftctl/pkg/lock"
	"github.com/rs/zerolog/log"
)

const (
	DefaultUnminedPath = "/opt/unmined/unmined-cli"
)

// runUnmined runs uNmINeD with args, streaming its output. A var so tests can
// stand in for the binary.
var runUnmined = func(unminedPath string, args []string) error {
	cmd := exec.Command(unminedPath, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

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
	RangesOnly  bool // Render every range and skip the base map
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

	// Default log level to warning if not specified
	if opts.LogLevel == "" {
		opts.LogLevel = "warning"
	}

	// Plan every map first, so --ranges-only on a format change fails before
	// anything renders
	plans, err := b.Plan(opts)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Join(b.mapsDir, opts.WorldName), 0755); err != nil {
		return fmt.Errorf("failed to create maps directory: %w", err)
	}

	for _, p := range plans {
		if err := b.runPlan(p, opts); err != nil {
			log.Error().Err(err).Str("map", p.Map).Msg("failed to build map")
			continue
		}
	}

	return nil
}

// buildMap plans and builds one map
func (b *Builder) buildMap(
	mapDef config.MapDefinition,
	defaults config.MapDefaults,
	worldDir string,
	worldMapsDir string,
	worldMtime int64,
	opts BuildOptions,
) error {
	p, err := planMap(mapDef, defaults, worldDir, worldMapsDir, worldMtime, opts)
	if err != nil {
		return err
	}
	return b.runPlan(p, opts)
}

// runPlan carries out one map's plan
func (b *Builder) runPlan(p MapPlan, opts BuildOptions) error {
	mapDef := p.mapDef
	defaults := p.defaults
	worldDir := p.worldDir
	mapOutput := p.mapOutput

	switch p.Action {
	case ActionSkip:
		log.Info().Str("map", p.Map).Str("reason", p.Reason).Msg("map is up to date, skipping")
		return nil
	case ActionFull:
		log.Info().Str("map", p.Map).Str("reason", p.Reason).Msg("rendering the whole map")
	case ActionRanges:
		log.Info().Str("map", p.Map).Str("reason", p.Reason).Strs("ranges", p.Ranges).
			Msg("rendering ranges only, skipping the base map")
	}

	// Verify dimension exists
	if _, ok := findRegionDir(worldDir, mapDef.Dimension); !ok {
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

	// The viewer's maxZoom has to cover every configured range, not just the
	// ones rendered this time
	maxZoomin := zoomin
	for _, r := range mapDef.Ranges {
		if r.Zoomin != nil && *r.Zoomin > maxZoomin {
			maxZoomin = *r.Zoomin
		}
	}

	if p.Action == ActionRanges {
		return b.renderRangesOnly(p, defaults, worldDir, zoomout, zoomin, maxZoomin, opts)
	}

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

	// uNmINeD is incremental: without --force it only re-renders regions it
	// considers changed, so tiles left by an earlier or partial render stay put.
	// A format change forces a full render: the viewer loads every tile with
	// the one format in its properties file, so unchanged regions would show
	// as holes.
	if p.Force {
		baseArgs = append(baseArgs, "--force")
	}

	// Add optional map options
	baseArgs = b.addMapOptions(baseArgs, mapDef.Options)

	// Build ranges FIRST so their properties don't overwrite the base map bounds
	// The base map render must come last to set the correct full-world bounds
	// in unmined.map.properties.js
	//
	// Each range render rewrites unmined.map.properties.js (with the new image
	// format and the range's bounds), so save it first and put it back if the
	// base render fails. Otherwise the next build would read the new format,
	// see no format change, and never convert the base map's tiles.
	propsPath := filepath.Join(mapOutput, "unmined.map.properties.js")
	savedProps, savedErr := os.ReadFile(propsPath)
	propsMode := os.FileMode(0644)
	if st, err := os.Stat(propsPath); err == nil {
		propsMode = st.Mode().Perm()
	}

	hashes := map[string]string{}
	rangeFailed := false
	for _, i := range p.rangeIdx {
		r := mapDef.Ranges[i]
		if err := b.buildRange(r, mapDef, defaults, worldDir, mapOutput, zoomout, zoomin, opts.LogLevel, p.rangeForced(i)); err != nil {
			log.Error().Err(err).Str("range", r.Name).Msg("failed to build range")
			rangeFailed = true
			continue
		}
		hashes[p.keys[i]] = p.hashes[i]
	}

	log.Info().Str("map", mapDef.Name).Str("dimension", mapDef.Dimension).
		Strs("args", baseArgs).Msg("rendering base map")

	// Build base map LAST so it sets the correct full-world bounds
	if err := runUnmined(b.unminedPath, baseArgs); err != nil {
		if savedErr == nil {
			if werr := os.WriteFile(propsPath, savedProps, propsMode); werr != nil {
				log.Error().Err(werr).Msg("failed to restore properties file after the base render failed")
			}
		}
		return fmt.Errorf("failed to render base map: %w", err)
	}

	// Patch maxZoom in properties file if ranges have higher zoom levels
	if maxZoomin > zoomin {
		if err := b.patchPropertiesMaxZoom(mapOutput, maxZoomin); err != nil {
			log.Warn().Err(err).Msg("failed to patch maxZoom in properties file")
		}
	}

	// Update manifest
	now := time.Now()
	m := Manifest{LastRendered: now.Format(time.RFC3339), LastRenderedEpoch: now.Unix(), RangeHashes: hashes}
	if err := b.writeBuildManifest(mapOutput, opts.WorldName, mapDef, m); err != nil {
		return err
	}

	if p.FormatChanged {
		b.removeOldFormatTiles(mapDef.Name, mapOutput, defaults.ImageFormat, rangeFailed)
	}
	return nil
}

// renderRangesOnly renders the plan's ranges without the base map. Each range
// render rewrites unmined.map.properties.js with that range's bounds, so the
// file is saved first and put back afterwards (also when a render fails),
// then maxZoom is patched again.
func (b *Builder) renderRangesOnly(p MapPlan, defaults config.MapDefaults, worldDir string, zoomout, zoomin, maxZoomin int, opts BuildOptions) error {
	mapDef := p.mapDef
	propsPath := filepath.Join(p.mapOutput, "unmined.map.properties.js")
	saved, err := os.ReadFile(propsPath)
	if err != nil {
		return fmt.Errorf("failed to save properties file before rendering ranges: %w", err)
	}
	propsMode := os.FileMode(0644)
	if st, err := os.Stat(propsPath); err == nil {
		propsMode = st.Mode().Perm()
	}

	// Start from the manifest's hashes, minus ranges no longer configured
	hashes := map[string]string{}
	configured := map[string]bool{}
	for _, k := range p.keys {
		configured[k] = true
	}
	if p.manifest != nil {
		for k, h := range p.manifest.RangeHashes {
			if configured[k] {
				hashes[k] = h
			}
		}
	}

	var failed []string
	func() {
		defer func() {
			if err := os.WriteFile(propsPath, saved, propsMode); err != nil {
				log.Error().Err(err).Msg("failed to restore properties file after rendering ranges")
				failed = append(failed, "restoring unmined.map.properties.js")
			}
		}()
		for _, i := range p.rangeIdx {
			r := mapDef.Ranges[i]
			if err := b.buildRange(r, mapDef, defaults, worldDir, p.mapOutput, zoomout, zoomin, opts.LogLevel, p.rangeForced(i)); err != nil {
				log.Error().Err(err).Str("range", r.Name).Msg("failed to build range")
				delete(hashes, p.keys[i])
				failed = append(failed, r.Name)
				continue
			}
			hashes[p.keys[i]] = p.hashes[i]
		}
	}()

	if maxZoomin > zoomin {
		if err := b.patchPropertiesMaxZoom(p.mapOutput, maxZoomin); err != nil {
			log.Warn().Err(err).Msg("failed to patch maxZoom in properties file")
		}
	}

	// The base map wasn't re-rendered, so keep the manifest's render time
	m := Manifest{RangeHashes: hashes}
	if p.manifest != nil {
		m.LastRendered = p.manifest.LastRendered
		m.LastRenderedEpoch = p.manifest.LastRenderedEpoch
	}
	if err := b.writeBuildManifest(p.mapOutput, opts.WorldName, mapDef, m); err != nil {
		return err
	}
	if len(failed) > 0 {
		return fmt.Errorf("failed to render: %s", strings.Join(failed, ", "))
	}
	return nil
}

// removeOldFormatTiles deletes the tiles left over from the previous image
// format once a map has been fully re-rendered in the new one. If a range
// failed, or the properties file doesn't name the new format, the old tiles
// are kept: the render is incomplete and `map clean` can tidy up later.
func (b *Builder) removeOldFormatTiles(mapName, mapOutput, format string, rangeFailed bool) {
	if rangeFailed {
		log.Warn().Str("map", mapName).Msg("a range failed to render, keeping old-format tiles")
		return
	}
	if got := renderedImageFormat(mapOutput); !strings.EqualFold(got, format) {
		log.Warn().Str("map", mapName).Str("rendered", got).Str("want", format).
			Msg("properties file doesn't show the new format, keeping old-format tiles")
		return
	}
	files, bytes, err := removeStaleTiles(mapOutput, format, false)
	if err != nil {
		log.Error().Err(err).Str("map", mapName).Msg("failed to remove old-format tiles")
		return
	}
	log.Info().Str("map", mapName).Int("files", files).Int64("bytes", bytes).
		Msg("removed old-format tiles")
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
	force bool,
) error {
	x1, z1, x2, z2, err := rangeBounds(r)
	if err != nil {
		return err
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

	if force {
		args = append(args, "--force")
	}

	// Apply map-specific options (gndxray, topY, bottomY, shadows, etc.)
	args = b.addMapOptions(args, mapDef.Options)

	log.Info().Str("range", r.Name).Strs("args", args).Msg("rendering range")

	return runUnmined(b.unminedPath, args)
}

// regionSize is the width of a Minecraft region in blocks. uNmINeD tiles are
// aligned to region boundaries at every zoom level.
const regionSize = 512

// rangeBounds returns the block bounds of a range, snapped outward to region
// boundaries. An --area that cuts through a tile leaves the rest of that tile
// blank, and because tiles from a range render are shared with the base render
// (and feed the zoomed-out levels), the blank part shows up in the final map as
// a white frame around the range.
func rangeBounds(r config.MapRange) (x1, z1, x2, z2 int, err error) {
	x1 = floorTo(r.Center[0]-r.Radius, regionSize)
	z1 = floorTo(r.Center[1]-r.Radius, regionSize)
	x2 = ceilTo(r.Center[0]+r.Radius, regionSize)
	z2 = ceilTo(r.Center[1]+r.Radius, regionSize)

	if r.Radius <= 0 {
		return 0, 0, 0, 0, fmt.Errorf("invalid range bounds")
	}
	return x1, z1, x2, z2, nil
}

func floorTo(v, m int) int {
	q := v / m
	if v%m != 0 && v < 0 {
		q--
	}
	return q * m
}

func ceilTo(v, m int) int {
	return -floorTo(-v, m)
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
		if shadowArg := config.ShadowsToString(opts.Shadows); shadowArg != "" {
			args = append(args, "--shadows", shadowArg)
		}
	}
	return args
}

var imageFormatRe = regexp.MustCompile(`imageFormat:\s*"([^"]*)"`)

// renderedImageFormat returns the image format a map was last rendered with,
// from uNmINeD's unmined.map.properties.js, or "" if it hasn't been rendered.
func renderedImageFormat(mapOutput string) string {
	data, err := os.ReadFile(filepath.Join(mapOutput, "unmined.map.properties.js"))
	if err != nil {
		return ""
	}
	m := imageFormatRe.FindSubmatch(data)
	if m == nil {
		return ""
	}
	return string(m[1])
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
