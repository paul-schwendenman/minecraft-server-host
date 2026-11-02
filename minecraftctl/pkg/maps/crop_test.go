package maps

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/paul/minecraftctl/pkg/config"
)

func TestCropArea(t *testing.T) {
	got, err := cropArea(config.MapDefinition{Name: "m"})
	if err != nil || got != "" {
		t.Errorf("no crop: got %q, %v; want \"\"", got, err)
	}
	// Snapped outward to region boundaries
	got, err = cropArea(config.MapDefinition{Name: "m", Crop: &config.MapCrop{Center: [2]int{-7424, 576}, Radius: 384}})
	if err != nil {
		t.Fatal(err)
	}
	if want := "b((-8192,0),(-6656,1024))"; got != want {
		t.Errorf("crop = %q, want %q", got, want)
	}
	if _, err := cropArea(config.MapDefinition{Name: "m", Crop: &config.MapCrop{Radius: 0}}); err == nil {
		t.Error("zero radius: want an error")
	}
}

func TestDecideMapCropChanges(t *testing.T) {
	defaults := config.MapDefaults{Zoomout: 6, Zoomin: 1, ImageFormat: "webp"}
	plain := config.MapDefinition{Name: "m", Dimension: "overworld"}
	cropped := plain
	cropped.Crop = &config.MapCrop{Center: [2]int{0, 0}, Radius: 100}
	area, _ := cropArea(cropped)
	moved := plain
	moved.Crop = &config.MapCrop{Center: [2]int{2000, 0}, Radius: 100}

	tests := []struct {
		name     string
		mapDef   config.MapDefinition
		manifest Manifest
		action   BuildAction
		force    bool
	}{
		{"no crop, unchanged", plain, Manifest{LastRenderedEpoch: 1000}, ActionSkip, false},
		{"crop unchanged", cropped, Manifest{LastRenderedEpoch: 1000, Crop: area}, ActionSkip, false},
		{"crop added", cropped, Manifest{LastRenderedEpoch: 1000}, ActionFull, true},
		{"crop moved", moved, Manifest{LastRenderedEpoch: 1000, Crop: area}, ActionFull, true},
		{"crop removed", plain, Manifest{LastRenderedEpoch: 1000, Crop: area}, ActionFull, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := tt.manifest
			p, err := decideMap(planInput{mapDef: tt.mapDef, defaults: defaults, renderedFormat: "webp", manifest: &m, worldMtime: 900})
			if err != nil {
				t.Fatal(err)
			}
			if p.Action != tt.action || p.Force != tt.force {
				t.Errorf("action = %s, force = %v (%s); want %s, %v", p.Action, p.Force, p.Reason, tt.action, tt.force)
			}
			if tt.action == ActionFull && !strings.Contains(p.Reason, "crop changed") {
				t.Errorf("reason = %q, want it to mention the crop", p.Reason)
			}
		})
	}
}

func TestBuildMapCropLimitsBaseRender(t *testing.T) {
	worldDir := filepath.Join(t.TempDir(), "world")
	if err := os.MkdirAll(filepath.Join(worldDir, "region"), 0755); err != nil {
		t.Fatal(err)
	}
	worldMaps := t.TempDir()
	var calls [][]string
	orig := runUnmined
	t.Cleanup(func() { runUnmined = orig })
	runUnmined = func(_ string, args []string) error {
		calls = append(calls, args)
		writeProperties(t, argValue(args, "--output"), argValue(args, "--imageformat"))
		return nil
	}

	defaults := config.MapDefaults{Zoomout: 6, Zoomin: 1, ImageFormat: "webp", ChunkProcessors: 1}
	mapDef := config.MapDefinition{Name: "mushroom", Dimension: "overworld", Zoomin: intp(3),
		Crop: &config.MapCrop{Center: [2]int{-7424, 576}, Radius: 384}}
	b := &Builder{unminedPath: "unmined-cli"}
	if err := b.buildMap(mapDef, defaults, worldDir, worldMaps, time.Now().Unix(), BuildOptions{WorldName: "w", LogLevel: "warning"}); err != nil {
		t.Fatalf("buildMap: %v", err)
	}
	if len(calls) != 1 {
		t.Fatalf("renders = %v, want one base render", calls)
	}
	joined := strings.Join(calls[0], " ")
	if !strings.Contains(joined, "--area=b((-8192,0),(-6656,1024))") {
		t.Errorf("base render args = %v, want the snapped crop area", calls[0])
	}
	if !strings.Contains(joined, "--zoomin 3") {
		t.Errorf("base render args = %v, want --zoomin 3", calls[0])
	}
	m := readTestManifest(t, filepath.Join(worldMaps, "mushroom"))
	if m.Crop != "b((-8192,0),(-6656,1024))" {
		t.Errorf("manifest crop = %q", m.Crop)
	}

	// Same crop, unchanged world: nothing renders
	calls = nil
	if err := b.buildMap(mapDef, defaults, worldDir, worldMaps, 0, BuildOptions{WorldName: "w"}); err != nil {
		t.Fatal(err)
	}
	if len(calls) != 0 {
		t.Errorf("second build rendered %v, want nothing", calls)
	}
}

func TestGeneratePreviewCenteredOnCrop(t *testing.T) {
	b, _ := previewFixture(t)
	cfg := previewWorldConfig + `  - name: island
    dimension: overworld
    crop:
      center: [-7424, 576]
      radius: 384
`
	if err := os.WriteFile(filepath.Join(b.worldsDir, "w", "map-config.yml"), []byte(cfg), 0644); err != nil {
		t.Fatal(err)
	}
	f := &previewUnmined{}
	f.install(t)
	if err := b.GeneratePreview("w", "island", ""); err != nil {
		t.Fatal(err)
	}
	if got, want := argValue(f.calls[len(f.calls)-1], "--area"), "b((-7488,512),(-7360,640))"; got != want {
		t.Errorf("--area = %s, want %s (the crop center, not spawn)", got, want)
	}
}

func TestBuildMapCropClipsRanges(t *testing.T) {
	worldDir := filepath.Join(t.TempDir(), "world")
	if err := os.MkdirAll(filepath.Join(worldDir, "region"), 0755); err != nil {
		t.Fatal(err)
	}
	worldMaps := t.TempDir()
	var calls [][]string
	orig := runUnmined
	t.Cleanup(func() { runUnmined = orig })
	runUnmined = func(_ string, args []string) error {
		calls = append(calls, args)
		writeProperties(t, argValue(args, "--output"), argValue(args, "--imageformat"))
		return nil
	}

	defaults := config.MapDefaults{Zoomout: 6, Zoomin: 1, ImageFormat: "webp", ChunkProcessors: 1}
	// Crop snaps to b((-1024,-1024),(1024,1024))
	mapDef := config.MapDefinition{Name: "m", Dimension: "overworld",
		Crop: &config.MapCrop{Center: [2]int{0, 0}, Radius: 1000},
		Ranges: []config.MapRange{
			// Snaps to b((512,-512),(1536,512)): clipped to x 512..1024
			{Name: "edge", Center: [2]int{1000, 0}, Radius: 400, Zoomin: intp(3)},
			// Snaps to b((4608,4608),(5632,5632)): entirely outside
			{Name: "far", Center: [2]int{5000, 5000}, Radius: 300, Zoomin: intp(3)},
		}}
	b := &Builder{unminedPath: "unmined-cli"}
	if err := b.buildMap(mapDef, defaults, worldDir, worldMaps, time.Now().Unix(), BuildOptions{WorldName: "w", LogLevel: "warning"}); err != nil {
		t.Fatalf("buildMap: %v", err)
	}
	if len(calls) != 2 {
		t.Fatalf("renders = %v, want the clipped range and the base map", calls)
	}
	if got, want := areaArg(calls[0]), "b((512,-512),(1024,512))"; got != want {
		t.Errorf("range --area = %s, want %s", got, want)
	}
	if got, want := areaArg(calls[1]), "b((-1024,-1024),(1024,1024))"; got != want {
		t.Errorf("base --area = %s, want %s", got, want)
	}
	// The skipped range counts as done, so it isn't retried on every build
	m := readTestManifest(t, filepath.Join(worldMaps, "m"))
	if len(m.RangeHashes) != 2 {
		t.Errorf("range hashes = %v, want both ranges", m.RangeHashes)
	}
}

// areaArg returns the value of a --area=... argument
func areaArg(args []string) string {
	for _, a := range args {
		if v, ok := strings.CutPrefix(a, "--area="); ok {
			return v
		}
	}
	return ""
}
