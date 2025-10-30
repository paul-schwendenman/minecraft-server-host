package nbt

import (
	"fmt"
	"os"

	nbtlib "github.com/Tnze/go-mc/nbt"
)

// ReadLevelDat reads basic information from a Minecraft world's level.dat file
func ReadLevelDat(levelDatPath string) (*LevelInfo, error) {
	file, err := os.Open(levelDatPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open level.dat: %w", err)
	}
	defer file.Close()

	var data LevelData
	if _, err := nbtlib.NewDecoder(file).Decode(&data); err != nil {
		return nil, fmt.Errorf("failed to decode NBT: %w", err)
	}

	return &data.Data, nil
}

// LevelInfo represents the Data tag within level.dat
type LevelInfo struct {
	Version         Version `nbt:"Version"`
	SpawnX          int32   `nbt:"SpawnX"`
	SpawnY          int32   `nbt:"SpawnY"`
	SpawnZ          int32   `nbt:"SpawnZ"`
	LastPlayed      int64   `nbt:"LastPlayed"`
	Difficulty      int32   `nbt:"Difficulty"`
	GameType        int32   `nbt:"GameType"`
	LevelName       string  `nbt:"LevelName"`
	ServerBrand     string  `nbt:"ServerBrand,omitempty"`
}

// LevelData is the root tag wrapper
type LevelData struct {
	Data LevelInfo `nbt:"Data"`
}

// Version represents the version info
type Version struct {
	ID     int32  `nbt:"Id"`
	Name   string `nbt:"Name"`
	Series string `nbt:"Series,omitempty"`
}

