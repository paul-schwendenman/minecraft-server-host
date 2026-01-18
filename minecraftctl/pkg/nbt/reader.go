package nbt

import (
	"compress/gzip"
	"fmt"
	"os"

	nbtlib "github.com/Tnze/go-mc/nbt"
)

// ReadLevelDat reads basic information from a Minecraft world's level.dat file
// level.dat files are gzip-compressed NBT files
func ReadLevelDat(levelDatPath string) (*LevelInfo, error) {
	file, err := os.Open(levelDatPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open level.dat: %w", err)
	}
	defer file.Close()

	// level.dat files are gzip-compressed
	gzReader, err := gzip.NewReader(file)
	if err != nil {
		return nil, fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzReader.Close()

	var data LevelData
	if _, err := nbtlib.NewDecoder(gzReader).Decode(&data); err != nil {
		return nil, fmt.Errorf("failed to decode NBT: %w", err)
	}

	return &data.Data, nil
}

// LevelInfo represents the Data tag within level.dat
type LevelInfo struct {
	// Version is the compound version tag (used in newer versions, 1.18+)
	Version Version `nbt:"Version,omitempty"`
	// VersionInt is the integer version tag (legacy, lowercase "version")
	VersionInt int32 `nbt:"version,omitempty"`
	// DataVersion is the integer data version tag (used in newer versions, 1.18+)
	DataVersion int32 `nbt:"DataVersion,omitempty"`
	// Old format spawn coordinates (pre-1.21)
	SpawnX int32 `nbt:"SpawnX,omitempty"`
	SpawnY int32 `nbt:"SpawnY,omitempty"`
	SpawnZ int32 `nbt:"SpawnZ,omitempty"`
	// New format spawn (1.21+) - compound with pos array
	Spawn      SpawnInfo `nbt:"spawn,omitempty"`
	LastPlayed int64     `nbt:"LastPlayed"`
	Difficulty int32     `nbt:"Difficulty"`
	GameType   int32     `nbt:"GameType"`
	LevelName  string    `nbt:"LevelName"`
	ServerBrand string   `nbt:"ServerBrand,omitempty"`
}

// SpawnInfo represents the spawn compound in Minecraft 1.21+ level.dat
type SpawnInfo struct {
	Pos       []int32 `nbt:"pos,omitempty"`
	Dimension string  `nbt:"dimension,omitempty"`
	Pitch     float32 `nbt:"pitch,omitempty"`
	Yaw       float32 `nbt:"yaw,omitempty"`
}

// GetSpawnX returns the spawn X coordinate, handling both old and new formats
func (l *LevelInfo) GetSpawnX() int32 {
	if len(l.Spawn.Pos) >= 1 {
		return l.Spawn.Pos[0]
	}
	return l.SpawnX
}

// GetSpawnY returns the spawn Y coordinate, handling both old and new formats
func (l *LevelInfo) GetSpawnY() int32 {
	if len(l.Spawn.Pos) >= 2 {
		return l.Spawn.Pos[1]
	}
	return l.SpawnY
}

// GetSpawnZ returns the spawn Z coordinate, handling both old and new formats
func (l *LevelInfo) GetSpawnZ() int32 {
	if len(l.Spawn.Pos) >= 3 {
		return l.Spawn.Pos[2]
	}
	return l.SpawnZ
}

// LevelData is the root tag wrapper
type LevelData struct {
	Data LevelInfo `nbt:"Data"`
}

// Version represents the version info (compound format, pre-1.18)
type Version struct {
	ID     int32  `nbt:"Id"`
	Name   string `nbt:"Name"`
	Series string `nbt:"Series,omitempty"`
}

// GetVersionName returns the version name string, handling both old (compound) and new (integer) formats
func (l *LevelInfo) GetVersionName() string {
	// Old format: use the compound Version tag
	if l.Version.Name != "" {
		return l.Version.Name
	}
	// New format: DataVersion is just an integer, so we return a formatted string
	if l.DataVersion != 0 {
		return fmt.Sprintf("DataVersion %d", l.DataVersion)
	}
	return "Unknown"
}
