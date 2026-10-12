package maps

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/paul/minecraftctl/pkg/config"
)

// Manifest represents a map manifest.json file
type Manifest struct {
	World             string `json:"world"`
	Map               string `json:"map"`
	Dimension         string `json:"dimension"`
	Path              string `json:"path,omitempty"`
	LastRendered      string `json:"last_rendered"`
	LastRenderedEpoch int64  `json:"last_rendered_epoch"`
	// RangeHashes maps each range's name to a hash of its render settings
	// (see rangeHash), so a build on an unchanged world renders only new or
	// changed ranges. Only ranges that rendered successfully are listed.
	RangeHashes map[string]string `json:"range_hashes,omitempty"`
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

// writeBuildManifest writes a map's manifest.json after a build. m carries
// the render time and range hashes; the rest comes from the map definition.
func (b *Builder) writeBuildManifest(mapOutput, worldName string, mapDef config.MapDefinition, m Manifest) error {
	manifestPath := filepath.Join(mapOutput, "manifest.json")

	m.World = worldName
	m.Map = mapDef.Name
	m.Dimension = mapDef.Dimension

	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal manifest: %w", err)
	}

	if err := os.WriteFile(manifestPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write manifest: %w", err)
	}

	return nil
}
