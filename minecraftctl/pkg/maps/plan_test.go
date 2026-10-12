package maps

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/paul/minecraftctl/pkg/config"
)

func intp(v int) *int    { return &v }
func boolp(v bool) *bool { return &v }

func TestRangeHashStable(t *testing.T) {
	base := func() (config.MapRange, config.MapDefinition, config.MapDefaults) {
		r := config.MapRange{Name: "spawn", Center: [2]int{10, -20}, Radius: 100}
		m := config.MapDefinition{Name: "overworld", Dimension: "overworld"}
		d := config.MapDefaults{Zoomout: 6, Zoomin: 2, ImageFormat: "webp", ChunkProcessors: 4}
		return r, m, d
	}
	r, m, d := base()
	want := rangeHash(r, m, d)
	r2, m2, d2 := base()
	if got := rangeHash(r2, m2, d2); got != want {
		t.Fatalf("same input gave %s and %s", want, got)
	}

	// Settings that don't change the render don't change the hash
	r2.Name = "renamed"
	d2.ChunkProcessors = 16
	m2.Options.Players = boolp(true)
	if got := rangeHash(r2, m2, d2); got != want {
		t.Errorf("name/chunkprocessors/players changed the hash")
	}
	// A map/range zoom that equals the default resolves to the same render
	m2.Zoomin = intp(2)
	if got := rangeHash(r2, m2, d2); got != want {
		t.Errorf("explicit zoomin equal to the default changed the hash")
	}

	changes := map[string]func(*config.MapRange, *config.MapDefinition, *config.MapDefaults){
		"center x":        func(r *config.MapRange, _ *config.MapDefinition, _ *config.MapDefaults) { r.Center[0] = 11 },
		"center z":        func(r *config.MapRange, _ *config.MapDefinition, _ *config.MapDefaults) { r.Center[1] = 0 },
		"radius":          func(r *config.MapRange, _ *config.MapDefinition, _ *config.MapDefaults) { r.Radius = 101 },
		"range zoomin":    func(r *config.MapRange, _ *config.MapDefinition, _ *config.MapDefaults) { r.Zoomin = intp(4) },
		"range zoomout":   func(r *config.MapRange, _ *config.MapDefinition, _ *config.MapDefaults) { r.Zoomout = intp(3) },
		"map zoomin":      func(_ *config.MapRange, m *config.MapDefinition, _ *config.MapDefaults) { m.Zoomin = intp(3) },
		"map zoomout":     func(_ *config.MapRange, m *config.MapDefinition, _ *config.MapDefaults) { m.Zoomout = intp(5) },
		"default zoomin":  func(_ *config.MapRange, _ *config.MapDefinition, d *config.MapDefaults) { d.Zoomin = 1 },
		"default zoomout": func(_ *config.MapRange, _ *config.MapDefinition, d *config.MapDefaults) { d.Zoomout = 4 },
		"imageformat":     func(_ *config.MapRange, _ *config.MapDefinition, d *config.MapDefaults) { d.ImageFormat = "jpeg" },
		"dimension":       func(_ *config.MapRange, m *config.MapDefinition, _ *config.MapDefaults) { m.Dimension = "nether" },
		"topY":            func(_ *config.MapRange, m *config.MapDefinition, _ *config.MapDefaults) { m.Options.TopY = intp(64) },
		"bottomY":         func(_ *config.MapRange, m *config.MapDefinition, _ *config.MapDefaults) { m.Options.BottomY = intp(0) },
		"gndxray": func(_ *config.MapRange, m *config.MapDefinition, _ *config.MapDefaults) {
			m.Options.GndXray = boolp(true)
		},
		"night": func(_ *config.MapRange, m *config.MapDefinition, _ *config.MapDefaults) {
			m.Options.Night = boolp(true)
		},
		"shadows":          func(_ *config.MapRange, m *config.MapDefinition, _ *config.MapDefaults) { m.Options.Shadows = "3do" },
		"shadows explicit": func(_ *config.MapRange, m *config.MapDefinition, _ *config.MapDefaults) { m.Options.Shadows = false },
	}
	for name, change := range changes {
		r, m, d := base()
		change(&r, &m, &d)
		if got := rangeHash(r, m, d); got == want {
			t.Errorf("%s: hash didn't change", name)
		}
	}
}

func TestRangeKeysDuplicateNames(t *testing.T) {
	got := rangeKeys([]config.MapRange{{Name: "a"}, {Name: "b"}, {Name: "a"}})
	want := []string{"a", "b", "a#2"}
	if !slices.Equal(got, want) {
		t.Errorf("rangeKeys = %v, want %v", got, want)
	}
}

func TestRangeKeysAvoidLiteralNames(t *testing.T) {
	cases := []struct {
		names []string
		want  []string
	}{
		{[]string{"spawn", "spawn", "spawn#2"}, []string{"spawn", "spawn#3", "spawn#2"}},
		{[]string{"town#2", "town", "town"}, []string{"town#2", "town", "town#3"}},
		{[]string{"a", "a", "a", "a#3"}, []string{"a", "a#2", "a#4", "a#3"}},
		{[]string{"a#2", "a#2"}, []string{"a#2", "a#2#2"}},
	}
	for _, c := range cases {
		var ranges []config.MapRange
		for _, n := range c.names {
			ranges = append(ranges, config.MapRange{Name: n})
		}
		got := rangeKeys(ranges)
		if !slices.Equal(got, c.want) {
			t.Errorf("rangeKeys(%v) = %v, want %v", c.names, got, c.want)
		}
		seen := map[string]bool{}
		for _, k := range got {
			if seen[k] {
				t.Errorf("rangeKeys(%v) = %v: duplicate key %q", c.names, got, k)
			}
			seen[k] = true
		}
		if again := rangeKeys(ranges); !slices.Equal(again, got) {
			t.Errorf("rangeKeys(%v) not stable: %v then %v", c.names, got, again)
		}
	}
}

// Ranges named spawn, spawn, spawn#2 each keep their own hash: a failed
// render of one doesn't drop another's, and an unchanged world renders
// nothing once all of them have rendered.
func TestBuildMapDuplicateRangeNamesKeepSeparateHashes(t *testing.T) {
	defaults := config.MapDefaults{Zoomout: 6, Zoomin: 1, ImageFormat: "webp", ChunkProcessors: 1}
	mapDef := config.MapDefinition{Name: "overworld", Dimension: "overworld", Ranges: []config.MapRange{
		{Name: "spawn", Center: [2]int{0, 0}, Radius: 100},
		{Name: "spawn", Center: [2]int{2000, 2000}, Radius: 10},
		{Name: "spawn#2", Center: [2]int{-3000, 0}, Radius: 10},
	}}
	b, worldDir, worldMaps := rangesFixture(t, mapDef, defaults)
	ow := filepath.Join(worldMaps, "overworld")
	worldMtime := time.Now().Add(-time.Hour).Unix()

	// The literal spawn#2 fails; the second spawn keeps its new hash
	f := &recordingUnmined{fail: []string{"(-3072,"}}
	f.install(t)
	if err := b.buildMap(mapDef, defaults, worldDir, worldMaps, worldMtime, BuildOptions{WorldName: "w"}); err == nil {
		t.Fatal("buildMap succeeded, want the spawn#2 failure")
	}
	if len(f.calls) != 2 {
		t.Fatalf("renders = %v, want the two new ranges", f.calls)
	}
	m := readTestManifest(t, ow)
	if len(m.RangeHashes) != 2 || m.RangeHashes["spawn"] != rangeHash(mapDef.Ranges[0], mapDef, defaults) {
		t.Errorf("range hashes = %v, want spawn and the second spawn only", m.RangeHashes)
	}
	if _, ok := m.RangeHashes["spawn#2"]; ok {
		t.Errorf("failed spawn#2 has a hash: %v", m.RangeHashes)
	}

	// Retry: only spawn#2 renders
	f.calls, f.fail = nil, nil
	if err := b.buildMap(mapDef, defaults, worldDir, worldMaps, worldMtime, BuildOptions{WorldName: "w"}); err != nil {
		t.Fatalf("retry buildMap: %v", err)
	}
	if len(f.calls) != 1 || !strings.Contains(f.calls[0], "(-3072,") {
		t.Fatalf("retry renders = %v, want only spawn#2", f.calls)
	}

	// Up to date: nothing renders
	f.calls = nil
	if err := b.buildMap(mapDef, defaults, worldDir, worldMaps, worldMtime, BuildOptions{WorldName: "w"}); err != nil {
		t.Fatalf("third buildMap: %v", err)
	}
	if len(f.calls) != 0 {
		t.Errorf("third build rendered %v, want nothing", f.calls)
	}
	if m := readTestManifest(t, ow); len(m.RangeHashes) != 3 {
		t.Errorf("range hashes = %v, want three", m.RangeHashes)
	}
}

func TestDecideMap(t *testing.T) {
	defaults := config.MapDefaults{Zoomout: 6, Zoomin: 1, ImageFormat: "webp"}
	spawn := config.MapRange{Name: "spawn", Center: [2]int{0, 0}, Radius: 100}
	base := config.MapRange{Name: "base", Center: [2]int{500, 500}, Radius: 50}
	mapDef := config.MapDefinition{Name: "overworld", Dimension: "overworld", Ranges: []config.MapRange{spawn, base}}

	hashesFor := func(m config.MapDefinition) map[string]string {
		h := map[string]string{}
		for _, r := range m.Ranges {
			h[r.Name] = rangeHash(r, m, defaults)
		}
		return h
	}
	current := &Manifest{LastRenderedEpoch: 1000, RangeHashes: hashesFor(mapDef)}

	movedBase := base
	movedBase.Radius = 80
	changedDef := mapDef
	changedDef.Ranges = []config.MapRange{spawn, movedBase}

	addedDef := mapDef
	addedDef.Ranges = []config.MapRange{spawn, base, {Name: "farm", Center: [2]int{1, 1}, Radius: 10}}

	removedDef := mapDef
	removedDef.Ranges = []config.MapRange{spawn}

	tests := []struct {
		name       string
		in         planInput
		action     BuildAction
		ranges     []string
		force      bool
		fmtChanged bool
		reason     string
	}{
		{name: "unchanged", in: planInput{mapDef: mapDef, renderedFormat: "webp", manifest: current, worldMtime: 900},
			action: ActionSkip, reason: "up to date"},
		{name: "changed range", in: planInput{mapDef: changedDef, renderedFormat: "webp", manifest: current, worldMtime: 900},
			action: ActionRanges, ranges: []string{"base"}, force: true, reason: "0 new, 1 changed"},
		{name: "new range", in: planInput{mapDef: addedDef, renderedFormat: "webp", manifest: current, worldMtime: 900},
			action: ActionRanges, ranges: []string{"farm"}, force: true, reason: "1 new, 0 changed"},
		{name: "removed range", in: planInput{mapDef: removedDef, renderedFormat: "webp", manifest: current, worldMtime: 900},
			action: ActionSkip},
		{name: "no hashes yet", in: planInput{mapDef: mapDef, renderedFormat: "webp", manifest: &Manifest{LastRenderedEpoch: 1000}, worldMtime: 900},
			action: ActionRanges, ranges: []string{"spawn", "base"}, force: true},
		{name: "world changed", in: planInput{mapDef: changedDef, renderedFormat: "webp", manifest: current, worldMtime: 1001},
			action: ActionFull, ranges: []string{"spawn", "base"}, reason: "world changed"},
		{name: "no manifest", in: planInput{mapDef: mapDef, renderedFormat: "webp", worldMtime: 900},
			action: ActionFull, ranges: []string{"spawn", "base"}},
		{name: "force", in: planInput{mapDef: mapDef, renderedFormat: "webp", manifest: current, worldMtime: 900, force: true},
			action: ActionFull, ranges: []string{"spawn", "base"}, force: true},
		{name: "format changed, world unchanged", in: planInput{mapDef: mapDef, renderedFormat: "jpeg", manifest: current, worldMtime: 900},
			action: ActionFull, ranges: []string{"spawn", "base"}, force: true, fmtChanged: true, reason: "imageformat changed (jpeg→webp)"},
		{name: "format case differs only", in: planInput{mapDef: mapDef, renderedFormat: "WEBP", manifest: current, worldMtime: 900},
			action: ActionSkip},
		{name: "never rendered", in: planInput{mapDef: mapDef, manifest: current, worldMtime: 900},
			action: ActionFull, ranges: []string{"spawn", "base"}},
		{name: "ranges-only", in: planInput{mapDef: mapDef, renderedFormat: "webp", manifest: current, worldMtime: 900, rangesOnly: true},
			action: ActionRanges, ranges: []string{"spawn", "base"}, force: true},
		{name: "ranges-only, no ranges", in: planInput{mapDef: config.MapDefinition{Name: "n"}, renderedFormat: "webp", rangesOnly: true},
			action: ActionSkip},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.in.defaults = defaults
			p, err := decideMap(tt.in)
			if err != nil {
				t.Fatalf("decideMap: %v", err)
			}
			if p.Action != tt.action {
				t.Errorf("action = %s (%s), want %s", p.Action, p.Reason, tt.action)
			}
			if !slices.Equal(p.Ranges, tt.ranges) {
				t.Errorf("ranges = %v, want %v", p.Ranges, tt.ranges)
			}
			if p.Force != tt.force {
				t.Errorf("force = %v, want %v", p.Force, tt.force)
			}
			if p.FormatChanged != tt.fmtChanged {
				t.Errorf("formatChanged = %v, want %v", p.FormatChanged, tt.fmtChanged)
			}
			if !strings.Contains(p.Reason, tt.reason) {
				t.Errorf("reason = %q, want it to contain %q", p.Reason, tt.reason)
			}
		})
	}
}

func TestDecideMapRangesOnlyErrors(t *testing.T) {
	mapDef := config.MapDefinition{Name: "overworld", Ranges: []config.MapRange{{Name: "spawn", Radius: 10}}}
	defaults := config.MapDefaults{ImageFormat: "webp"}

	_, err := decideMap(planInput{mapDef: mapDef, defaults: defaults, renderedFormat: "jpeg", rangesOnly: true})
	want := "imageformat changed (jpeg→webp): a full rebuild is needed; run without --ranges-only"
	if err == nil || err.Error() != want {
		t.Errorf("format change: err = %v, want %q", err, want)
	}

	if _, err := decideMap(planInput{mapDef: mapDef, defaults: defaults, rangesOnly: true}); err == nil {
		t.Error("never rendered: expected an error")
	}
}

// recordingUnmined is a fake uNmINeD that records each render. A render
// writes a tile and rewrites the properties file like the real one (a range
// render puts its area in it). Ranges whose --area contains a string in fail
// return an error after writing the properties file.
type recordingUnmined struct {
	calls []string // "base" or the --area value
	fail  []string
}

func (f *recordingUnmined) install(t *testing.T) {
	t.Helper()
	orig := runUnmined
	t.Cleanup(func() { runUnmined = orig })
	runUnmined = func(_ string, args []string) error {
		var output, format, area string
		for i, a := range args {
			switch {
			case a == "--output":
				output = args[i+1]
			case a == "--imageformat":
				format = args[i+1]
			case strings.HasPrefix(a, "--area="):
				area = strings.TrimPrefix(a, "--area=")
			}
		}
		if area == "" {
			f.calls = append(f.calls, "base")
			writeProperties(t, output, format)
			return nil
		}
		f.calls = append(f.calls, area)
		props := `UnminedMapProperties = { imageFormat: "` + format + `", maxZoom: 9, area: "` + area + `" };`
		if err := os.WriteFile(filepath.Join(output, "unmined.map.properties.js"), []byte(props), 0644); err != nil {
			t.Fatal(err)
		}
		for _, s := range f.fail {
			if strings.Contains(area, s) {
				return errors.New("range render failed")
			}
		}
		return nil
	}
}

// rangesFixture is a world with one overworld map rendered as webp, with a
// manifest from a full render after the world last changed. The manifest has
// hashes for spawn and a since-removed range.
func rangesFixture(t *testing.T, mapDef config.MapDefinition, defaults config.MapDefaults) (b *Builder, worldDir, worldMaps string) {
	t.Helper()
	worldsDir := t.TempDir()
	mapsDir := t.TempDir()
	worldDir = filepath.Join(worldsDir, "w", "world")
	if err := os.MkdirAll(filepath.Join(worldDir, "region"), 0755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(worldDir, "level.dat"), 1)
	old := time.Now().Add(-time.Hour)
	if err := os.Chtimes(filepath.Join(worldDir, "level.dat"), old, old); err != nil {
		t.Fatal(err)
	}
	worldMaps = filepath.Join(mapsDir, "w")
	ow := filepath.Join(worldMaps, "overworld")
	writeProperties(t, ow, "webp")

	m := Manifest{
		World: "w", Map: "overworld", Dimension: "overworld",
		LastRendered: "earlier", LastRenderedEpoch: time.Now().Unix(),
		RangeHashes: map[string]string{
			"spawn":   rangeHash(mapDef.Ranges[0], mapDef, defaults),
			"removed": "0123456789abcdef",
		},
	}
	data, _ := json.Marshal(m)
	if err := os.WriteFile(filepath.Join(ow, "manifest.json"), data, 0644); err != nil {
		t.Fatal(err)
	}
	return &Builder{unminedPath: "unmined", worldsDir: worldsDir, mapsDir: mapsDir}, worldDir, worldMaps
}

func readTestManifest(t *testing.T, mapOutput string) *Manifest {
	t.Helper()
	m, err := readManifest(filepath.Join(mapOutput, "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func readProps(t *testing.T, mapOutput string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(mapOutput, "unmined.map.properties.js"))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestBuildMapRendersOnlyNewRanges(t *testing.T) {
	defaults := config.MapDefaults{Zoomout: 6, Zoomin: 1, ImageFormat: "webp", ChunkProcessors: 1}
	mapDef := config.MapDefinition{Name: "overworld", Dimension: "overworld", Ranges: []config.MapRange{
		{Name: "spawn", Center: [2]int{0, 0}, Radius: 100},
		{Name: "farm", Center: [2]int{2000, 2000}, Radius: 10, Zoomin: intp(3)},
	}}
	b, worldDir, worldMaps := rangesFixture(t, mapDef, defaults)
	ow := filepath.Join(worldMaps, "overworld")
	before := readTestManifest(t, ow)
	f := &recordingUnmined{}
	f.install(t)

	worldMtime := time.Now().Add(-time.Hour).Unix()
	if err := b.buildMap(mapDef, defaults, worldDir, worldMaps, worldMtime, BuildOptions{WorldName: "w", LogLevel: "warning"}); err != nil {
		t.Fatalf("buildMap: %v", err)
	}

	if len(f.calls) != 1 || !strings.Contains(f.calls[0], "1536,1536") {
		t.Fatalf("renders = %v, want only the farm range", f.calls)
	}
	want := `UnminedMapProperties = { imageFormat: "webp", maxZoom: 3 };`
	if got := readProps(t, ow); got != want {
		t.Errorf("properties = %q, want %q", got, want)
	}
	m := readTestManifest(t, ow)
	wantHashes := map[string]string{
		"spawn": rangeHash(mapDef.Ranges[0], mapDef, defaults),
		"farm":  rangeHash(mapDef.Ranges[1], mapDef, defaults),
	}
	if len(m.RangeHashes) != 2 || m.RangeHashes["spawn"] != wantHashes["spawn"] || m.RangeHashes["farm"] != wantHashes["farm"] {
		t.Errorf("range hashes = %v, want %v", m.RangeHashes, wantHashes)
	}
	if m.LastRenderedEpoch != before.LastRenderedEpoch || m.LastRendered != before.LastRendered {
		t.Errorf("render time changed on a ranges-only build: %+v", m)
	}

	// Nothing changed now: the next build renders nothing
	f.calls = nil
	if err := b.buildMap(mapDef, defaults, worldDir, worldMaps, worldMtime, BuildOptions{WorldName: "w"}); err != nil {
		t.Fatalf("second buildMap: %v", err)
	}
	if len(f.calls) != 0 {
		t.Errorf("second build rendered %v, want nothing", f.calls)
	}
}

func TestBuildMapRangesOnlyFailureRestoresProperties(t *testing.T) {
	defaults := config.MapDefaults{Zoomout: 6, Zoomin: 1, ImageFormat: "webp", ChunkProcessors: 1}
	mapDef := config.MapDefinition{Name: "overworld", Dimension: "overworld", Ranges: []config.MapRange{
		{Name: "spawn", Center: [2]int{0, 0}, Radius: 100},
		{Name: "farm", Center: [2]int{2000, 2000}, Radius: 10, Zoomin: intp(2)},
		{Name: "tower", Center: [2]int{-3000, 0}, Radius: 10},
	}}
	b, worldDir, worldMaps := rangesFixture(t, mapDef, defaults)
	ow := filepath.Join(worldMaps, "overworld")
	f := &recordingUnmined{fail: []string{"1536,1536"}} // farm fails
	f.install(t)

	err := b.buildMap(mapDef, defaults, worldDir, worldMaps, 0, BuildOptions{WorldName: "w", RangesOnly: true})
	if err == nil || !strings.Contains(err.Error(), "farm") {
		t.Fatalf("buildMap err = %v, want it to name farm", err)
	}
	if len(f.calls) != 3 || slices.Contains(f.calls, "base") {
		t.Errorf("renders = %v, want all three ranges and no base map", f.calls)
	}
	want := `UnminedMapProperties = { imageFormat: "webp", maxZoom: 2 };`
	if got := readProps(t, ow); got != want {
		t.Errorf("properties = %q, want %q", got, want)
	}
	m := readTestManifest(t, ow)
	if _, ok := m.RangeHashes["farm"]; ok {
		t.Error("failed range got a hash")
	}
	if _, ok := m.RangeHashes["removed"]; ok {
		t.Error("removed range kept its hash")
	}
	if m.RangeHashes["spawn"] == "" || m.RangeHashes["tower"] != rangeHash(mapDef.Ranges[2], mapDef, defaults) {
		t.Errorf("range hashes = %v, want spawn and tower", m.RangeHashes)
	}
}

func TestBuildMapFormatChangeUnchangedWorldIsFull(t *testing.T) {
	defaults := config.MapDefaults{Zoomout: 6, Zoomin: 1, ImageFormat: "webp", ChunkProcessors: 1}
	mapDef := config.MapDefinition{Name: "overworld", Dimension: "overworld", Ranges: []config.MapRange{
		{Name: "spawn", Center: [2]int{0, 0}, Radius: 100},
	}}
	b, worldDir, worldMaps := rangesFixture(t, mapDef, defaults)
	ow := filepath.Join(worldMaps, "overworld")
	writeProperties(t, ow, "jpeg")
	oldTile := filepath.Join(ow, "tiles", "zoom.0", "0", "0", "tile.5.5.jpeg")
	writeFile(t, oldTile, 3)
	f := &recordingUnmined{}
	f.install(t)

	if err := b.buildMap(mapDef, defaults, worldDir, worldMaps, 0, BuildOptions{WorldName: "w"}); err != nil {
		t.Fatalf("buildMap: %v", err)
	}
	if len(f.calls) != 2 || f.calls[1] != "base" {
		t.Errorf("renders = %v, want the range then the base map", f.calls)
	}
	if exists(oldTile) {
		t.Error("old-format tile kept after a full render")
	}
	m := readTestManifest(t, ow)
	if len(m.RangeHashes) != 1 || m.RangeHashes["spawn"] != rangeHash(mapDef.Ranges[0], mapDef, defaults) {
		t.Errorf("range hashes = %v, want a fresh hash for spawn only", m.RangeHashes)
	}
}

func TestBuildRangesOnlyFormatChangeErrorsBeforeRendering(t *testing.T) {
	defaults := config.MapDefaults{Zoomout: 6, Zoomin: 1, ImageFormat: "webp", ChunkProcessors: 1}
	mapDef := config.MapDefinition{Name: "overworld", Dimension: "overworld", Ranges: []config.MapRange{
		{Name: "spawn", Center: [2]int{0, 0}, Radius: 100},
	}}
	b, _, worldMaps := rangesFixture(t, mapDef, defaults)
	// A second map that's fine on its own, listed first: it mustn't render either
	writeProperties(t, filepath.Join(worldMaps, "caves"), "webp")
	writeProperties(t, filepath.Join(worldMaps, "overworld"), "jpeg")
	cfg := `defaults:
  imageformat: webp
maps:
  - name: caves
    dimension: overworld
    ranges:
      - name: spawn
        center: [0, 0]
        radius: 100
  - name: overworld
    dimension: overworld
    ranges:
      - name: spawn
        center: [0, 0]
        radius: 100
`
	if err := os.WriteFile(filepath.Join(b.worldsDir, "w", "map-config.yml"), []byte(cfg), 0644); err != nil {
		t.Fatal(err)
	}
	f := &recordingUnmined{}
	f.install(t)

	err := b.Build(BuildOptions{WorldName: "w", RangesOnly: true, NoLock: true})
	if err == nil || !strings.Contains(err.Error(), "imageformat changed (jpeg→webp): a full rebuild is needed; run without --ranges-only") {
		t.Fatalf("Build err = %v", err)
	}
	if len(f.calls) != 0 {
		t.Errorf("renders = %v, want none", f.calls)
	}
}

func TestBuildManifestsKeepsBuildState(t *testing.T) {
	defaults := config.MapDefaults{ImageFormat: "webp"}
	mapDef := config.MapDefinition{Name: "overworld", Dimension: "overworld", Ranges: []config.MapRange{{Name: "spawn", Radius: 10}}}
	b, _, worldMaps := rangesFixture(t, mapDef, defaults)
	ow := filepath.Join(worldMaps, "overworld")
	before := readTestManifest(t, ow)

	if err := os.WriteFile(filepath.Join(b.worldsDir, "w", "map-config.yml"),
		[]byte("maps:\n  - name: overworld\n    dimension: overworld\n"), 0644); err != nil {
		t.Fatal(err)
	}
	mb := &ManifestBuilder{worldsDir: b.worldsDir, mapsDir: b.mapsDir, builder: b}
	// level.dat isn't real NBT, so the world manifest fails; the map
	// manifest is written before that
	_ = mb.BuildManifests("w", ManifestOptions{})

	m := readTestManifest(t, ow)
	if m.LastRenderedEpoch != before.LastRenderedEpoch || len(m.RangeHashes) != len(before.RangeHashes) {
		t.Errorf("manifest lost build state: %+v", m)
	}
	if m.Path != "w/overworld" {
		t.Errorf("path = %q, want w/overworld", m.Path)
	}
}

func TestDecideMapFullBuildForcesChangedRanges(t *testing.T) {
	defaults := config.MapDefaults{Zoomout: 6, Zoomin: 1, ImageFormat: "webp"}
	spawn := config.MapRange{Name: "spawn", Center: [2]int{0, 0}, Radius: 100}
	base := config.MapRange{Name: "base", Center: [2]int{500, 500}, Radius: 50}
	mapDef := config.MapDefinition{Name: "overworld", Dimension: "overworld", Ranges: []config.MapRange{spawn, base}}
	manifest := &Manifest{LastRenderedEpoch: 1000, RangeHashes: map[string]string{
		"spawn": rangeHash(spawn, mapDef, defaults),
		"base":  rangeHash(base, mapDef, defaults),
	}}
	movedBase := base
	movedBase.Radius = 80
	changedDef := mapDef
	changedDef.Ranges = []config.MapRange{spawn, movedBase}

	tests := []struct {
		name   string
		in     planInput
		forced []string
	}{
		{"world changed, one range changed", planInput{mapDef: changedDef, renderedFormat: "webp", manifest: manifest, worldMtime: 1001}, []string{"base"}},
		{"world changed, ranges unchanged", planInput{mapDef: mapDef, renderedFormat: "webp", manifest: manifest, worldMtime: 1001}, nil},
		{"no manifest", planInput{mapDef: mapDef, renderedFormat: "webp", worldMtime: 900}, []string{"spawn", "base"}},
		{"--force covers every range", planInput{mapDef: changedDef, renderedFormat: "webp", manifest: manifest, worldMtime: 900, force: true}, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.in.defaults = defaults
			p, err := decideMap(tt.in)
			if err != nil {
				t.Fatal(err)
			}
			if p.Action != ActionFull {
				t.Fatalf("action = %s, want full", p.Action)
			}
			if !slices.Equal(p.ForceRanges, tt.forced) {
				t.Errorf("ForceRanges = %v, want %v", p.ForceRanges, tt.forced)
			}
			for i, r := range tt.in.mapDef.Ranges {
				want := p.Force || slices.Contains(tt.forced, r.Name)
				if got := p.rangeForced(i); got != want {
					t.Errorf("rangeForced(%s) = %v, want %v", r.Name, got, want)
				}
			}
			if len(tt.forced) > 0 && !strings.Contains(p.Reason, "render with --force") {
				t.Errorf("reason = %q, want it to mention the forced ranges", p.Reason)
			}
		})
	}
}

// A full build after the world changed must still --force a range whose
// settings changed: uNmINeD's incremental check misses option/zoom changes,
// and the new hash means it would never be re-rendered.
func TestBuildMapWorldChangedForcesChangedRange(t *testing.T) {
	defaults := config.MapDefaults{Zoomout: 6, Zoomin: 1, ImageFormat: "webp", ChunkProcessors: 1}
	mapDef := config.MapDefinition{Name: "overworld", Dimension: "overworld", Ranges: []config.MapRange{
		{Name: "spawn", Center: [2]int{0, 0}, Radius: 100},
	}}
	b, worldDir, worldMaps := rangesFixture(t, mapDef, defaults)
	// spawn's zoom changed since the manifest was written
	changed := mapDef
	changed.Ranges = []config.MapRange{{Name: "spawn", Center: [2]int{0, 0}, Radius: 100, Zoomin: intp(3)}}

	forced := map[string]bool{}
	orig := runUnmined
	t.Cleanup(func() { runUnmined = orig })
	runUnmined = func(_ string, args []string) error {
		name := "base"
		for _, a := range args {
			if strings.HasPrefix(a, "--area=") {
				name = "range"
			}
		}
		forced[name] = slices.Contains(args, "--force")
		writeProperties(t, filepath.Join(worldMaps, "overworld"), "webp")
		return nil
	}

	worldMtime := time.Now().Add(time.Hour).Unix() // newer than the manifest
	if err := b.buildMap(changed, defaults, worldDir, worldMaps, worldMtime, BuildOptions{WorldName: "w"}); err != nil {
		t.Fatalf("buildMap: %v", err)
	}
	if !forced["range"] {
		t.Error("changed range rendered without --force on a full build")
	}
	if forced["base"] {
		t.Error("base map rendered with --force on a non-forced full build")
	}
}

// A format change whose range renders but whose base render fails must
// leave the old format in the properties file, so the next build is still a
// full one that converts the base tiles.
func TestBuildMapFormatChangeBaseFailureKeepsFullRebuild(t *testing.T) {
	defaults := config.MapDefaults{Zoomout: 6, Zoomin: 1, ImageFormat: "webp", ChunkProcessors: 1}
	mapDef := config.MapDefinition{Name: "overworld", Dimension: "overworld", Ranges: []config.MapRange{
		{Name: "spawn", Center: [2]int{0, 0}, Radius: 100},
	}}
	b, worldDir, worldMaps := rangesFixture(t, mapDef, defaults)
	ow := filepath.Join(worldMaps, "overworld")
	writeProperties(t, ow, "jpeg")
	before := readProps(t, ow)
	cfg := "defaults:\n  imageformat: webp\n  zoomout: 6\n  zoomin: 1\nmaps:\n  - name: overworld\n    dimension: overworld\n    ranges:\n      - name: spawn\n        center: [0, 0]\n        radius: 100\n"
	if err := os.WriteFile(filepath.Join(b.worldsDir, "w", "map-config.yml"), []byte(cfg), 0644); err != nil {
		t.Fatal(err)
	}

	orig := runUnmined
	t.Cleanup(func() { runUnmined = orig })
	runUnmined = func(_ string, args []string) error {
		for _, a := range args {
			if strings.HasPrefix(a, "--area=") {
				writeProperties(t, ow, "webp") // the range render succeeds
				return nil
			}
		}
		return errors.New("base render failed")
	}

	if err := b.buildMap(mapDef, defaults, worldDir, worldMaps, 0, BuildOptions{WorldName: "w"}); err == nil {
		t.Fatal("expected buildMap to fail")
	}
	if got := readProps(t, ow); got != before {
		t.Errorf("properties = %q, want the pre-build %q", got, before)
	}

	plans, err := b.Plan(BuildOptions{WorldName: "w"})
	if err != nil {
		t.Fatal(err)
	}
	if len(plans) != 1 || plans[0].Action != ActionFull || !plans[0].FormatChanged {
		t.Errorf("next plan = %+v, want a full build with formatChanged", plans)
	}
}
