package maps

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Manifest represents a map manifest.json file
type Manifest struct {
	World             string `json:"world"`
	Map               string `json:"map"`
	Dimension         string `json:"dimension"`
	Path              string `json:"path,omitempty"`
	LastRendered      string `json:"last_rendered"`
	LastRenderedEpoch int64  `json:"last_rendered_epoch"`
}

// readManifest reads a manifest.json file
func readManifest(path string) (*Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var manifest Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, err
	}

	return &manifest, nil
}

// updateManifest updates or creates a manifest.json file
func (b *Builder) updateManifest(mapOutput, worldName, mapName, dimension string) error {
	manifestPath := filepath.Join(mapOutput, "manifest.json")

	now := time.Now()
	manifest := Manifest{
		World:             worldName,
		Map:               mapName,
		Dimension:         dimension,
		LastRendered:      now.Format(time.RFC3339),
		LastRenderedEpoch: now.Unix(),
	}

	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal manifest: %w", err)
	}

	if err := os.WriteFile(manifestPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write manifest: %w", err)
	}

	return nil
}

