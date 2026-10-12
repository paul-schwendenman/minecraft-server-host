package maps

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/paul/minecraftctl/pkg/config"
	"github.com/paul/minecraftctl/pkg/lock"
)

func writeFile(t *testing.T, path string, size int) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, make([]byte, size), 0644); err != nil {
		t.Fatal(err)
	}
}

func writeProperties(t *testing.T, mapOutput, format string) {
	t.Helper()
	if err := os.MkdirAll(mapOutput, 0755); err != nil {
		t.Fatal(err)
	}
	props := `UnminedMapProperties = { imageFormat: "` + format + `", maxZoom: 1 };`
	if err := os.WriteFile(filepath.Join(mapOutput, "unmined.map.properties.js"), []byte(props), 0644); err != nil {
		t.Fatal(err)
	}
}

func exists(path string) bool {
	_, err := os.Lstat(path)
	return err == nil
}

// cleanFixture lays out a world with three maps in map-config.yml:
//   - overworld: rendered as webp, with leftover jpeg (and one .jpg) tiles
//   - caves (output_subdir cave): rendered as png, nothing stale
//   - nether: never rendered (no properties file), holds a jpeg tile
//
// plus an orphan map dir and world-level files
func cleanFixture(t *testing.T) (b *Builder, worldMaps string) {
	t.Helper()
	worldsDir := t.TempDir()
	mapsDir := t.TempDir()

	cfg := `defaults:
  imageformat: webp
maps:
  - name: overworld
    dimension: overworld
  - name: caves
    dimension: overworld
    output_subdir: cave
  - name: nether
    dimension: nether
`
	writeFile(t, filepath.Join(worldsDir, "w", "world", "level.dat"), 1)
	if err := os.WriteFile(filepath.Join(worldsDir, "w", "map-config.yml"), []byte(cfg), 0644); err != nil {
		t.Fatal(err)
	}

	worldMaps = filepath.Join(mapsDir, "w")
	ow := filepath.Join(worldMaps, "overworld")
	writeProperties(t, ow, "webp")
	writeFile(t, filepath.Join(ow, "manifest.json"), 10)
	writeFile(t, filepath.Join(ow, "preview.png"), 10)
	writeFile(t, filepath.Join(ow, "index.html"), 10)
	writeFile(t, filepath.Join(ow, "tiles", "zoom.0", "0", "0", "tile.0.0.webp"), 5)
	writeFile(t, filepath.Join(ow, "tiles", "zoom.0", "0", "0", "tile.0.0.jpeg"), 100)
	writeFile(t, filepath.Join(ow, "tiles", "zoom.0", "0", "0", "tile.1.0.jpg"), 50)
	writeFile(t, filepath.Join(ow, "tiles", "zoom.-1", "0", "0", "tile.0.0.jpeg"), 200)
	writeFile(t, filepath.Join(ow, "tiles", "zoom.0", "0", "0", "notes.txt"), 7)

	cave := filepath.Join(worldMaps, "cave")
	writeProperties(t, cave, "png")
	writeFile(t, filepath.Join(cave, "tiles", "zoom.0", "0", "0", "tile.0.0.png"), 5)

	writeFile(t, filepath.Join(worldMaps, "nether", "tiles", "zoom.0", "0", "0", "tile.0.0.jpeg"), 9)

	writeFile(t, filepath.Join(worldMaps, "old_end", "tiles", "zoom.0", "0", "0", "tile.0.0.png"), 30)
	writeFile(t, filepath.Join(worldMaps, "old_end", "manifest.json"), 3)
	writeFile(t, filepath.Join(worldMaps, "manifest.json"), 10)
	writeFile(t, filepath.Join(worldMaps, "preview.png"), 10)

	return &Builder{worldsDir: worldsDir, mapsDir: mapsDir}, worldMaps
}

// snapshot lists every path under dir
func snapshot(t *testing.T, dir string) []string {
	t.Helper()
	var paths []string
	filepath.Walk(dir, func(p string, _ os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		paths = append(paths, p)
		return nil
	})
	return paths
}

func TestCleanDryRunRemovesNothing(t *testing.T) {
	b, worldMaps := cleanFixture(t)
	before := snapshot(t, worldMaps)

	res, err := b.Clean(CleanOptions{LockFile: testLockFile(t), WorldName: "w", DryRun: true})
	if err != nil {
		t.Fatalf("Clean: %v", err)
	}
	if after := snapshot(t, worldMaps); !slices.Equal(before, after) {
		t.Errorf("dry run changed the tree:\nbefore %v\nafter  %v", before, after)
	}

	if len(res.Maps) != 3 {
		t.Fatalf("got %d map results, want 3", len(res.Maps))
	}
	ow, cave, nether := res.Maps[0], res.Maps[1], res.Maps[2]
	if ow.Files != 3 || ow.Bytes != 350 || ow.Format != "webp" {
		t.Errorf("overworld = %+v, want 3 files, 350 bytes, webp", ow)
	}
	if cave.Files != 0 || cave.Skipped != "" {
		t.Errorf("caves = %+v, want nothing stale", cave)
	}
	if nether.Skipped == "" || nether.Files != 0 {
		t.Errorf("nether = %+v, want skipped (format unknown)", nether)
	}
	if len(res.Orphans) != 1 || filepath.Base(res.Orphans[0].Dir) != "old_end" ||
		res.Orphans[0].Files != 2 || res.Orphans[0].Bytes != 33 {
		t.Errorf("orphans = %+v, want old_end with 2 files, 33 bytes", res.Orphans)
	}
	if res.TotalFiles() != 5 || res.TotalBytes() != 383 {
		t.Errorf("totals = %d files, %d bytes; want 5, 383", res.TotalFiles(), res.TotalBytes())
	}
}

func TestCleanRemovesStaleTilesAndOrphans(t *testing.T) {
	b, worldMaps := cleanFixture(t)

	res, err := b.Clean(CleanOptions{LockFile: testLockFile(t), WorldName: "w"})
	if err != nil {
		t.Fatalf("Clean: %v", err)
	}
	if res.TotalFiles() != 5 || res.TotalBytes() != 383 {
		t.Errorf("totals = %d files, %d bytes; want 5, 383", res.TotalFiles(), res.TotalBytes())
	}

	ow := filepath.Join(worldMaps, "overworld")
	gone := []string{
		filepath.Join(ow, "tiles", "zoom.0", "0", "0", "tile.0.0.jpeg"),
		filepath.Join(ow, "tiles", "zoom.0", "0", "0", "tile.1.0.jpg"),
		filepath.Join(ow, "tiles", "zoom.-1"), // emptied, so pruned
		filepath.Join(worldMaps, "old_end"),
	}
	kept := []string{
		filepath.Join(ow, "tiles", "zoom.0", "0", "0", "tile.0.0.webp"),
		filepath.Join(ow, "tiles", "zoom.0", "0", "0", "notes.txt"),
		filepath.Join(ow, "manifest.json"),
		filepath.Join(ow, "preview.png"),
		filepath.Join(ow, "index.html"),
		filepath.Join(ow, "unmined.map.properties.js"),
		filepath.Join(worldMaps, "cave", "tiles", "zoom.0", "0", "0", "tile.0.0.png"),
		filepath.Join(worldMaps, "nether", "tiles", "zoom.0", "0", "0", "tile.0.0.jpeg"),
		filepath.Join(worldMaps, "manifest.json"),
		filepath.Join(worldMaps, "preview.png"),
	}
	for _, p := range gone {
		if exists(p) {
			t.Errorf("%s still exists", p)
		}
	}
	for _, p := range kept {
		if !exists(p) {
			t.Errorf("%s was removed", p)
		}
	}
}

func TestCleanSingleMapSkipsOrphans(t *testing.T) {
	b, worldMaps := cleanFixture(t)

	res, err := b.Clean(CleanOptions{LockFile: testLockFile(t), WorldName: "w", MapName: "overworld"})
	if err != nil {
		t.Fatalf("Clean: %v", err)
	}
	if len(res.Maps) != 1 || res.Maps[0].Files != 3 || len(res.Orphans) != 0 {
		t.Errorf("result = %+v, want overworld only", res)
	}
	if !exists(filepath.Join(worldMaps, "old_end")) {
		t.Error("orphan dir removed when cleaning a single map")
	}

	if _, err := b.Clean(CleanOptions{LockFile: testLockFile(t), WorldName: "w", MapName: "missing"}); err == nil {
		t.Error("expected an error for a map not in map-config.yml")
	}
}

func TestOrphanMapDirsKeepsNestedOutputSubdir(t *testing.T) {
	dir := t.TempDir()
	for _, d := range []string{"overworld", "extra", "stale"} {
		os.MkdirAll(filepath.Join(dir, d), 0755)
	}
	writeFile(t, filepath.Join(dir, "manifest.json"), 1)
	defs := []config.MapDefinition{
		{Name: "overworld"},
		{Name: "caves", OutputSubdir: "extra/caves"},
	}
	got, err := orphanMapDirs(dir, defs)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(got, []string{filepath.Join(dir, "stale")}) {
		t.Errorf("orphans = %v, want [stale]", got)
	}
}

func TestRemoveStaleTilesUnknownFormat(t *testing.T) {
	if _, _, err := removeStaleTiles(t.TempDir(), "tiff", false); err == nil {
		t.Error("expected an error for an unknown keep format")
	}
}

// fakeUnmined stands in for uNmINeD: a render writes a tile in the requested
// format and, like the real thing, rewrites the properties file. Range renders
// (those with --area) fail when failRanges is set.
func fakeUnmined(t *testing.T, failRanges bool) {
	t.Helper()
	orig := runUnmined
	t.Cleanup(func() { runUnmined = orig })
	runUnmined = func(_ string, args []string) error {
		var output, format string
		isRange := false
		for i, a := range args {
			switch {
			case a == "--output":
				output = args[i+1]
			case a == "--imageformat":
				format = args[i+1]
			case strings.HasPrefix(a, "--area"):
				isRange = true
			}
		}
		if isRange && failRanges {
			return errors.New("range render failed")
		}
		writeFile(t, filepath.Join(output, "tiles", "zoom.0", "0", "0", "tile.0.0."+format), 4)
		writeProperties(t, output, format)
		return nil
	}
}

func formatChangeFixture(t *testing.T) (b *Builder, worldDir, worldMaps, oldTile string) {
	t.Helper()
	worldsDir := t.TempDir()
	mapsDir := t.TempDir()
	worldDir = filepath.Join(worldsDir, "w", "world")
	if err := os.MkdirAll(filepath.Join(worldDir, "region"), 0755); err != nil {
		t.Fatal(err)
	}
	worldMaps = filepath.Join(mapsDir, "w")
	ow := filepath.Join(worldMaps, "overworld")
	writeProperties(t, ow, "jpeg")
	oldTile = filepath.Join(ow, "tiles", "zoom.0", "0", "0", "tile.0.0.jpeg")
	writeFile(t, oldTile, 10)
	return &Builder{unminedPath: "unmined", worldsDir: worldsDir, mapsDir: mapsDir}, worldDir, worldMaps, oldTile
}

func TestBuildMapFormatChangeRemovesOldTiles(t *testing.T) {
	fakeUnmined(t, false)
	b, worldDir, worldMaps, oldTile := formatChangeFixture(t)

	mapDef := config.MapDefinition{
		Name: "overworld", Dimension: "overworld",
		Ranges: []config.MapRange{{Name: "spawn", Radius: 100}},
	}
	defaults := config.MapDefaults{ImageFormat: "webp", ChunkProcessors: 1}
	if err := b.buildMap(mapDef, defaults, worldDir, worldMaps, 0, BuildOptions{WorldName: "w"}); err != nil {
		t.Fatalf("buildMap: %v", err)
	}

	if exists(oldTile) {
		t.Error("old jpeg tile still exists after a successful full render")
	}
	ow := filepath.Join(worldMaps, "overworld")
	for _, p := range []string{
		filepath.Join(ow, "tiles", "zoom.0", "0", "0", "tile.0.0.webp"),
		filepath.Join(ow, "manifest.json"),
		filepath.Join(ow, "unmined.map.properties.js"),
	} {
		if !exists(p) {
			t.Errorf("%s missing after build", p)
		}
	}
}

func TestBuildMapFormatChangeKeepsOldTilesWhenRangeFails(t *testing.T) {
	fakeUnmined(t, true)
	b, worldDir, worldMaps, oldTile := formatChangeFixture(t)

	mapDef := config.MapDefinition{
		Name: "overworld", Dimension: "overworld",
		Ranges: []config.MapRange{{Name: "spawn", Radius: 100}},
	}
	defaults := config.MapDefaults{ImageFormat: "webp", ChunkProcessors: 1}
	if err := b.buildMap(mapDef, defaults, worldDir, worldMaps, 0, BuildOptions{WorldName: "w"}); err != nil {
		t.Fatalf("buildMap: %v", err)
	}
	if !exists(oldTile) {
		t.Error("old jpeg tile removed although a range failed to render")
	}
}

func TestBuildMapFormatChangeKeepsOldTilesWhenBaseFails(t *testing.T) {
	b, worldDir, worldMaps, oldTile := formatChangeFixture(t)
	orig := runUnmined
	t.Cleanup(func() { runUnmined = orig })
	runUnmined = func(string, []string) error { return errors.New("boom") }

	mapDef := config.MapDefinition{Name: "overworld", Dimension: "overworld"}
	defaults := config.MapDefaults{ImageFormat: "webp", ChunkProcessors: 1}
	if err := b.buildMap(mapDef, defaults, worldDir, worldMaps, 0, BuildOptions{WorldName: "w"}); err == nil {
		t.Fatal("expected buildMap to fail")
	}
	if !exists(oldTile) {
		t.Error("old jpeg tile removed although the base render failed")
	}
}

func TestBuildMapSameFormatKeepsOtherTiles(t *testing.T) {
	fakeUnmined(t, false)
	b, worldDir, worldMaps, oldTile := formatChangeFixture(t)

	// No format change: the build mustn't clean anything up on its own
	mapDef := config.MapDefinition{Name: "overworld", Dimension: "overworld"}
	stray := filepath.Join(worldMaps, "overworld", "tiles", "zoom.0", "0", "0", "tile.9.9.png")
	writeFile(t, stray, 1)
	defaults := config.MapDefaults{ImageFormat: "jpeg", ChunkProcessors: 1}
	if err := b.buildMap(mapDef, defaults, worldDir, worldMaps, 0, BuildOptions{WorldName: "w"}); err != nil {
		t.Fatalf("buildMap: %v", err)
	}
	if !exists(oldTile) || !exists(stray) {
		t.Error("tiles removed although the format didn't change")
	}
}

func testLockFile(t *testing.T) string {
	t.Helper()
	return filepath.Join(t.TempDir(), "map.lock")
}

// With output_subdir "." the map renders into the world's maps dir itself, so
// its tiles/ (and every other dir there) looks like an orphan. Clean must
// leave directories alone but still clean that map's own tiles/.
func TestCleanRootOutputSubdirSkipsOrphans(t *testing.T) {
	for _, subdir := range []string{".", "x/.."} {
		t.Run(subdir, func(t *testing.T) {
			b, worldMaps := cleanFixture(t)
			cfg := "maps:\n  - name: overworld\n    dimension: overworld\n  - name: root\n    dimension: overworld\n    output_subdir: " + subdir + "\n"
			if err := os.WriteFile(filepath.Join(b.worldsDir, "w", "map-config.yml"), []byte(cfg), 0644); err != nil {
				t.Fatal(err)
			}
			writeProperties(t, worldMaps, "webp")
			rootTile := filepath.Join(worldMaps, "tiles", "zoom.0", "0", "0", "tile.0.0.webp")
			rootOld := filepath.Join(worldMaps, "tiles", "zoom.0", "0", "0", "tile.0.0.jpeg")
			writeFile(t, rootTile, 4)
			writeFile(t, rootOld, 6)

			res, err := b.Clean(CleanOptions{LockFile: testLockFile(t), WorldName: "w"})
			if err != nil {
				t.Fatalf("Clean: %v", err)
			}
			if len(res.Orphans) != 0 || res.OrphansSkipped == "" {
				t.Errorf("orphans = %v, skipped = %q; want none removed and a reason", res.Orphans, res.OrphansSkipped)
			}
			for _, p := range []string{rootTile, filepath.Join(worldMaps, "old_end"), filepath.Join(worldMaps, "nether"), filepath.Join(worldMaps, "cave")} {
				if !exists(p) {
					t.Errorf("%s removed", p)
				}
			}
			if exists(rootOld) {
				t.Error("root map's stale jpeg tile kept")
			}
		})
	}
}

func TestUnsafeOrphanCheck(t *testing.T) {
	dir := t.TempDir()
	for _, tt := range []struct {
		subdir string
		unsafe bool
	}{
		{"overworld", false},
		{"extra/caves", false},
		{"/abs", false}, // joined under the maps dir
		{".", true},
		{"a/..", true},
		{"..", true},
		{"../other", true},
	} {
		got := unsafeOrphanCheck(dir, []config.MapDefinition{{Name: "m", OutputSubdir: tt.subdir}})
		if (got != "") != tt.unsafe {
			t.Errorf("output_subdir %q: reason = %q, want unsafe=%v", tt.subdir, got, tt.unsafe)
		}
	}
	// An empty output_subdir falls back to the map name
	if got := unsafeOrphanCheck(dir, []config.MapDefinition{{Name: "m"}}); got != "" {
		t.Errorf("empty output_subdir: reason = %q", got)
	}
}

func TestOrphanMapDirsAbsoluteOutputSubdir(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "abs"), 0755)
	got, err := orphanMapDirs(dir, []config.MapDefinition{{Name: "m", OutputSubdir: "/abs"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Errorf("orphans = %v, want none (/abs renders to <maps>/abs)", got)
	}
}

// A running build holds the lock and may have a range's properties file in
// place: Clean must not read or delete anything, dry run included.
func TestCleanFailsWhileBuildHoldsLock(t *testing.T) {
	for _, dryRun := range []bool{false, true} {
		b, worldMaps := cleanFixture(t)
		before := snapshot(t, worldMaps)
		lockPath := testLockFile(t)
		held := lock.NewFileLock(lockPath)
		if err := held.Lock(); err != nil {
			t.Fatal(err)
		}

		res, err := b.Clean(CleanOptions{LockFile: lockPath, WorldName: "w", DryRun: dryRun})
		if err == nil || !strings.Contains(err.Error(), "map build is running") {
			t.Errorf("dryRun=%v: Clean err = %v, want a lock error", dryRun, err)
		}
		if res != nil {
			t.Errorf("dryRun=%v: result = %+v, want nil", dryRun, res)
		}
		if after := snapshot(t, worldMaps); !slices.Equal(before, after) {
			t.Errorf("dryRun=%v: files changed while the lock was held", dryRun)
		}
		_ = held.Unlock()

		// Released: Clean runs and lets go of the lock again
		if _, err := b.Clean(CleanOptions{LockFile: lockPath, WorldName: "w", DryRun: dryRun}); err != nil {
			t.Errorf("dryRun=%v: Clean after unlock: %v", dryRun, err)
		}
		if err := held.LockWithOptions(lock.LockOptions{NonBlocking: true}); err != nil {
			t.Errorf("lock still held after Clean: %v", err)
		}
		_ = held.Unlock()
	}
}
