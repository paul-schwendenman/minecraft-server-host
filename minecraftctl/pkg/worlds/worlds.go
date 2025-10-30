package worlds

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/paul/minecraftctl/pkg/config"
	"github.com/paul/minecraftctl/pkg/nbt"
	"github.com/rs/zerolog/log"
)

// WorldInfo contains information about a Minecraft world
type WorldInfo struct {
	Name         string
	Path         string
	Version      string
	SpawnX       int32
	SpawnY       int32
	SpawnZ       int32
	LastPlayed   time.Time
	Difficulty   string
	GameType     string
	LevelName    string
	HasMapConfig bool
}

// ListWorlds returns a list of all worlds in the configured worlds directory
func ListWorlds() ([]WorldInfo, error) {
	cfg := config.Get()
	worldsDir := cfg.WorldsDir

	entries, err := os.ReadDir(worldsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read worlds directory: %w", err)
	}

	var worlds []WorldInfo
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		worldPath := filepath.Join(worldsDir, entry.Name())
		levelDatPath := filepath.Join(worldPath, "world", "level.dat")

		// Check if this looks like a world directory
		if _, err := os.Stat(levelDatPath); os.IsNotExist(err) {
			continue
		}

		info, err := GetWorldInfo(entry.Name())
		if err != nil {
			log.Warn().Err(err).Str("world", entry.Name()).Msg("failed to get world info")
			continue
		}

		worlds = append(worlds, *info)
	}

	// Sort by name
	sort.Slice(worlds, func(i, j int) bool {
		return worlds[i].Name < worlds[j].Name
	})

	return worlds, nil
}

// GetWorldInfo retrieves information about a specific world
func GetWorldInfo(worldName string) (*WorldInfo, error) {
	cfg := config.Get()
	worldPath := filepath.Join(cfg.WorldsDir, worldName)
	levelDatPath := filepath.Join(worldPath, "world", "level.dat")

	if _, err := os.Stat(levelDatPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("world not found: %s (no level.dat at %s)", worldName, levelDatPath)
	}

	levelInfo, err := nbt.ReadLevelDat(levelDatPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read level.dat: %w", err)
	}

	// Check for map-config.yml
	mapConfigPath := filepath.Join(worldPath, "map-config.yml")
	hasMapConfig := true
	if _, err := os.Stat(mapConfigPath); os.IsNotExist(err) {
		hasMapConfig = false
	}

	difficulty := difficultyName(levelInfo.Difficulty)
	gameType := gameTypeName(levelInfo.GameType)

	return &WorldInfo{
		Name:         worldName,
		Path:         worldPath,
		Version:      levelInfo.Version.Name,
		SpawnX:       levelInfo.SpawnX,
		SpawnY:       levelInfo.SpawnY,
		SpawnZ:       levelInfo.SpawnZ,
		LastPlayed:   time.Unix(levelInfo.LastPlayed/1000, 0),
		Difficulty:   difficulty,
		GameType:     gameType,
		LevelName:    levelInfo.LevelName,
		HasMapConfig: hasMapConfig,
	}, nil
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

func gameTypeName(gt int32) string {
	switch gt {
	case 0:
		return "Survival"
	case 1:
		return "Creative"
	case 2:
		return "Adventure"
	case 3:
		return "Spectator"
	default:
		return "Unknown"
	}
}

