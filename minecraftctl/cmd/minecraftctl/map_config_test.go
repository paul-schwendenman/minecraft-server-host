package main

import (
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
