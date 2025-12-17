package maps

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

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

	// Calculate preview area (64 blocks around spawn)
	const previewRange = 64
	x1 := int(levelInfo.SpawnX) - previewRange
	z1 := int(levelInfo.SpawnZ) - previewRange
	x2 := int(levelInfo.SpawnX) + previewRange
	z2 := int(levelInfo.SpawnZ) + previewRange

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
		Int32("spawn_x", levelInfo.SpawnX).
		Int32("spawn_z", levelInfo.SpawnZ).
		Msg("generating preview")

	args := []string{
		"image", "render",
		"--world", worldDir,
		"--dimension", mapDef.Dimension,
		"--area", area,
		"--zoom", "2",
		"--log-level", logLevel,
		"--trim",
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
