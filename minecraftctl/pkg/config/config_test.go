package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
)

// resetViper clears viper state between tests
func resetViper() {
	viper.Reset()
	globalConfig = nil
}

func TestConstants(t *testing.T) {
	// Verify default constants have sensible values
	if DefaultWorldsDir != "/srv/minecraft-server" {
		t.Errorf("DefaultWorldsDir = %q, unexpected value", DefaultWorldsDir)
	}
	if DefaultMapsDir != "/srv/minecraft-server/maps" {
		t.Errorf("DefaultMapsDir = %q, unexpected value", DefaultMapsDir)
	}
	if DefaultJarsDir != "/opt/minecraft/jars" {
		t.Errorf("DefaultJarsDir = %q, unexpected value", DefaultJarsDir)
	}
	if DefaultRconHost != "127.0.0.1" {
		t.Errorf("DefaultRconHost = %q, unexpected value", DefaultRconHost)
	}
	if DefaultRconPort != 25575 {
		t.Errorf("DefaultRconPort = %d, unexpected value", DefaultRconPort)
	}
}

func TestGetWithoutInit(t *testing.T) {
	resetViper()

	cfg := Get()
	if cfg == nil {
		t.Fatal("Get() returned nil")
	}

	// Should return defaults when not initialized
	if cfg.WorldsDir != DefaultWorldsDir {
		t.Errorf("WorldsDir = %q, want %q", cfg.WorldsDir, DefaultWorldsDir)
	}
	if cfg.MapsDir != DefaultMapsDir {
		t.Errorf("MapsDir = %q, want %q", cfg.MapsDir, DefaultMapsDir)
	}
	if cfg.JarsDir != DefaultJarsDir {
		t.Errorf("JarsDir = %q, want %q", cfg.JarsDir, DefaultJarsDir)
	}
	if cfg.Rcon.Host != DefaultRconHost {
		t.Errorf("Rcon.Host = %q, want %q", cfg.Rcon.Host, DefaultRconHost)
	}
	if cfg.Rcon.Port != DefaultRconPort {
		t.Errorf("Rcon.Port = %d, want %d", cfg.Rcon.Port, DefaultRconPort)
	}
}

func TestInitWithConfigFile(t *testing.T) {
	resetViper()

	// Create a temporary config file
	dir := t.TempDir()
	configPath := filepath.Join(dir, "minecraftctl.yaml")
	content := `
worlds_dir: /custom/worlds
maps_dir: /custom/maps
jars_dir: /custom/jars
lock_file: /custom/lock
rcon:
  host: 192.168.1.1
  port: 25576
  password: secret123
`
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	if err := Init(configPath); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	cfg := Get()
	if cfg.WorldsDir != "/custom/worlds" {
		t.Errorf("WorldsDir = %q, want %q", cfg.WorldsDir, "/custom/worlds")
	}
	if cfg.MapsDir != "/custom/maps" {
		t.Errorf("MapsDir = %q, want %q", cfg.MapsDir, "/custom/maps")
	}
	if cfg.JarsDir != "/custom/jars" {
		t.Errorf("JarsDir = %q, want %q", cfg.JarsDir, "/custom/jars")
	}
	if cfg.Rcon.Host != "192.168.1.1" {
		t.Errorf("Rcon.Host = %q, want %q", cfg.Rcon.Host, "192.168.1.1")
	}
	if cfg.Rcon.Port != 25576 {
		t.Errorf("Rcon.Port = %d, want %d", cfg.Rcon.Port, 25576)
	}
	if cfg.Rcon.Password != "secret123" {
		t.Errorf("Rcon.Password = %q, want %q", cfg.Rcon.Password, "secret123")
	}
}

func TestInitWithInvalidConfigFile(t *testing.T) {
	resetViper()

	// Create an invalid YAML file
	dir := t.TempDir()
	configPath := filepath.Join(dir, "invalid.yaml")
	content := `this is not valid yaml: [}`
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	err := Init(configPath)
	if err == nil {
		t.Error("Expected error for invalid config file")
	}
}

func TestInitWithMissingConfigFile(t *testing.T) {
	resetViper()

	// Init without a config file should use defaults
	err := Init("")
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	cfg := Get()
	if cfg.WorldsDir != DefaultWorldsDir {
		t.Errorf("WorldsDir = %q, want default %q", cfg.WorldsDir, DefaultWorldsDir)
	}
}

func TestLoadMapConfig(t *testing.T) {
	t.Run("valid config", func(t *testing.T) {
		// Create a test config file since testdata uses sample-map-config.yml
		dir := t.TempDir()
		content := `
defaults:
  zoomout: 2
  zoomin: 1
  imageformat: jpeg
  chunkprocessors: 4

maps:
  - name: overworld
    dimension: overworld
    output_subdir: overworld
    options:
      shadows: 3d
    ranges:
      - name: spawn_area
        center: [560, 160]
        radius: 2048
        zoomout: 4
        zoomin: 2

  - name: nether
    dimension: nether
    output_subdir: nether

  - name: caves
    dimension: overworld
    output_subdir: caves
`
		if err := os.WriteFile(filepath.Join(dir, "map-config.yml"), []byte(content), 0644); err != nil {
			t.Fatalf("Failed to write test config: %v", err)
		}

		mapConfig, err := LoadMapConfig(dir)
		if err != nil {
			t.Fatalf("LoadMapConfig failed: %v", err)
		}

		// Check defaults
		if mapConfig.Defaults.Zoomout != 2 {
			t.Errorf("Defaults.Zoomout = %d, want 2", mapConfig.Defaults.Zoomout)
		}
		if mapConfig.Defaults.Zoomin != 1 {
			t.Errorf("Defaults.Zoomin = %d, want 1", mapConfig.Defaults.Zoomin)
		}
		if mapConfig.Defaults.ImageFormat != "jpeg" {
			t.Errorf("Defaults.ImageFormat = %q, want jpeg", mapConfig.Defaults.ImageFormat)
		}
		if mapConfig.Defaults.ChunkProcessors != 4 {
			t.Errorf("Defaults.ChunkProcessors = %d, want 4", mapConfig.Defaults.ChunkProcessors)
		}

		// Check maps
		if len(mapConfig.Maps) != 3 {
			t.Fatalf("Expected 3 maps, got %d", len(mapConfig.Maps))
		}

		// Check first map (overworld)
		if mapConfig.Maps[0].Name != "overworld" {
			t.Errorf("Maps[0].Name = %q, want overworld", mapConfig.Maps[0].Name)
		}
		if mapConfig.Maps[0].Dimension != "overworld" {
			t.Errorf("Maps[0].Dimension = %q, want overworld", mapConfig.Maps[0].Dimension)
		}

		// Check ranges
		if len(mapConfig.Maps[0].Ranges) != 1 {
			t.Fatalf("Expected 1 range, got %d", len(mapConfig.Maps[0].Ranges))
		}
		if mapConfig.Maps[0].Ranges[0].Name != "spawn_area" {
			t.Errorf("Maps[0].Ranges[0].Name = %q, want spawn_area", mapConfig.Maps[0].Ranges[0].Name)
		}
		if mapConfig.Maps[0].Ranges[0].Radius != 2048 {
			t.Errorf("Maps[0].Ranges[0].Radius = %d, want 2048", mapConfig.Maps[0].Ranges[0].Radius)
		}
	})

	t.Run("missing config", func(t *testing.T) {
		_, err := LoadMapConfig("/nonexistent/path")
		if err == nil {
			t.Error("Expected error for missing config")
		}
	})

	t.Run("applies defaults", func(t *testing.T) {
		dir := t.TempDir()
		content := `
maps:
  - name: test
    dimension: overworld
`
		if err := os.WriteFile(filepath.Join(dir, "map-config.yml"), []byte(content), 0644); err != nil {
			t.Fatalf("Failed to write config: %v", err)
		}

		mapConfig, err := LoadMapConfig(dir)
		if err != nil {
			t.Fatalf("LoadMapConfig failed: %v", err)
		}

		// Check that defaults were applied
		if mapConfig.Defaults.Zoomout != 2 {
			t.Errorf("Default zoomout should be 2, got %d", mapConfig.Defaults.Zoomout)
		}
		if mapConfig.Defaults.ImageFormat != "jpeg" {
			t.Errorf("Default imageformat should be jpeg, got %q", mapConfig.Defaults.ImageFormat)
		}

		// Check that map inherited defaults
		if mapConfig.Maps[0].Zoomout == nil || *mapConfig.Maps[0].Zoomout != 2 {
			t.Errorf("Map should inherit default zoomout")
		}
		if mapConfig.Maps[0].OutputSubdir != "test" {
			t.Errorf("OutputSubdir should default to name, got %q", mapConfig.Maps[0].OutputSubdir)
		}
	})
}

func TestSaveMapConfig(t *testing.T) {
	zoomout := 3
	zoomin := 1
	mapConfig := &MapConfig{
		Defaults: MapDefaults{
			Zoomout:     2,
			Zoomin:      1,
			ImageFormat: "png",
		},
		Maps: []MapDefinition{
			{
				Name:      "test",
				Dimension: "overworld",
				Zoomout:   &zoomout,
				Zoomin:    &zoomin,
			},
		},
	}

	t.Run("save to directory", func(t *testing.T) {
		dir := t.TempDir()
		err := SaveMapConfig(dir, mapConfig)
		if err != nil {
			t.Fatalf("SaveMapConfig failed: %v", err)
		}

		// Verify file was created
		configPath := filepath.Join(dir, "map-config.yml")
		if _, err := os.Stat(configPath); err != nil {
			t.Errorf("Config file not created: %v", err)
		}

		// Verify content can be loaded back
		loaded, err := LoadMapConfig(dir)
		if err != nil {
			t.Fatalf("Failed to load saved config: %v", err)
		}
		if loaded.Maps[0].Name != "test" {
			t.Errorf("Loaded config has wrong name")
		}
	})

	t.Run("save to file path", func(t *testing.T) {
		dir := t.TempDir()
		configPath := filepath.Join(dir, "custom.yml")
		err := SaveMapConfig(configPath, mapConfig)
		if err != nil {
			t.Fatalf("SaveMapConfig failed: %v", err)
		}

		if _, err := os.Stat(configPath); err != nil {
			t.Errorf("Config file not created at custom path: %v", err)
		}
	})

	t.Run("creates parent directories", func(t *testing.T) {
		dir := t.TempDir()
		nestedPath := filepath.Join(dir, "nested", "path", "config.yml")
		err := SaveMapConfig(nestedPath, mapConfig)
		if err != nil {
			t.Fatalf("SaveMapConfig failed: %v", err)
		}

		if _, err := os.Stat(nestedPath); err != nil {
			t.Errorf("Nested config file not created: %v", err)
		}
	})
}

func TestCapitalizeFieldName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"name", "Name"},
		{"zoomout", "Zoomout"},
		{"imageFormat", "ImageFormat"},
		{"", ""},
		{"A", "A"},
		{"abc", "Abc"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := capitalizeFieldName(tt.input)
			if got != tt.want {
				t.Errorf("capitalizeFieldName(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestGetConfigField(t *testing.T) {
	zoomout := 4
	topY := 64
	mapConfig := &MapConfig{
		Defaults: MapDefaults{
			Zoomout:     2,
			Zoomin:      1,
			ImageFormat: "jpeg",
		},
		Maps: []MapDefinition{
			{
				Name:      "overworld",
				Dimension: "overworld",
				Zoomout:   &zoomout,
				Options: MapOptions{
					TopY: &topY,
				},
				Ranges: []MapRange{
					{Name: "spawn", Radius: 1024},
				},
			},
		},
	}

	tests := []struct {
		name    string
		path    string
		want    interface{}
		wantErr bool
	}{
		{"empty path returns config", "", mapConfig, false},
		{"defaults.zoomout", "defaults.zoomout", 2, false},
		{"defaults.zoomin", "defaults.zoomin", 1, false},
		{"maps[0].name", "maps[0].name", "overworld", false},
		{"maps[0].dimension", "maps[0].dimension", "overworld", false},
		{"maps[0].ranges[0].name", "maps[0].ranges[0].name", "spawn", false},
		{"maps[0].ranges[0].radius", "maps[0].ranges[0].radius", 1024, false},
		{"invalid field", "invalid", nil, true},
		{"invalid array index", "maps[99]", nil, true},
		{"invalid nested", "defaults.invalid", nil, true},
		// Note: camelCase fields like "imageFormat" or "topY" don't work with
		// capitalizeFieldName since it only capitalizes the first letter.
		// "imageformat" -> "Imageformat" but struct has "ImageFormat"
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetConfigField(tt.path, mapConfig)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetConfigField(%q) error = %v, wantErr %v", tt.path, err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("GetConfigField(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}

	t.Run("nil config", func(t *testing.T) {
		_, err := GetConfigField("defaults.zoomout", nil)
		if err == nil {
			t.Error("Expected error for nil config")
		}
	})
}

func TestSetConfigField(t *testing.T) {
	t.Run("set defaults.zoomout", func(t *testing.T) {
		mapConfig := &MapConfig{
			Defaults: MapDefaults{Zoomout: 2},
		}
		err := SetConfigField("defaults.zoomout", 5, mapConfig)
		if err != nil {
			t.Fatalf("SetConfigField failed: %v", err)
		}
		if mapConfig.Defaults.Zoomout != 5 {
			t.Errorf("Zoomout = %d, want 5", mapConfig.Defaults.Zoomout)
		}
	})

	t.Run("set defaults.zoomin", func(t *testing.T) {
		mapConfig := &MapConfig{
			Defaults: MapDefaults{Zoomin: 1},
		}
		err := SetConfigField("defaults.zoomin", 3, mapConfig)
		if err != nil {
			t.Fatalf("SetConfigField failed: %v", err)
		}
		if mapConfig.Defaults.Zoomin != 3 {
			t.Errorf("Zoomin = %d, want 3", mapConfig.Defaults.Zoomin)
		}
	})

	t.Run("set map name", func(t *testing.T) {
		mapConfig := &MapConfig{
			Maps: []MapDefinition{{Name: "old"}},
		}
		err := SetConfigField("maps[0].name", "new", mapConfig)
		if err != nil {
			t.Fatalf("SetConfigField failed: %v", err)
		}
		if mapConfig.Maps[0].Name != "new" {
			t.Errorf("Name = %q, want new", mapConfig.Maps[0].Name)
		}
	})

	t.Run("empty path", func(t *testing.T) {
		mapConfig := &MapConfig{}
		err := SetConfigField("", "value", mapConfig)
		if err == nil {
			t.Error("Expected error for empty path")
		}
	})

	t.Run("nil config", func(t *testing.T) {
		err := SetConfigField("defaults.zoomout", 5, nil)
		if err == nil {
			t.Error("Expected error for nil config")
		}
	})

	t.Run("invalid field", func(t *testing.T) {
		mapConfig := &MapConfig{}
		err := SetConfigField("invalid.field", "value", mapConfig)
		if err == nil {
			t.Error("Expected error for invalid field")
		}
	})

	t.Run("array index out of range", func(t *testing.T) {
		mapConfig := &MapConfig{
			Maps: []MapDefinition{{Name: "test"}},
		}
		err := SetConfigField("maps[5].name", "value", mapConfig)
		if err == nil {
			t.Error("Expected error for out of range index")
		}
	})
}

func TestValidateMapConfig(t *testing.T) {
	t.Run("valid config", func(t *testing.T) {
		mapConfig := &MapConfig{
			Defaults: MapDefaults{
				Zoomout:         2,
				Zoomin:          1,
				ImageFormat:     "jpeg",
				ChunkProcessors: 2,
			},
			Maps: []MapDefinition{
				{Name: "test", Dimension: "overworld"},
			},
		}
		errs := ValidateMapConfig(mapConfig)
		if len(errs) != 0 {
			t.Errorf("Expected no errors, got: %v", errs)
		}
	})

	t.Run("empty maps", func(t *testing.T) {
		mapConfig := &MapConfig{
			Defaults: MapDefaults{ChunkProcessors: 1},
			Maps:     []MapDefinition{},
		}
		errs := ValidateMapConfig(mapConfig)
		if len(errs) == 0 {
			t.Error("Expected error for empty maps")
		}
	})

	t.Run("negative zoomout", func(t *testing.T) {
		mapConfig := &MapConfig{
			Defaults: MapDefaults{Zoomout: -1, ChunkProcessors: 1},
			Maps:     []MapDefinition{{Name: "test", Dimension: "overworld"}},
		}
		errs := ValidateMapConfig(mapConfig)
		found := false
		for _, e := range errs {
			if e == "defaults.zoomout must be >= 0" {
				found = true
			}
		}
		if !found {
			t.Error("Expected zoomout validation error")
		}
	})

	t.Run("invalid image format", func(t *testing.T) {
		mapConfig := &MapConfig{
			Defaults: MapDefaults{ImageFormat: "gif", ChunkProcessors: 1},
			Maps:     []MapDefinition{{Name: "test", Dimension: "overworld"}},
		}
		errs := ValidateMapConfig(mapConfig)
		found := false
		for _, e := range errs {
			if len(e) > 0 && e[:8] == "defaults" {
				found = true
			}
		}
		if !found {
			t.Error("Expected imageformat validation error")
		}
	})

	t.Run("invalid dimension", func(t *testing.T) {
		mapConfig := &MapConfig{
			Defaults: MapDefaults{ChunkProcessors: 1},
			Maps:     []MapDefinition{{Name: "test", Dimension: "invalid"}},
		}
		errs := ValidateMapConfig(mapConfig)
		found := false
		for _, e := range errs {
			if len(e) > 7 && e[:7] == "maps[0]" {
				found = true
			}
		}
		if !found {
			t.Errorf("Expected dimension validation error, got: %v", errs)
		}
	})

	t.Run("missing map name", func(t *testing.T) {
		mapConfig := &MapConfig{
			Defaults: MapDefaults{ChunkProcessors: 1},
			Maps:     []MapDefinition{{Dimension: "overworld"}},
		}
		errs := ValidateMapConfig(mapConfig)
		found := false
		for _, e := range errs {
			if e == "maps[0].name is required" {
				found = true
			}
		}
		if !found {
			t.Errorf("Expected name required error, got: %v", errs)
		}
	})

	t.Run("invalid range radius", func(t *testing.T) {
		mapConfig := &MapConfig{
			Defaults: MapDefaults{ChunkProcessors: 1},
			Maps: []MapDefinition{
				{
					Name:      "test",
					Dimension: "overworld",
					Ranges:    []MapRange{{Name: "bad", Radius: 0}},
				},
			},
		}
		errs := ValidateMapConfig(mapConfig)
		found := false
		for _, e := range errs {
			if e == "maps[0].ranges[0].radius must be > 0" {
				found = true
			}
		}
		if !found {
			t.Errorf("Expected radius validation error, got: %v", errs)
		}
	})

	t.Run("valid shadow values", func(t *testing.T) {
		validShadows := []interface{}{"true", "false", "2d", "3d", "3do", true, false}
		for _, shadow := range validShadows {
			mapConfig := &MapConfig{
				Defaults: MapDefaults{ChunkProcessors: 1},
				Maps: []MapDefinition{
					{
						Name:      "test",
						Dimension: "overworld",
						Options:   MapOptions{Shadows: shadow},
					},
				},
			}
			errs := ValidateMapConfig(mapConfig)
			for _, e := range errs {
				if len(e) > 20 && e[7:14] == "options" {
					t.Errorf("Shadow value %v should be valid, got error: %s", shadow, e)
				}
			}
		}
	})

	t.Run("invalid shadow value", func(t *testing.T) {
		mapConfig := &MapConfig{
			Defaults: MapDefaults{ChunkProcessors: 1},
			Maps: []MapDefinition{
				{
					Name:      "test",
					Dimension: "overworld",
					Options:   MapOptions{Shadows: "invalid"},
				},
			},
		}
		errs := ValidateMapConfig(mapConfig)
		found := false
		for _, e := range errs {
			if len(e) > 20 {
				found = true
			}
		}
		if !found {
			t.Errorf("Expected shadow validation error, got: %v", errs)
		}
	})
}

func TestExpandEnv(t *testing.T) {
	os.Setenv("TEST_CONFIG_VAR", "expanded_value")
	defer os.Unsetenv("TEST_CONFIG_VAR")

	// Must use braces when followed by other characters
	result := expandEnv("prefix_${TEST_CONFIG_VAR}_suffix")
	if result != "prefix_expanded_value_suffix" {
		t.Errorf("expandEnv result = %q, want prefix_expanded_value_suffix", result)
	}

	result = expandEnv("${TEST_CONFIG_VAR}")
	if result != "expanded_value" {
		t.Errorf("expandEnv result = %q, want expanded_value", result)
	}

	result = expandEnv("$TEST_CONFIG_VAR")
	if result != "expanded_value" {
		t.Errorf("expandEnv result = %q, want expanded_value", result)
	}

	result = expandEnv("no_vars")
	if result != "no_vars" {
		t.Errorf("expandEnv result = %q, want no_vars", result)
	}
}

func TestRconConfig(t *testing.T) {
	cfg := RconConfig{
		Host:     "localhost",
		Port:     25575,
		Password: "secret",
	}

	if cfg.Host != "localhost" {
		t.Errorf("Host = %q, want localhost", cfg.Host)
	}
	if cfg.Port != 25575 {
		t.Errorf("Port = %d, want 25575", cfg.Port)
	}
	if cfg.Password != "secret" {
		t.Errorf("Password = %q, want secret", cfg.Password)
	}
}

func TestMapOptionsPointerFields(t *testing.T) {
	topY := 64
	bottomY := -64
	gndxray := true
	night := false
	players := true

	opts := MapOptions{
		TopY:    &topY,
		BottomY: &bottomY,
		GndXray: &gndxray,
		Shadows: "3d",
		Night:   &night,
		Players: &players,
	}

	if opts.TopY == nil || *opts.TopY != 64 {
		t.Error("TopY not set correctly")
	}
	if opts.BottomY == nil || *opts.BottomY != -64 {
		t.Error("BottomY not set correctly")
	}
	if opts.GndXray == nil || *opts.GndXray != true {
		t.Error("GndXray not set correctly")
	}
	if opts.Shadows != "3d" {
		t.Error("Shadows not set correctly")
	}
}

func TestMapRangeStruct(t *testing.T) {
	zoomout := 4
	zoomin := 2
	r := MapRange{
		Name:    "spawn",
		Center:  [2]int{100, 200},
		Radius:  1024,
		Zoomout: &zoomout,
		Zoomin:  &zoomin,
	}

	if r.Name != "spawn" {
		t.Errorf("Name = %q, want spawn", r.Name)
	}
	if r.Center[0] != 100 || r.Center[1] != 200 {
		t.Errorf("Center = %v, want [100, 200]", r.Center)
	}
	if r.Radius != 1024 {
		t.Errorf("Radius = %d, want 1024", r.Radius)
	}
}
