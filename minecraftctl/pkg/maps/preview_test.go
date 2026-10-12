package maps

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/paul/minecraftctl/pkg/config"
	"github.com/paul/minecraftctl/pkg/lock"
	"github.com/paul/minecraftctl/pkg/nbt"
	"github.com/spf13/viper"
)

// previewWorldConfig has a map per dimension, plus a second overworld map.
const previewWorldConfig = `defaults:
  imageformat: webp
maps:
  - name: overworld
    dimension: overworld
  - name: caves
    dimension: overworld
    output_subdir: cave
    options:
      topY: 40
  - name: nether
    dimension: nether
  - name: end
    dimension: end
`

// previewFixture lays out world "w" with the sample level.dat, region dirs
// for the overworld, nether (two region files) and end, and the config above.
func previewFixture(t *testing.T) (b *Builder, worldDir string) {
	t.Helper()
	worldsDir := t.TempDir()
	mapsDir := t.TempDir()
	worldDir = filepath.Join(worldsDir, "w", "world")
	for _, d := range []string{"region", "DIM-1/region", "DIM1/region"} {
		if err := os.MkdirAll(filepath.Join(worldDir, d), 0755); err != nil {
			t.Fatal(err)
		}
	}
	writeFile(t, filepath.Join(worldDir, "DIM-1", "region", "r.-1.2.mca"), 1)
	writeFile(t, filepath.Join(worldDir, "DIM-1", "region", "r.1.0.mca"), 1)
	writeFile(t, filepath.Join(worldDir, "DIM-1", "region", "notes.txt"), 1)
	copyLevelDat(t, filepath.Join(worldDir, "level.dat"))
	if err := os.WriteFile(filepath.Join(worldsDir, "w", "map-config.yml"), []byte(previewWorldConfig), 0644); err != nil {
		t.Fatal(err)
	}
	return &Builder{unminedPath: "unmined-cli", worldsDir: worldsDir, mapsDir: mapsDir}, worldDir
}

func copyLevelDat(t *testing.T, dst string) {
	t.Helper()
	src, err := os.Open(filepath.Join("..", "..", "testdata", "default", "world", "level.dat"))
	if err != nil {
		t.Fatal(err)
	}
	defer src.Close()
	out, err := os.Create(dst)
	if err != nil {
		t.Fatal(err)
	}
	defer out.Close()
	if _, err := io.Copy(out, src); err != nil {
		t.Fatal(err)
	}
}

// previewUnmined fakes `unmined-cli image render`: it records the args of
// each call and writes a file at --output. Calls whose --dimension is in
// fail return an error instead.
type previewUnmined struct {
	calls [][]string
	fail  map[string]bool
}

func (f *previewUnmined) install(t *testing.T) {
	t.Helper()
	orig := runUnmined
	t.Cleanup(func() { runUnmined = orig })
	runUnmined = func(_ string, args []string) error {
		f.calls = append(f.calls, args)
		if f.fail[argValue(args, "--dimension")] {
			return errors.New("render failed")
		}
		return os.WriteFile(argValue(args, "--output"), []byte("png:"+argValue(args, "--dimension")), 0644)
	}
}

func argValue(args []string, flag string) string {
	for i, a := range args {
		if a == flag && i+1 < len(args) {
			return args[i+1]
		}
	}
	return ""
}

func TestGeneratePreviewCenters(t *testing.T) {
	b, worldDir := previewFixture(t)
	f := &previewUnmined{}
	f.install(t)

	info, err := nbt.ReadLevelDat(filepath.Join(worldDir, "level.dat"))
	if err != nil {
		t.Fatal(err)
	}
	sx, sz := int(info.GetSpawnX()), int(info.GetSpawnZ())

	tests := []struct {
		mapName, area string
	}{
		// Overworld: 64 blocks around spawn
		{"overworld", fmt.Sprintf("b((%d,%d),(%d,%d))", sx-64, sz-64, sx+64, sz+64)},
		// Nether: average of the region centers, (-256,1280) and (768,256)
		{"nether", "b((192,704),(320,832))"},
		// End: the main island at 0,0
		{"end", "b((-64,-64),(64,64))"},
	}
	for _, tt := range tests {
		if err := b.GeneratePreview("w", tt.mapName, ""); err != nil {
			t.Fatalf("GeneratePreview(%s) = %v", tt.mapName, err)
		}
		args := f.calls[len(f.calls)-1]
		if got := argValue(args, "--area"); got != tt.area {
			t.Errorf("%s: --area = %s, want %s", tt.mapName, got, tt.area)
		}
		if got := argValue(args, "--log-level"); got != "warning" {
			t.Errorf("%s: --log-level = %s, want the warning default", tt.mapName, got)
		}
		if args[0] != "image" || args[1] != "render" || argValue(args, "--zoom") != "2" {
			t.Errorf("%s: args = %v", tt.mapName, args)
		}
		want := filepath.Join(b.mapsDir, "w", tt.mapName, "preview.png")
		if got := argValue(args, "--output"); got != want {
			t.Errorf("%s: --output = %s, want %s", tt.mapName, got, want)
		}
	}

	// output_subdir and map options are honoured
	if err := b.GeneratePreview("w", "caves", "debug"); err != nil {
		t.Fatal(err)
	}
	args := f.calls[len(f.calls)-1]
	if got := argValue(args, "--output"); got != filepath.Join(b.mapsDir, "w", "cave", "preview.png") {
		t.Errorf("caves: --output = %s", got)
	}
	if got := argValue(args, "--log-level"); got != "debug" {
		t.Errorf("caves: --log-level = %s, want debug", got)
	}
	if !strings.Contains(strings.Join(args, " "), "--topY") {
		t.Errorf("caves: map options not passed: %v", args)
	}
}

func TestGeneratePreviewErrors(t *testing.T) {
	b, worldDir := previewFixture(t)
	f := &previewUnmined{fail: map[string]bool{"overworld": true}}
	f.install(t)

	if err := b.GeneratePreview("w", "nope", ""); err == nil || !strings.Contains(err.Error(), "not found in map-config.yml") {
		t.Errorf("unknown map: %v", err)
	}
	if err := b.GeneratePreview("w", "overworld", ""); err == nil || !strings.Contains(err.Error(), "failed to render preview") {
		t.Errorf("render failure: %v", err)
	}
	if err := b.GeneratePreview("missing", "overworld", ""); err == nil || !strings.Contains(err.Error(), "map config") {
		t.Errorf("missing world: %v", err)
	}

	// Nether with a region dir but no region files
	for _, e := range []string{"r.-1.2.mca", "r.1.0.mca"} {
		if err := os.Remove(filepath.Join(worldDir, "DIM-1", "region", e)); err != nil {
			t.Fatal(err)
		}
	}
	if err := b.GeneratePreview("w", "nether", ""); err == nil || !strings.Contains(err.Error(), "no region files") {
		t.Errorf("empty nether: %v", err)
	}

	if err := os.RemoveAll(filepath.Join(worldDir, "DIM1")); err != nil {
		t.Fatal(err)
	}
	if err := b.GeneratePreview("w", "end", ""); err == nil || !strings.Contains(err.Error(), "no region data") {
		t.Errorf("no end region data: %v", err)
	}

	if err := os.Remove(filepath.Join(worldDir, "level.dat")); err != nil {
		t.Fatal(err)
	}
	if err := b.GeneratePreview("w", "caves", ""); err == nil || !strings.Contains(err.Error(), "level.dat") {
		t.Errorf("missing level.dat: %v", err)
	}
}

func TestCalculateRegionCenterMissingDir(t *testing.T) {
	if _, _, err := calculateRegionCenter(filepath.Join(t.TempDir(), "nope")); err == nil {
		t.Error("calculateRegionCenter(missing dir) = nil error")
	}
}

func TestBuildManifestsWithPreviews(t *testing.T) {
	b, worldDir := previewFixture(t)
	// Nobody has been to the End
	if err := os.RemoveAll(filepath.Join(worldDir, "DIM1")); err != nil {
		t.Fatal(err)
	}
	f := &previewUnmined{fail: map[string]bool{"nether": true}}
	f.install(t)
	mb := &ManifestBuilder{worldsDir: b.worldsDir, mapsDir: b.mapsDir, builder: b}

	if err := mb.BuildManifests("w", ManifestOptions{GeneratePreviews: true}); err != nil {
		t.Fatalf("BuildManifests() = %v", err)
	}
	// The end has no region data: no preview render, no manifest
	if len(f.calls) != 3 {
		t.Errorf("preview renders = %d, want 3", len(f.calls))
	}
	worldMaps := filepath.Join(b.mapsDir, "w")
	if exists(filepath.Join(worldMaps, "end")) {
		t.Error("map dir created for a dimension with no data")
	}
	// A failed preview doesn't stop the map's manifest
	if m := readTestManifest(t, filepath.Join(worldMaps, "nether")); m.Path != "w/nether" {
		t.Errorf("nether manifest path = %q", m.Path)
	}
	if m := readTestManifest(t, filepath.Join(worldMaps, "cave")); m.Map != "caves" {
		t.Errorf("caves manifest = %+v", m)
	}

	var wm WorldManifest
	data, err := os.ReadFile(filepath.Join(worldMaps, "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &wm); err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, m := range wm.Maps {
		names = append(names, m["name"])
	}
	if got := strings.Join(names, ","); got != "overworld,caves,nether" {
		t.Errorf("world manifest maps = %s", got)
	}
	if wm.World != "w" || wm.Preview != "w/preview.png" || wm.Version == "" || wm.DifficultyName == "" {
		t.Errorf("world manifest = %+v", wm)
	}
	// The world preview is the first overworld map's
	preview, err := os.ReadFile(filepath.Join(worldMaps, "preview.png"))
	if err != nil || string(preview) != "png:overworld" {
		t.Errorf("world preview = %q, %v", preview, err)
	}
}

func TestBuildManifestsPreviewOnly(t *testing.T) {
	b, _ := previewFixture(t)
	f := &previewUnmined{}
	f.install(t)
	mb := &ManifestBuilder{worldsDir: b.worldsDir, mapsDir: b.mapsDir, builder: b}

	if err := mb.BuildManifests("ignored", ManifestOptions{WorldName: "w", PreviewOnly: true}); err != nil {
		t.Fatalf("BuildManifests() = %v", err)
	}
	if len(f.calls) != 4 {
		t.Errorf("preview renders = %d, want 4", len(f.calls))
	}
	worldMaps := filepath.Join(b.mapsDir, "w")
	for _, p := range []string{"manifest.json", "overworld/manifest.json", "preview.png"} {
		if exists(filepath.Join(worldMaps, p)) {
			t.Errorf("preview-only wrote %s", p)
		}
	}
	if !exists(filepath.Join(worldMaps, "overworld", "preview.png")) {
		t.Error("overworld preview missing")
	}
}

func TestBuildManifestsNoPreviews(t *testing.T) {
	b, _ := previewFixture(t)
	f := &previewUnmined{}
	f.install(t)
	mb := &ManifestBuilder{worldsDir: b.worldsDir, mapsDir: b.mapsDir, builder: b}

	if err := mb.BuildManifests("w", ManifestOptions{}); err != nil {
		t.Fatalf("BuildManifests() = %v", err)
	}
	if len(f.calls) != 0 {
		t.Errorf("renders = %d, want none", len(f.calls))
	}
	// No overworld preview to copy: the world manifest is still written
	if !exists(filepath.Join(b.mapsDir, "w", "manifest.json")) {
		t.Error("world manifest missing")
	}
	if err := mb.BuildManifests("missing", ManifestOptions{}); err == nil {
		t.Error("BuildManifests(missing world) = nil error")
	}
}

func TestBuildAggregateIndex(t *testing.T) {
	mapsDir := t.TempDir()
	write := func(rel, content string) {
		t.Helper()
		p := filepath.Join(mapsDir, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
	write("survival/manifest.json", `{"world":"survival","version":"1.21.4"}`)
	write("survival/overworld/manifest.json", `{}`)
	write("survival/nether/manifest.json", `{}`)
	write("creative/manifest.json", `{"world":"creative"}`)
	write("creative/overworld/x", "")
	write("broken/manifest.json", `{not json`)
	write("empty/.keep", "")
	write("stray.txt", "")

	mb := &ManifestBuilder{mapsDir: mapsDir}
	if err := mb.BuildAggregateIndex(); err != nil {
		t.Fatalf("BuildAggregateIndex() = %v", err)
	}

	var agg []WorldManifest
	data, err := os.ReadFile(filepath.Join(mapsDir, "world_manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &agg); err != nil {
		t.Fatal(err)
	}
	if len(agg) != 2 || agg[0].World != "creative" || agg[1].World != "survival" || agg[1].Version != "1.21.4" {
		t.Errorf("aggregate = %+v, want creative and survival", agg)
	}

	html, err := os.ReadFile(filepath.Join(mapsDir, "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	s := string(html)
	for _, want := range []string{
		"<h2>survival</h2>",
		"<li><a href='./survival/nether/'>nether</a></li>\n<li><a href='./survival/overworld/'>overworld</a></li>",
		"<a href='./creative/overworld/'>",
		"<h2>empty</h2><ul>\n</ul>",
	} {
		if !strings.Contains(s, want) {
			t.Errorf("index.html missing %q:\n%s", want, s)
		}
	}
	if strings.Index(s, "<h2>creative</h2>") > strings.Index(s, "<h2>survival</h2>") {
		t.Error("worlds not sorted in index.html")
	}

	mb = &ManifestBuilder{mapsDir: filepath.Join(mapsDir, "missing")}
	if err := mb.BuildAggregateIndex(); err == nil {
		t.Error("BuildAggregateIndex(missing dir) = nil error")
	}
}

func TestCopyFileErrors(t *testing.T) {
	dir := t.TempDir()
	if err := copyFile(filepath.Join(dir, "missing"), filepath.Join(dir, "out")); err == nil {
		t.Error("copyFile(missing src) = nil error")
	}
	src := filepath.Join(dir, "src")
	writeFile(t, src, 3)
	// The destination's parent is a file
	if err := copyFile(src, filepath.Join(src, "out")); err == nil {
		t.Error("copyFile(dst under a file) = nil error")
	}
	if err := copyFile(src, filepath.Join(dir, "a", "b", "out")); err != nil {
		t.Errorf("copyFile() = %v", err)
	}
}

func TestNewBuildersUseConfig(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)
	viper.Set("worlds_dir", "/tmp/worlds-x")
	viper.Set("maps_dir", "/tmp/maps-x")
	if err := config.Init(""); err != nil {
		t.Fatal(err)
	}
	mb := NewManifestBuilder()
	if mb.worldsDir != "/tmp/worlds-x" || mb.mapsDir != "/tmp/maps-x" || !mb.generatePreviews {
		t.Errorf("NewManifestBuilder() = %+v", mb)
	}
	if mb.builder.unminedPath != DefaultUnminedPath || mb.builder.mapsDir != "/tmp/maps-x" {
		t.Errorf("builder = %+v", mb.builder)
	}
}

func TestBuildTakesLock(t *testing.T) {
	b, _ := previewFixture(t)
	if err := os.WriteFile(filepath.Join(b.worldsDir, "w", "map-config.yml"),
		[]byte("maps:\n  - name: overworld\n    dimension: overworld\n"), 0644); err != nil {
		t.Fatal(err)
	}
	f := &recordingUnmined{}
	f.install(t)
	lockPath := filepath.Join(t.TempDir(), "map.lock")

	// Held by someone else: a non-blocking build skips quietly
	held := lock.NewFileLock(lockPath)
	if err := held.Lock(); err != nil {
		t.Fatal(err)
	}
	if err := b.Build(BuildOptions{WorldName: "w", LockFile: lockPath, NonBlocking: true}); err != nil {
		t.Errorf("non-blocking Build with the lock held = %v, want nil", err)
	}
	if len(f.calls) != 0 {
		t.Errorf("renders while locked = %v", f.calls)
	}
	if err := held.Unlock(); err != nil {
		t.Fatal(err)
	}

	if err := b.Build(BuildOptions{WorldName: "w", LockFile: lockPath}); err != nil {
		t.Fatalf("Build() = %v", err)
	}
	if len(f.calls) != 1 || f.calls[0] != "base" {
		t.Errorf("renders = %v, want [base]", f.calls)
	}
	if m := readTestManifest(t, filepath.Join(b.mapsDir, "w", "overworld")); m.LastRenderedEpoch == 0 {
		t.Errorf("manifest after build = %+v", m)
	}
	// The lock was released
	if err := held.LockWithOptions(lock.LockOptions{NonBlocking: true}); err != nil {
		t.Errorf("lock still held after Build: %v", err)
	}
	_ = held.Unlock()
}
