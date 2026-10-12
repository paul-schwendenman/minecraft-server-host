package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMapConfigCatCmd(t *testing.T) {
	t.Run("prints raw file verbatim", func(t *testing.T) {
		dir := t.TempDir()
		setupWorldTestConfig(t, dir)

		worldDir := createTestWorld(t, dir, "myworld", true)
		raw := "# hand-written comment\ndefaults:\n  zoomout:   4\nmaps:\n  - name: overworld\n"
		if err := os.WriteFile(filepath.Join(worldDir, "map-config.yml"), []byte(raw), 0644); err != nil {
			t.Fatalf("failed to write map-config.yml: %v", err)
		}

		output, err := captureWorldStdout(t, func() error {
			return mapConfigCatCmd.RunE(mapConfigCatCmd, []string{"myworld"})
		})
		if err != nil {
			t.Fatalf("mapConfigCatCmd.RunE() failed: %v", err)
		}
		if output != raw {
			t.Errorf("output = %q, want %q", output, raw)
		}
	})

	t.Run("errors when map-config.yml is missing", func(t *testing.T) {
		dir := t.TempDir()
		setupWorldTestConfig(t, dir)

		createTestWorld(t, dir, "nomap", false)

		_, err := captureWorldStdout(t, func() error {
			return mapConfigCatCmd.RunE(mapConfigCatCmd, []string{"nomap"})
		})
		if err == nil || !strings.Contains(err.Error(), "map-config.yml not found") {
			t.Errorf("expected 'not found' error, got %v", err)
		}
	})

	t.Run("show is an alias", func(t *testing.T) {
		found, _, err := mapConfigCmd.Find([]string{"show"})
		if err != nil {
			t.Fatalf("Find(show) failed: %v", err)
		}
		if found != mapConfigCatCmd {
			t.Errorf("Find(show) = %q, want cat", found.Name())
		}
	})
}

func TestMapConfigGetCmdJSON(t *testing.T) {
	dir := t.TempDir()
	setupWorldTestConfig(t, dir)

	worldDir := createTestWorld(t, dir, "myworld", true)
	raw := `defaults:
  zoomout: 4
  imageformat: webp
maps:
  - name: overworld
    dimension: overworld
    output_subdir: overworld
    options:
      shadows: true
    ranges:
      - name: spawn
        center: [10, -20]
        radius: 64
`
	if err := os.WriteFile(filepath.Join(worldDir, "map-config.yml"), []byte(raw), 0644); err != nil {
		t.Fatalf("failed to write map-config.yml: %v", err)
	}

	if err := mapConfigGetCmd.Flags().Set("format", "json"); err != nil {
		t.Fatalf("set format: %v", err)
	}
	t.Cleanup(func() { _ = mapConfigGetCmd.Flags().Set("format", "yaml") })

	get := func(t *testing.T, args ...string) interface{} {
		t.Helper()
		output, err := captureWorldStdout(t, func() error {
			return mapConfigGetCmd.RunE(mapConfigGetCmd, append([]string{"myworld"}, args...))
		})
		if err != nil {
			t.Fatalf("mapConfigGetCmd.RunE(%v) failed: %v", args, err)
		}
		if !strings.HasSuffix(output, "\n") {
			t.Errorf("output %q has no trailing newline", output)
		}
		var v interface{}
		if err := json.Unmarshal([]byte(output), &v); err != nil {
			t.Fatalf("output is not JSON: %v\n%s", err, output)
		}
		return v
	}

	t.Run("scalar", func(t *testing.T) {
		if got := get(t, "defaults.imageformat"); got != "webp" {
			t.Errorf("got %v, want webp", got)
		}
		if got := get(t, "defaults.zoomout"); got != float64(4) {
			t.Errorf("got %v, want 4", got)
		}
	})

	t.Run("nested map uses yaml key names", func(t *testing.T) {
		got, ok := get(t, "defaults").(map[string]interface{})
		if !ok {
			t.Fatalf("got %T, want object", got)
		}
		if got["imageformat"] != "webp" {
			t.Errorf("imageformat = %v, want webp (keys: %v)", got["imageformat"], got)
		}
		if _, ok := got["ImageFormat"]; ok {
			t.Errorf("got Go field name ImageFormat, want yaml key")
		}
	})

	t.Run("list", func(t *testing.T) {
		got, ok := get(t, "maps[0].ranges").([]interface{})
		if !ok || len(got) != 1 {
			t.Fatalf("got %#v, want one range", got)
		}
		r := got[0].(map[string]interface{})
		if r["name"] != "spawn" || r["radius"] != float64(64) {
			t.Errorf("range = %v", r)
		}
		center := r["center"].([]interface{})
		if center[0] != float64(10) || center[1] != float64(-20) {
			t.Errorf("center = %v", center)
		}
	})

	t.Run("whole config omits empty fields", func(t *testing.T) {
		got := get(t).(map[string]interface{})
		m := got["maps"].([]interface{})[0].(map[string]interface{})
		opts := m["options"].(map[string]interface{})
		if _, ok := opts["topY"]; ok {
			t.Errorf("unset topY should be omitted: %v", opts)
		}
		if opts["shadows"] != true {
			t.Errorf("shadows = %v", opts)
		}
	})
}

func TestMarshalConfigValue(t *testing.T) {
	t.Run("yaml", func(t *testing.T) {
		data, err := marshalConfigValue(map[string]int{"a": 1}, "yaml")
		if err != nil || string(data) != "a: 1\n" {
			t.Errorf("got %q, %v", data, err)
		}
	})
	t.Run("json nil", func(t *testing.T) {
		data, err := marshalConfigValue(nil, "json")
		if err != nil || string(data) != "null\n" {
			t.Errorf("got %q, %v", data, err)
		}
	})
	t.Run("unsupported format", func(t *testing.T) {
		_, err := marshalConfigValue("x", "toml")
		if err == nil || !strings.Contains(err.Error(), "unsupported format: toml") {
			t.Errorf("expected unsupported format error, got %v", err)
		}
	})
}
