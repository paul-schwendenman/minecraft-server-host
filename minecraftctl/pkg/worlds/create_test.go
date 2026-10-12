package worlds

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/paul/minecraftctl/pkg/config"
	"github.com/spf13/viper"
)

// setupCreate points worlds_dir at a temp dir and the server jar at a temp
// file, returning the worlds dir
func setupCreate(t *testing.T) string {
	t.Helper()
	worldsDir := t.TempDir()
	viper.Reset()
	t.Cleanup(viper.Reset)
	viper.Set("worlds_dir", worldsDir)
	if err := config.Init(""); err != nil {
		t.Fatalf("config.Init() = %v", err)
	}

	jar := filepath.Join(t.TempDir(), "server.jar")
	if err := os.WriteFile(jar, []byte("jar"), 0644); err != nil {
		t.Fatal(err)
	}
	orig := jarPathFor
	jarPathFor = func(string) string { return jar }
	t.Cleanup(func() { jarPathFor = orig })
	return worldsDir
}

func stepDescs(p *CreatePlan) string {
	var descs []string
	for _, s := range p.Steps {
		descs = append(descs, s.Desc)
	}
	return strings.Join(descs, "\n")
}

func TestPlanCreateWorldChangesNothing(t *testing.T) {
	worldsDir := setupCreate(t)

	plan, err := PlanCreateWorld("fresh", CreateWorldOptions{Version: "1.21.1", CreateMapConfig: true, EnableSystemd: true})
	if err != nil {
		t.Fatalf("PlanCreateWorld() = %v", err)
	}
	worldDir := filepath.Join(worldsDir, "fresh")
	if _, err := os.Lstat(worldDir); !os.IsNotExist(err) {
		t.Fatalf("planning created %s (err %v)", worldDir, err)
	}

	got := stepDescs(plan)
	for _, want := range []string{
		"create directory " + worldDir,
		"symlink " + filepath.Join(worldDir, "server.jar") + " -> ",
		"write " + filepath.Join(worldDir, "eula.txt"),
		"write " + filepath.Join(worldDir, "server.properties"),
		"write " + filepath.Join(worldDir, "map-config.yml"),
		"chown -R minecraft:minecraft " + worldDir,
		"systemctl daemon-reload",
		"systemctl enable minecraft-map-build@fresh.timer",
		"enable --now minecraft-world-backup@fresh.timer minecraft-map-backup@fresh.timer",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("plan missing %q:\n%s", want, got)
		}
	}
}

func TestPlanCreateWorldSkipsExistingFiles(t *testing.T) {
	worldsDir := setupCreate(t)
	worldDir := filepath.Join(worldsDir, "half")
	if err := os.MkdirAll(worldDir, 0755); err != nil {
		t.Fatal(err)
	}
	for _, f := range []string{"server.properties", "map-config.yml"} {
		if err := os.WriteFile(filepath.Join(worldDir, f), []byte("keep"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	plan, err := PlanCreateWorld("half", CreateWorldOptions{Version: "1.21.1", CreateMapConfig: true})
	if err != nil {
		t.Fatalf("PlanCreateWorld() = %v", err)
	}
	got := stepDescs(plan)
	for _, unwanted := range []string{"server.properties", "map-config.yml", "systemctl"} {
		if strings.Contains(got, unwanted) {
			t.Errorf("plan mentions %q, which exists or wasn't asked for:\n%s", unwanted, got)
		}
	}
}

func TestPlanCreateWorldRefusesExistingWorld(t *testing.T) {
	worldsDir := setupCreate(t)
	level := filepath.Join(worldsDir, "old", "world", "level.dat")
	if err := os.MkdirAll(filepath.Dir(level), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(level, nil, 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := PlanCreateWorld("old", CreateWorldOptions{Version: "1.21.1"}); err == nil {
		t.Error("PlanCreateWorld() on an existing world = nil, want error")
	}
}

func TestCreateWorldRunsPlan(t *testing.T) {
	worldsDir := setupCreate(t)

	if err := CreateWorld("made", CreateWorldOptions{Version: "1.21.1", CreateMapConfig: true}); err != nil {
		t.Fatalf("CreateWorld() = %v", err)
	}
	worldDir := filepath.Join(worldsDir, "made")
	for _, f := range []string{"eula.txt", "server.properties", "map-config.yml"} {
		if _, err := os.Stat(filepath.Join(worldDir, f)); err != nil {
			t.Errorf("%s not created: %v", f, err)
		}
	}
	if target, err := os.Readlink(filepath.Join(worldDir, "server.jar")); err != nil || target != jarPathFor("") {
		t.Errorf("server.jar -> %q (%v), want %q", target, err, jarPathFor(""))
	}
}
