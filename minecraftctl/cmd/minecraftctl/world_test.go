package main

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/paul/minecraftctl/pkg/config"
	"github.com/spf13/viper"
)

// setupWorldTestConfig resets viper and initializes config with test directories
func setupWorldTestConfig(t *testing.T, worldsDir string) {
	t.Helper()
	viper.Reset()
	viper.Set("worlds_dir", worldsDir)
	// Initialize config so Get() reads from Viper
	if err := config.Init(""); err != nil {
		t.Fatalf("config.Init() failed: %v", err)
	}
}

// createTestWorld creates a test world directory structure with a level.dat file
// It copies the level.dat from testdata/default/world/level.dat
func createTestWorld(t *testing.T, worldsDir, worldName string, withMapConfig bool) string {
	t.Helper()

	worldDir := filepath.Join(worldsDir, worldName)
	worldDataDir := filepath.Join(worldDir, "world")

	if err := os.MkdirAll(worldDataDir, 0755); err != nil {
		t.Fatalf("Failed to create world directory: %v", err)
	}

	// Copy the level.dat from testdata
	testdataPath := "../../testdata/default/world/level.dat"

	// Get absolute path to testdata
	absPath, err := filepath.Abs(testdataPath)
	if err != nil {
		t.Fatalf("Failed to get absolute path: %v", err)
	}

	levelDatSrc, err := os.Open(absPath)
	if err != nil {
		// Fallback: try from test working directory
		altPath := filepath.Join(os.Getenv("PWD"), "testdata", "default", "world", "level.dat")
		levelDatSrc, err = os.Open(altPath)
		if err != nil {
			// Another fallback: try from module root
			rootPath := "/Users/paul/projects/minecraft-server-host/minecraftctl/testdata/default/world/level.dat"
			levelDatSrc, err = os.Open(rootPath)
			if err != nil {
				t.Fatalf("Failed to open testdata level.dat: %v (tried %s, %s, %s)", err, absPath, altPath, rootPath)
			}
		}
	}
	defer levelDatSrc.Close()

	levelDatDst := filepath.Join(worldDataDir, "level.dat")
	dstFile, err := os.Create(levelDatDst)
	if err != nil {
		t.Fatalf("Failed to create level.dat: %v", err)
	}
	defer dstFile.Close()

	if _, err := io.Copy(dstFile, levelDatSrc); err != nil {
		t.Fatalf("Failed to copy level.dat: %v", err)
	}

	if withMapConfig {
		mapConfigPath := filepath.Join(worldDir, "map-config.yml")
		mapConfig := `defaults:
  zoomout: 2
maps:
  - name: overworld
    dimension: overworld
`
		if err := os.WriteFile(mapConfigPath, []byte(mapConfig), 0644); err != nil {
			t.Fatalf("Failed to create map-config.yml: %v", err)
		}
	}

	return worldDir
}

// captureWorldStdout captures stdout during a function execution
func captureWorldStdout(t *testing.T, fn func() error) (string, error) {
	t.Helper()
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := fn()

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	buf.ReadFrom(r)
	return buf.String(), err
}

func TestWorldListCmdExecution(t *testing.T) {
	t.Run("empty directory shows no worlds message", func(t *testing.T) {
		dir := t.TempDir()
		setupWorldTestConfig(t, dir)

		output, err := captureWorldStdout(t, func() error {
			return worldListCmd.RunE(worldListCmd, []string{})
		})

		if err != nil {
			t.Fatalf("worldListCmd.RunE() failed: %v", err)
		}

		if !strings.Contains(output, "No worlds found") {
			t.Errorf("Expected 'No worlds found' message, got: %s", output)
		}
	})

	t.Run("lists worlds with info", func(t *testing.T) {
		dir := t.TempDir()
		setupWorldTestConfig(t, dir)

		// Create test worlds
		createTestWorld(t, dir, "survival", true)
		createTestWorld(t, dir, "creative", false)

		output, err := captureWorldStdout(t, func() error {
			return worldListCmd.RunE(worldListCmd, []string{})
		})

		if err != nil {
			t.Fatalf("worldListCmd.RunE() failed: %v", err)
		}

		// Check header
		if !strings.Contains(output, "NAME") {
			t.Errorf("Expected NAME header, got: %s", output)
		}
		if !strings.Contains(output, "VERSION") {
			t.Errorf("Expected VERSION header, got: %s", output)
		}

		// Check world names are listed
		if !strings.Contains(output, "survival") {
			t.Errorf("Expected world 'survival' in output, got: %s", output)
		}
		if !strings.Contains(output, "creative") {
			t.Errorf("Expected world 'creative' in output, got: %s", output)
		}

		// Check map config indicator
		if !strings.Contains(output, "yes") {
			t.Errorf("Expected 'yes' for map config, got: %s", output)
		}
		if !strings.Contains(output, "no") {
			t.Errorf("Expected 'no' for map config, got: %s", output)
		}
	})

	t.Run("ignores non-world directories", func(t *testing.T) {
		dir := t.TempDir()
		setupWorldTestConfig(t, dir)

		// Create a valid world
		createTestWorld(t, dir, "survival", false)

		// Create a non-world directory (no level.dat)
		notAWorld := filepath.Join(dir, "not-a-world")
		os.MkdirAll(notAWorld, 0755)
		os.WriteFile(filepath.Join(notAWorld, "readme.txt"), []byte("not a world"), 0644)

		// Create a file (not a directory)
		os.WriteFile(filepath.Join(dir, "somefile.txt"), []byte("file"), 0644)

		output, err := captureWorldStdout(t, func() error {
			return worldListCmd.RunE(worldListCmd, []string{})
		})

		if err != nil {
			t.Fatalf("worldListCmd.RunE() failed: %v", err)
		}

		// Should list survival but not the non-world directory
		if !strings.Contains(output, "survival") {
			t.Errorf("Expected world 'survival' in output, got: %s", output)
		}
		if strings.Contains(output, "not-a-world") {
			t.Errorf("Should not list 'not-a-world' (no level.dat), got: %s", output)
		}
	})
}

func TestWorldInfoCmdExecution(t *testing.T) {
	t.Run("shows world info", func(t *testing.T) {
		dir := t.TempDir()
		setupWorldTestConfig(t, dir)

		createTestWorld(t, dir, "myworld", true)

		output, err := captureWorldStdout(t, func() error {
			return worldInfoCmd.RunE(worldInfoCmd, []string{"myworld"})
		})

		if err != nil {
			t.Fatalf("worldInfoCmd.RunE() failed: %v", err)
		}

		// Check output contains expected fields
		expectedFields := []string{
			"World: myworld",
			"Path:",
			"Version:",
			"Difficulty:",
			"Game Type:",
			"Spawn:",
			"Map Config: yes",
		}
		for _, field := range expectedFields {
			if !strings.Contains(output, field) {
				t.Errorf("Expected %q in output, got: %s", field, output)
			}
		}
	})

	t.Run("shows no map config for world without one", func(t *testing.T) {
		dir := t.TempDir()
		setupWorldTestConfig(t, dir)

		createTestWorld(t, dir, "nomap", false)

		output, err := captureWorldStdout(t, func() error {
			return worldInfoCmd.RunE(worldInfoCmd, []string{"nomap"})
		})

		if err != nil {
			t.Fatalf("worldInfoCmd.RunE() failed: %v", err)
		}

		if !strings.Contains(output, "Map Config: no") {
			t.Errorf("Expected 'Map Config: no' in output, got: %s", output)
		}
	})

	t.Run("error for nonexistent world", func(t *testing.T) {
		dir := t.TempDir()
		setupWorldTestConfig(t, dir)

		err := worldInfoCmd.RunE(worldInfoCmd, []string{"nonexistent"})
		if err == nil {
			t.Error("Expected error for nonexistent world")
		}
		if !strings.Contains(err.Error(), "not found") {
			t.Errorf("Error should mention 'not found': %v", err)
		}
	})
}

func TestWorldCmdStructure(t *testing.T) {
	t.Run("world cmd has correct subcommands", func(t *testing.T) {
		subcommands := []string{"list", "info", "create", "register", "upgrade"}
		for _, name := range subcommands {
			found := false
			for _, cmd := range WorldCmd.Commands() {
				if cmd.Name() == name {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("WorldCmd missing subcommand %q", name)
			}
		}
	})

	t.Run("world create cmd has required flags", func(t *testing.T) {
		versionFlag := worldCreateCmd.Flags().Lookup("version")
		if versionFlag == nil {
			t.Error("worldCreateCmd should have --version flag")
		}

		seedFlag := worldCreateCmd.Flags().Lookup("seed")
		if seedFlag == nil {
			t.Error("worldCreateCmd should have --seed flag")
		}

		noMapConfigFlag := worldCreateCmd.Flags().Lookup("no-map-config")
		if noMapConfigFlag == nil {
			t.Error("worldCreateCmd should have --no-map-config flag")
		}

		noSystemdFlag := worldCreateCmd.Flags().Lookup("no-systemd")
		if noSystemdFlag == nil {
			t.Error("worldCreateCmd should have --no-systemd flag")
		}
	})

	t.Run("world upgrade cmd has required flags", func(t *testing.T) {
		versionFlag := worldUpgradeCmd.Flags().Lookup("version")
		if versionFlag == nil {
			t.Error("worldUpgradeCmd should have --version flag")
		}

		stopFlag := worldUpgradeCmd.Flags().Lookup("stop")
		if stopFlag == nil {
			t.Error("worldUpgradeCmd should have --stop flag")
		}
	})

	t.Run("world list cmd uses", func(t *testing.T) {
		if worldListCmd.Use != "list" {
			t.Errorf("worldListCmd.Use = %q, want 'list'", worldListCmd.Use)
		}
		if worldListCmd.Short == "" {
			t.Error("worldListCmd should have Short description")
		}
	})

	t.Run("world info cmd uses", func(t *testing.T) {
		if worldInfoCmd.Use != "info <world>" {
			t.Errorf("worldInfoCmd.Use = %q, want 'info <world>'", worldInfoCmd.Use)
		}
	})

	t.Run("world create cmd uses", func(t *testing.T) {
		if worldCreateCmd.Use != "create <world-name>" {
			t.Errorf("worldCreateCmd.Use = %q, want 'create <world-name>'", worldCreateCmd.Use)
		}
	})

	t.Run("world register cmd uses", func(t *testing.T) {
		if worldRegisterCmd.Use != "register <world-name>" {
			t.Errorf("worldRegisterCmd.Use = %q, want 'register <world-name>'", worldRegisterCmd.Use)
		}
	})

	t.Run("world upgrade cmd uses", func(t *testing.T) {
		if worldUpgradeCmd.Use != "upgrade <world-name>" {
			t.Errorf("worldUpgradeCmd.Use = %q, want 'upgrade <world-name>'", worldUpgradeCmd.Use)
		}
	})

	t.Run("world cmd has description", func(t *testing.T) {
		if WorldCmd.Short == "" {
			t.Error("WorldCmd should have Short description")
		}
		if WorldCmd.Long == "" {
			t.Error("WorldCmd should have Long description")
		}
	})
}

func TestWorldCreateCmdExecution(t *testing.T) {
	t.Run("requires version flag", func(t *testing.T) {
		dir := t.TempDir()
		setupWorldTestConfig(t, dir)

		// Reset the flag value
		createVersion = ""

		err := worldCreateCmd.RunE(worldCreateCmd, []string{"testworld"})
		if err == nil {
			t.Error("Expected error when --version is missing")
		}
		if !strings.Contains(err.Error(), "version") {
			t.Errorf("Error should mention version: %v", err)
		}
	})

	t.Run("error when jar not found", func(t *testing.T) {
		dir := t.TempDir()
		setupWorldTestConfig(t, dir)

		// Set version flag to a version that doesn't exist
		createVersion = "99.99.99"

		err := worldCreateCmd.RunE(worldCreateCmd, []string{"testworld"})
		if err == nil {
			t.Error("Expected error when jar not found")
		}
		if !strings.Contains(err.Error(), "jar not found") && !strings.Contains(err.Error(), "Minecraft server jar") {
			t.Errorf("Error should mention jar not found: %v", err)
		}

		// Reset
		createVersion = ""
	})
}

func TestWorldUpgradeCmdExecution(t *testing.T) {
	t.Run("requires version flag", func(t *testing.T) {
		dir := t.TempDir()
		setupWorldTestConfig(t, dir)

		createTestWorld(t, dir, "testworld", false)

		// Reset the flag value
		upgradeVersion = ""

		err := worldUpgradeCmd.RunE(worldUpgradeCmd, []string{"testworld"})
		if err == nil {
			t.Error("Expected error when --version is missing")
		}
		if !strings.Contains(err.Error(), "version") {
			t.Errorf("Error should mention version: %v", err)
		}
	})
}
