package maps

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"

	"github.com/paul/minecraftctl/pkg/config"
	"github.com/paul/minecraftctl/pkg/nbt"
	"github.com/rs/zerolog/log"
)

// GeneratePreview generates a preview image for a map
func (b *Builder) GeneratePreview(worldName, mapName, logLevel string) error {
	if logLevel == "" {
		logLevel = "warning"
	}
	worldPath := filepath.Join(b.worldsDir, worldName)
	mapConfig, err := config.LoadMapConfig(worldPath)
	if err != nil {
		return fmt.Errorf("failed to load map config: %w", err)
	}

	// Find the map definition
	var mapDef *config.MapDefinition
	for i := range mapConfig.Maps {
		if mapConfig.Maps[i].Name == mapName {
			mapDef = &mapConfig.Maps[i]
			break
		}
	}
	if mapDef == nil {
		return fmt.Errorf("map '%s' not found in map-config.yml", mapName)
	}

	worldDir := filepath.Join(worldPath, "world")
	levelDatPath := filepath.Join(worldDir, "level.dat")

	// Read spawn coordinates
	levelInfo, err := nbt.ReadLevelDat(levelDatPath)
	if err != nil {
		return fmt.Errorf("failed to read level.dat: %w", err)
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

	// Calculate preview center based on dimension
	var centerX, centerZ int
	switch mapDef.Dimension {
	case "overworld":
		// For overworld, use spawn coordinates
		centerX = int(levelInfo.GetSpawnX())
		centerZ = int(levelInfo.GetSpawnZ())
	case "end", "1":
		// The End's main island is always centered at 0,0
		centerX = 0
		centerZ = 0
	default:
		// For nether, calculate center from region files
		cx, cz, err := calculateRegionCenter(regionDir)
		if err != nil {
			return fmt.Errorf("failed to calculate region center: %w", err)
		}
		centerX = cx
		centerZ = cz
	}

	// Calculate preview area (64 blocks around center)
	const previewRange = 64
	x1 := centerX - previewRange
	z1 := centerZ - previewRange
	x2 := centerX + previewRange
	z2 := centerZ + previewRange

	area := fmt.Sprintf("b((%d,%d),(%d,%d))", x1, z1, x2, z2)

	outputSubdir := mapDef.OutputSubdir
	if outputSubdir == "" {
		outputSubdir = mapDef.Name
	}
	worldMapsDir := filepath.Join(b.mapsDir, worldName)
	mapOutput := filepath.Join(worldMapsDir, outputSubdir)
	previewPath := filepath.Join(mapOutput, "preview.png")

	if err := os.MkdirAll(mapOutput, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	log.Info().
		Str("map", mapName).
		Str("dimension", mapDef.Dimension).
		Int("center_x", centerX).
		Int("center_z", centerZ).
		Msg("generating preview")

	args := []string{
		"image", "render",
		"--world", worldDir,
		"--dimension", mapDef.Dimension,
		"--area", area,
		"--zoom", "2",
		"--log-level", logLevel,
		"--output", previewPath,
	}

	// Apply map-specific options (gndxray, topY, bottomY, shadows, etc.)
	args = b.addMapOptions(args, mapDef.Options)

	cmd := exec.Command(b.unminedPath, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to render preview: %w", err)
	}

	log.Info().Str("preview", previewPath).Msg("preview generated")
	return nil
}

// calculateRegionCenter calculates the center block coordinates from region files.
// Region files are named r.X.Z.mca where X and Z are region coordinates.
// Each region is 512x512 blocks (32x32 chunks, each chunk is 16x16 blocks).
func calculateRegionCenter(regionDir string) (int, int, error) {
	entries, err := os.ReadDir(regionDir)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to read region directory: %w", err)
	}

	// Match region file names: r.X.Z.mca
	regionRegex := regexp.MustCompile(`^r\.(-?\d+)\.(-?\d+)\.mca$`)

	var sumX, sumZ, count int
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		matches := regionRegex.FindStringSubmatch(entry.Name())
		if matches == nil {
			continue
		}

		regionX, _ := strconv.Atoi(matches[1])
		regionZ, _ := strconv.Atoi(matches[2])

		// Convert region coords to block coords (center of region)
		// Region (0,0) covers blocks (0,0) to (511,511)
		// Center of region is at (regionX * 512 + 256, regionZ * 512 + 256)
		sumX += regionX*512 + 256
		sumZ += regionZ*512 + 256
		count++
	}

	if count == 0 {
		return 0, 0, fmt.Errorf("no region files found")
	}

	// Return average center
	return sumX / count, sumZ / count, nil
}
