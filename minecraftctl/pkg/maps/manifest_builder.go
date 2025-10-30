package maps

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/paul/minecraftctl/pkg/config"
	"github.com/paul/minecraftctl/pkg/nbt"
	"github.com/rs/zerolog/log"
)

// ManifestBuilder builds manifests for maps and worlds
type ManifestBuilder struct {
	worldsDir        string
	mapsDir          string
	generatePreviews bool
	builder          *Builder // For preview generation
}

// ManifestOptions control manifest building behavior
type ManifestOptions struct {
	WorldName        string
	GeneratePreviews bool // Generate previews during manifest creation
	PreviewOnly      bool // Only generate previews, skip manifest
}

// NewManifestBuilder creates a new manifest builder
func NewManifestBuilder() *ManifestBuilder {
	cfg := config.Get()
	return &ManifestBuilder{
		worldsDir:        cfg.WorldsDir,
		mapsDir:          cfg.MapsDir,
		generatePreviews: true, // Default to generating previews
		builder:          NewBuilder(),
	}
}

// WorldManifest represents a world-level manifest
type WorldManifest struct {
	World          string                   `json:"world"`
	Version        string                   `json:"version"`
	Difficulty     int32                    `json:"difficulty"`
	DifficultyName string                   `json:"difficulty_name"`
	LastPlayed     string                   `json:"last_played"`
	Maps           []map[string]string      `json:"maps"`
	LastRendered   string                   `json:"last_rendered"`
}

// BuildManifests builds manifests for all maps in a world
func (mb *ManifestBuilder) BuildManifests(worldName string, opts ManifestOptions) error {
	if opts.WorldName != "" {
		worldName = opts.WorldName
	}

	worldPath := filepath.Join(mb.worldsDir, worldName)
	mapConfig, err := config.LoadMapConfig(worldPath)
	if err != nil {
		return fmt.Errorf("failed to load map config: %w", err)
	}

	worldDir := filepath.Join(worldPath, "world")
	worldMapsDir := filepath.Join(mb.mapsDir, worldName)
	mapList := []map[string]string{}

	// Build manifest for each map
	for _, mapDef := range mapConfig.Maps {
		// Generate preview if requested
		if (opts.GeneratePreviews || opts.PreviewOnly) && !opts.PreviewOnly {
			log.Info().Str("map", mapDef.Name).Msg("generating preview")
			if err := mb.builder.GeneratePreview(worldName, mapDef.Name); err != nil {
				log.Warn().Err(err).Str("map", mapDef.Name).Msg("failed to generate preview, continuing")
				// Don't fail entire operation on preview error
			}
		} else if opts.PreviewOnly {
			// Preview-only mode: generate preview without manifest
			log.Info().Str("map", mapDef.Name).Msg("generating preview only")
			if err := mb.builder.GeneratePreview(worldName, mapDef.Name); err != nil {
				log.Warn().Err(err).Str("map", mapDef.Name).Msg("failed to generate preview")
				// Continue with other maps
			}
			continue
		}

		// Skip manifest generation if preview-only
		if opts.PreviewOnly {
			continue
		}

		outputSubdir := mapDef.OutputSubdir
		if outputSubdir == "" {
			outputSubdir = mapDef.Name
		}
		mapOutput := filepath.Join(worldMapsDir, outputSubdir)

		manifest := Manifest{
			World:        worldName,
			Map:          mapDef.Name,
			Dimension:    mapDef.Dimension,
			Path:         fmt.Sprintf("%s/%s", worldName, outputSubdir),
			LastRendered: time.Now().Format(time.RFC3339),
		}

		manifestPath := filepath.Join(mapOutput, "manifest.json")
		if err := mb.writeManifest(manifestPath, manifest); err != nil {
			log.Error().Err(err).Str("map", mapDef.Name).Msg("failed to write map manifest")
			continue
		}

		mapList = append(mapList, map[string]string{
			"name":      mapDef.Name,
			"dimension": mapDef.Dimension,
		})
	}

	// Skip world manifest if preview-only
	if opts.PreviewOnly {
		return nil
	}

	// Build world-level manifest
	levelDatPath := filepath.Join(worldDir, "level.dat")
	levelInfo, err := nbt.ReadLevelDat(levelDatPath)
	if err != nil {
		return fmt.Errorf("failed to read level.dat: %w", err)
	}

	difficulty := levelInfo.Difficulty
	difficultyName := difficultyName(difficulty)

	var lastPlayed string
	if levelInfo.LastPlayed > 0 {
		lastPlayed = time.Unix(levelInfo.LastPlayed/1000, 0).UTC().Format(time.RFC3339)
	}

	worldManifest := WorldManifest{
		World:          worldName,
		Version:        levelInfo.Version.Name,
		Difficulty:     difficulty,
		DifficultyName: difficultyName,
		LastPlayed:     lastPlayed,
		Maps:           mapList,
		LastRendered:   time.Now().Format(time.RFC3339),
	}

	worldManifestPath := filepath.Join(worldMapsDir, "manifest.json")
	if err := mb.writeManifest(worldManifestPath, worldManifest); err != nil {
		return fmt.Errorf("failed to write world manifest: %w", err)
	}

	log.Info().Str("world", worldName).Str("path", worldManifestPath).Msg("world manifest created")
	return nil
}

func (mb *ManifestBuilder) writeManifest(path string, data interface{}) error {
	fileData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal manifest: %w", err)
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	if err := os.WriteFile(path, fileData, 0644); err != nil {
		return fmt.Errorf("failed to write manifest: %w", err)
	}

	return nil
}

func difficultyName(diff int32) string {
	switch diff {
	case 0:
		return "Peaceful"
	case 1:
		return "Easy"
	case 2:
		return "Normal"
	case 3:
		return "Hard"
	default:
		return "Unknown"
	}
}

