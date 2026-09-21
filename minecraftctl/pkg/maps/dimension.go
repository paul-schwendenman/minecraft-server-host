package maps

import (
	"os"
	"path/filepath"
)

// dimensionNames maps a map-config dimension to its folder name in the
// namespaced world layout (world/dimensions/minecraft/<name>).
var dimensionNames = map[string]string{
	"overworld": "overworld",
	"0":         "overworld",
	"nether":    "the_nether",
	"-1":        "the_nether",
	"end":       "the_end",
	"1":         "the_end",
}

// legacyDimensionDir returns the pre-migration dimension folder inside a world.
func legacyDimensionDir(worldDir, dimension string) string {
	switch dimension {
	case "nether", "-1":
		return filepath.Join(worldDir, "DIM-1")
	case "end", "1":
		return filepath.Join(worldDir, "DIM1")
	}
	return worldDir
}

// findRegionDir returns the region directory for a dimension. Newer Minecraft
// versions store dimensions under world/dimensions/minecraft/<name>/ and older
// ones use world/region, world/DIM-1 and world/DIM1; the namespaced layout wins
// when both exist. ok is false when neither has a region directory, in which
// case the returned path is the legacy location.
func findRegionDir(worldDir, dimension string) (path string, ok bool) {
	if name, known := dimensionNames[dimension]; known {
		p := filepath.Join(worldDir, "dimensions", "minecraft", name, "region")
		if info, err := os.Stat(p); err == nil && info.IsDir() {
			return p, true
		}
	}

	p := filepath.Join(legacyDimensionDir(worldDir, dimension), "region")
	if info, err := os.Stat(p); err == nil && info.IsDir() {
		return p, true
	}
	return p, false
}
