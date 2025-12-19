package nbt

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadLevelDat(t *testing.T) {
	// Find the testdata directory relative to this test file
	testdataPath := filepath.Join("..", "..", "testdata", "default", "world", "level.dat")

	t.Run("valid level.dat", func(t *testing.T) {
		info, err := ReadLevelDat(testdataPath)
		if err != nil {
			t.Fatalf("ReadLevelDat(%q) returned error: %v", testdataPath, err)
		}

		if info == nil {
			t.Fatal("ReadLevelDat returned nil info without error")
		}

		// Verify we got some data (exact values depend on the test file)
		if info.LevelName == "" {
			t.Error("Expected LevelName to be non-empty")
		}

		t.Logf("Read level.dat: LevelName=%q, Version=%q, SpawnX=%d, SpawnY=%d, SpawnZ=%d",
			info.LevelName, info.GetVersionName(), info.SpawnX, info.SpawnY, info.SpawnZ)
	})

	t.Run("missing file", func(t *testing.T) {
		_, err := ReadLevelDat("/nonexistent/path/level.dat")
		if err == nil {
			t.Error("Expected error for missing file, got nil")
		}
	})

	t.Run("invalid file (not gzip)", func(t *testing.T) {
		// Create a temporary non-gzip file
		tmpFile, err := os.CreateTemp("", "invalid-level-*.dat")
		if err != nil {
			t.Fatalf("Failed to create temp file: %v", err)
		}
		defer os.Remove(tmpFile.Name())

		tmpFile.WriteString("not gzip data")
		tmpFile.Close()

		_, err = ReadLevelDat(tmpFile.Name())
		if err == nil {
			t.Error("Expected error for non-gzip file, got nil")
		}
	})

	t.Run("empty file", func(t *testing.T) {
		tmpFile, err := os.CreateTemp("", "empty-level-*.dat")
		if err != nil {
			t.Fatalf("Failed to create temp file: %v", err)
		}
		defer os.Remove(tmpFile.Name())
		tmpFile.Close()

		_, err = ReadLevelDat(tmpFile.Name())
		if err == nil {
			t.Error("Expected error for empty file, got nil")
		}
	})
}

func TestGetVersionName(t *testing.T) {
	tests := []struct {
		name string
		info LevelInfo
		want string
	}{
		{
			name: "compound version format",
			info: LevelInfo{
				Version: Version{
					ID:   3578,
					Name: "1.20.4",
				},
			},
			want: "1.20.4",
		},
		{
			name: "compound version with series",
			info: LevelInfo{
				Version: Version{
					ID:     3700,
					Name:   "1.21",
					Series: "main",
				},
			},
			want: "1.21",
		},
		{
			name: "data version only (new format)",
			info: LevelInfo{
				DataVersion: 3700,
			},
			want: "DataVersion 3700",
		},
		{
			name: "both formats prefers compound",
			info: LevelInfo{
				Version: Version{
					ID:   3578,
					Name: "1.20.4",
				},
				DataVersion: 3578,
			},
			want: "1.20.4",
		},
		{
			name: "no version info",
			info: LevelInfo{},
			want: "Unknown",
		},
		{
			name: "empty version name with data version",
			info: LevelInfo{
				Version: Version{
					ID:   3578,
					Name: "", // empty
				},
				DataVersion: 3578,
			},
			want: "DataVersion 3578",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.info.GetVersionName()
			if got != tt.want {
				t.Errorf("GetVersionName() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestLevelInfoFields(t *testing.T) {
	// Test that the struct fields are properly accessible
	info := LevelInfo{
		Version: Version{
			ID:     3578,
			Name:   "1.20.4",
			Series: "main",
		},
		VersionInt:  19133,
		DataVersion: 3578,
		SpawnX:      100,
		SpawnY:      64,
		SpawnZ:      -200,
		LastPlayed:  1700000000000,
		Difficulty:  2,
		GameType:    0,
		LevelName:   "Test World",
		ServerBrand: "vanilla",
	}

	if info.SpawnX != 100 {
		t.Errorf("SpawnX = %d, want 100", info.SpawnX)
	}
	if info.SpawnY != 64 {
		t.Errorf("SpawnY = %d, want 64", info.SpawnY)
	}
	if info.SpawnZ != -200 {
		t.Errorf("SpawnZ = %d, want -200", info.SpawnZ)
	}
	if info.Difficulty != 2 {
		t.Errorf("Difficulty = %d, want 2", info.Difficulty)
	}
	if info.GameType != 0 {
		t.Errorf("GameType = %d, want 0", info.GameType)
	}
	if info.LevelName != "Test World" {
		t.Errorf("LevelName = %q, want %q", info.LevelName, "Test World")
	}
	if info.ServerBrand != "vanilla" {
		t.Errorf("ServerBrand = %q, want %q", info.ServerBrand, "vanilla")
	}
}

func TestReadLevelDatIntegration(t *testing.T) {
	// Integration test with actual testdata file
	testdataPath := filepath.Join("..", "..", "testdata", "default", "world", "level.dat")

	// Skip if testdata doesn't exist
	if _, err := os.Stat(testdataPath); os.IsNotExist(err) {
		t.Skipf("Testdata file not found: %s", testdataPath)
	}

	info, err := ReadLevelDat(testdataPath)
	if err != nil {
		t.Fatalf("ReadLevelDat failed: %v", err)
	}

	// The testdata should have valid spawn coordinates
	// (they should be reasonable Minecraft coordinates)
	if info.SpawnY < -64 || info.SpawnY > 320 {
		t.Errorf("SpawnY = %d, outside reasonable range [-64, 320]", info.SpawnY)
	}

	// Difficulty should be 0-3
	if info.Difficulty < 0 || info.Difficulty > 3 {
		t.Errorf("Difficulty = %d, should be 0-3", info.Difficulty)
	}

	// GameType should be 0-3
	if info.GameType < 0 || info.GameType > 3 {
		t.Errorf("GameType = %d, should be 0-3", info.GameType)
	}

	// LastPlayed should be a reasonable timestamp (after 2020)
	minTimestamp := int64(1577836800000) // 2020-01-01 in milliseconds
	if info.LastPlayed < minTimestamp {
		t.Errorf("LastPlayed = %d, seems too old (before 2020)", info.LastPlayed)
	}

	// Version should be parseable
	version := info.GetVersionName()
	if version == "" || version == "Unknown" {
		t.Errorf("Expected valid version, got %q", version)
	}
}
