package maps

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/paul/minecraftctl/pkg/config"
)

func TestManifestStruct(t *testing.T) {
	m := Manifest{
		World:             "test_world",
		Map:               "overworld",
		Dimension:         "overworld",
		Path:              "/path/to/map",
		LastRendered:      "2024-01-15T10:30:00Z",
		LastRenderedEpoch: 1705314600,
	}

	if m.World != "test_world" {
		t.Errorf("World = %q, want test_world", m.World)
	}
	if m.Map != "overworld" {
		t.Errorf("Map = %q, want overworld", m.Map)
	}
	if m.Dimension != "overworld" {
		t.Errorf("Dimension = %q, want overworld", m.Dimension)
	}
	if m.LastRenderedEpoch != 1705314600 {
		t.Errorf("LastRenderedEpoch = %d, want 1705314600", m.LastRenderedEpoch)
	}
}

func TestManifestJSON(t *testing.T) {
	m := Manifest{
		World:             "test_world",
		Map:               "overworld",
		Dimension:         "overworld",
		LastRendered:      "2024-01-15T10:30:00Z",
		LastRenderedEpoch: 1705314600,
	}

	// Marshal
	data, err := json.Marshal(m)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	// Unmarshal
	var m2 Manifest
	if err := json.Unmarshal(data, &m2); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if m2.World != m.World {
		t.Errorf("World mismatch after round-trip")
	}
	if m2.Map != m.Map {
		t.Errorf("Map mismatch after round-trip")
	}
	if m2.LastRenderedEpoch != m.LastRenderedEpoch {
		t.Errorf("LastRenderedEpoch mismatch after round-trip")
	}
}

func TestManifestJSONOmitEmpty(t *testing.T) {
	m := Manifest{
		World:             "test",
		Map:               "test",
		Dimension:         "overworld",
		LastRendered:      "2024-01-01T00:00:00Z",
		LastRenderedEpoch: 1704067200,
		// Path is empty, should be omitted
	}

	data, err := json.Marshal(m)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	// Path should not appear in JSON when empty
	jsonStr := string(data)
	if jsonStr != `{"world":"test","map":"test","dimension":"overworld","last_rendered":"2024-01-01T00:00:00Z","last_rendered_epoch":1704067200}` {
		t.Logf("JSON: %s", jsonStr)
		// Just check path is not present
		var parsed map[string]interface{}
		json.Unmarshal(data, &parsed)
		if _, ok := parsed["path"]; ok {
			t.Error("Path should be omitted when empty")
		}
	}
}

func TestReadManifest(t *testing.T) {
	t.Run("valid manifest", func(t *testing.T) {
		dir := t.TempDir()
		manifestPath := filepath.Join(dir, "manifest.json")

		content := `{
			"world": "test_world",
			"map": "overworld",
			"dimension": "overworld",
			"last_rendered": "2024-01-15T10:30:00Z",
			"last_rendered_epoch": 1705314600
		}`
		if err := os.WriteFile(manifestPath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to write manifest: %v", err)
		}

		m, err := readManifest(manifestPath)
		if err != nil {
			t.Fatalf("readManifest failed: %v", err)
		}

		if m.World != "test_world" {
			t.Errorf("World = %q, want test_world", m.World)
		}
		if m.LastRenderedEpoch != 1705314600 {
			t.Errorf("LastRenderedEpoch = %d, want 1705314600", m.LastRenderedEpoch)
		}
	})

	t.Run("missing file", func(t *testing.T) {
		_, err := readManifest("/nonexistent/manifest.json")
		if err == nil {
			t.Error("Expected error for missing file")
		}
	})

	t.Run("invalid JSON", func(t *testing.T) {
		dir := t.TempDir()
		manifestPath := filepath.Join(dir, "manifest.json")
		os.WriteFile(manifestPath, []byte("not valid json"), 0644)

		_, err := readManifest(manifestPath)
		if err == nil {
			t.Error("Expected error for invalid JSON")
		}
	})
}

func TestBuilderStruct(t *testing.T) {
	b := &Builder{
		unminedPath: "/custom/unmined",
		worldsDir:   "/srv/worlds",
		mapsDir:     "/srv/maps",
	}

	if b.unminedPath != "/custom/unmined" {
		t.Errorf("unminedPath = %q, unexpected", b.unminedPath)
	}
	if b.worldsDir != "/srv/worlds" {
		t.Errorf("worldsDir = %q, unexpected", b.worldsDir)
	}
	if b.mapsDir != "/srv/maps" {
		t.Errorf("mapsDir = %q, unexpected", b.mapsDir)
	}
}

func TestBuildOptionsStruct(t *testing.T) {
	opts := BuildOptions{
		WorldName:   "test_world",
		MapName:     "overworld",
		Force:       true,
		Clean:       false,
		LockFile:    "/tmp/test.lock",
		LockTimeout: 5 * time.Second,
		NoLock:      false,
		NonBlocking: true,
		LogLevel:    "warning",
	}

	if opts.WorldName != "test_world" {
		t.Errorf("WorldName = %q, unexpected", opts.WorldName)
	}
	if !opts.Force {
		t.Error("Force should be true")
	}
	if opts.LockTimeout != 5*time.Second {
		t.Errorf("LockTimeout = %v, unexpected", opts.LockTimeout)
	}
	if opts.LogLevel != "warning" {
		t.Errorf("LogLevel = %q, unexpected", opts.LogLevel)
	}
}

func TestAddMapOptions(t *testing.T) {
	b := &Builder{}

	t.Run("empty options", func(t *testing.T) {
		args := []string{"web", "render"}
		opts := config.MapOptions{}
		result := b.addMapOptions(args, opts)

		if len(result) != 2 {
			t.Errorf("Expected no additional args, got %v", result)
		}
	})

	t.Run("topY option", func(t *testing.T) {
		args := []string{"web", "render"}
		topY := 64
		opts := config.MapOptions{TopY: &topY}
		result := b.addMapOptions(args, opts)

		if len(result) != 4 {
			t.Fatalf("Expected 4 args, got %d", len(result))
		}
		if result[2] != "--topY" || result[3] != "64" {
			t.Errorf("TopY not added correctly: %v", result)
		}
	})

	t.Run("bottomY option", func(t *testing.T) {
		args := []string{"web", "render"}
		bottomY := -64
		opts := config.MapOptions{BottomY: &bottomY}
		result := b.addMapOptions(args, opts)

		if len(result) != 4 {
			t.Fatalf("Expected 4 args, got %d", len(result))
		}
		if result[2] != "--bottomY" || result[3] != "-64" {
			t.Errorf("BottomY not added correctly: %v", result)
		}
	})

	t.Run("gndxray true", func(t *testing.T) {
		args := []string{"web", "render"}
		gndxray := true
		opts := config.MapOptions{GndXray: &gndxray}
		result := b.addMapOptions(args, opts)

		if len(result) != 4 {
			t.Fatalf("Expected 4 args, got %d", len(result))
		}
		if result[2] != "--gndxray" || result[3] != "true" {
			t.Errorf("GndXray not added correctly: %v", result)
		}
	})

	t.Run("gndxray false", func(t *testing.T) {
		args := []string{"web", "render"}
		gndxray := false
		opts := config.MapOptions{GndXray: &gndxray}
		result := b.addMapOptions(args, opts)

		// GndXray false should not add the arg
		if len(result) != 2 {
			t.Errorf("GndXray false should not add args: %v", result)
		}
	})

	t.Run("night option", func(t *testing.T) {
		args := []string{"web", "render"}
		night := true
		opts := config.MapOptions{Night: &night}
		result := b.addMapOptions(args, opts)

		if len(result) != 4 {
			t.Fatalf("Expected 4 args, got %d", len(result))
		}
		if result[2] != "--night" || result[3] != "true" {
			t.Errorf("Night not added correctly: %v", result)
		}
	})

	t.Run("shadows bool true", func(t *testing.T) {
		args := []string{"web", "render"}
		opts := config.MapOptions{Shadows: true}
		result := b.addMapOptions(args, opts)

		if len(result) != 4 {
			t.Fatalf("Expected 4 args, got %d", len(result))
		}
		if result[2] != "--shadows" || result[3] != "true" {
			t.Errorf("Shadows bool true not added correctly: %v", result)
		}
	})

	t.Run("shadows bool false", func(t *testing.T) {
		args := []string{"web", "render"}
		opts := config.MapOptions{Shadows: false}
		result := b.addMapOptions(args, opts)

		if len(result) != 4 {
			t.Fatalf("Expected 4 args, got %d", len(result))
		}
		if result[2] != "--shadows" || result[3] != "false" {
			t.Errorf("Shadows bool false not added correctly: %v", result)
		}
	})

	t.Run("shadows string 3d", func(t *testing.T) {
		args := []string{"web", "render"}
		opts := config.MapOptions{Shadows: "3d"}
		result := b.addMapOptions(args, opts)

		if len(result) != 4 {
			t.Fatalf("Expected 4 args, got %d", len(result))
		}
		if result[2] != "--shadows" || result[3] != "3d" {
			t.Errorf("Shadows string 3d not added correctly: %v", result)
		}
	})

	t.Run("shadows string 2d", func(t *testing.T) {
		args := []string{"web", "render"}
		opts := config.MapOptions{Shadows: "2d"}
		result := b.addMapOptions(args, opts)

		if result[2] != "--shadows" || result[3] != "2d" {
			t.Errorf("Shadows string 2d not added correctly: %v", result)
		}
	})

	t.Run("multiple options", func(t *testing.T) {
		args := []string{"web", "render"}
		topY := 64
		bottomY := -64
		gndxray := true
		opts := config.MapOptions{
			TopY:    &topY,
			BottomY: &bottomY,
			GndXray: &gndxray,
			Shadows: "3d",
		}
		result := b.addMapOptions(args, opts)

		// Should have: web, render, --topY, 64, --bottomY, -64, --gndxray, true, --shadows, 3d
		if len(result) != 10 {
			t.Errorf("Expected 10 args, got %d: %v", len(result), result)
		}
	})
}

func TestDefaultUnminedPath(t *testing.T) {
	if DefaultUnminedPath != "/opt/unmined/unmined-cli" {
		t.Errorf("DefaultUnminedPath = %q, unexpected", DefaultUnminedPath)
	}
}

func TestRangeBounds(t *testing.T) {
	tests := []struct {
		name           string
		r              config.MapRange
		x1, z1, x2, z2 int
		wantErr        bool
	}{
		{"already aligned", config.MapRange{Center: [2]int{0, 0}, Radius: 2048}, -2048, -2048, 2048, 2048, false},
		{"western_base", config.MapRange{Center: [2]int{-7424, 576}, Radius: 384}, -8192, 0, -6656, 1024, false},
		{"small positive", config.MapRange{Center: [2]int{560, 160}, Radius: 100}, 0, 0, 1024, 512, false},
		{"zero radius", config.MapRange{Center: [2]int{0, 0}, Radius: 0}, 0, 0, 0, 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			x1, z1, x2, z2, err := rangeBounds(tt.r)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if x1 != tt.x1 || z1 != tt.z1 || x2 != tt.x2 || z2 != tt.z2 {
				t.Errorf("got (%d,%d)-(%d,%d), want (%d,%d)-(%d,%d)", x1, z1, x2, z2, tt.x1, tt.z1, tt.x2, tt.z2)
			}
		})
	}
}

func TestFindRegionDir(t *testing.T) {
	mk := func(t *testing.T, dirs ...string) string {
		t.Helper()
		w := t.TempDir()
		for _, d := range dirs {
			if err := os.MkdirAll(filepath.Join(w, d), 0755); err != nil {
				t.Fatal(err)
			}
		}
		return w
	}

	t.Run("namespaced layout", func(t *testing.T) {
		w := mk(t, "dimensions/minecraft/overworld/region", "dimensions/minecraft/the_nether/region")
		for dim, want := range map[string]string{
			"overworld": "dimensions/minecraft/overworld/region",
			"nether":    "dimensions/minecraft/the_nether/region",
		} {
			got, ok := findRegionDir(w, dim)
			if !ok || got != filepath.Join(w, want) {
				t.Errorf("%s: got (%q, %v), want %q", dim, got, ok, want)
			}
		}
		if _, ok := findRegionDir(w, "end"); ok {
			t.Error("end should not exist")
		}
	})

	t.Run("legacy layout", func(t *testing.T) {
		w := mk(t, "region", "DIM-1/region", "DIM1/region")
		for dim, want := range map[string]string{
			"overworld": "region",
			"nether":    "DIM-1/region",
			"end":       "DIM1/region",
		} {
			got, ok := findRegionDir(w, dim)
			if !ok || got != filepath.Join(w, want) {
				t.Errorf("%s: got (%q, %v), want %q", dim, got, ok, want)
			}
		}
	})

	t.Run("namespaced wins over legacy", func(t *testing.T) {
		w := mk(t, "region", "dimensions/minecraft/overworld/region")
		got, _ := findRegionDir(w, "overworld")
		if got != filepath.Join(w, "dimensions/minecraft/overworld/region") {
			t.Errorf("got %q", got)
		}
	})

	t.Run("missing", func(t *testing.T) {
		if _, ok := findRegionDir(t.TempDir(), "overworld"); ok {
			t.Error("expected not found")
		}
	})
}
