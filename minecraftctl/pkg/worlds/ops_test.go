package worlds

import (
	"errors"
	"io"
	"net"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	gorcon "github.com/gorcon/rcon"
	"github.com/gorcon/rcon/rcontest"
	"github.com/paul/minecraftctl/pkg/config"
	"github.com/spf13/viper"
)

// setupWorlds points worlds_dir and jars_dir at temp dirs and returns them.
func setupWorlds(t *testing.T) (worldsDir, jarsDir string) {
	t.Helper()
	worldsDir = t.TempDir()
	jarsDir = t.TempDir()
	viper.Reset()
	t.Cleanup(viper.Reset)
	viper.Set("worlds_dir", worldsDir)
	viper.Set("jars_dir", jarsDir)
	if err := config.Init(""); err != nil {
		t.Fatalf("config.Init() = %v", err)
	}
	return worldsDir, jarsDir
}

// addLevelDat gives a world the sample level.dat from testdata.
func addLevelDat(t *testing.T, worldsDir, name string) {
	t.Helper()
	src, err := os.Open(filepath.Join("..", "..", "testdata", "default", "world", "level.dat"))
	if err != nil {
		t.Fatal(err)
	}
	defer src.Close()
	dir := filepath.Join(worldsDir, name, "world")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	dst, err := os.Create(filepath.Join(dir, "level.dat"))
	if err != nil {
		t.Fatal(err)
	}
	defer dst.Close()
	if _, err := io.Copy(dst, src); err != nil {
		t.Fatal(err)
	}
}

// fakeSystemctl puts a systemctl on PATH that logs each call to the returned
// file and then runs script (with the call's args as "$@").
func fakeSystemctl(t *testing.T, script string) (logPath string) {
	t.Helper()
	bin := t.TempDir()
	logPath = filepath.Join(bin, "calls.log")
	body := "#!/bin/sh\necho \"$*\" >> " + logPath + "\n" + script + "\n"
	if err := os.WriteFile(filepath.Join(bin, "systemctl"), []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	return logPath
}

func readCalls(t *testing.T, logPath string) []string {
	t.Helper()
	data, err := os.ReadFile(logPath)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		t.Fatal(err)
	}
	return strings.Split(strings.TrimSpace(string(data)), "\n")
}

func TestGetWorldNamesAndListWorlds(t *testing.T) {
	worldsDir, _ := setupWorlds(t)
	addLevelDat(t, worldsDir, "zeta")
	addLevelDat(t, worldsDir, "alpha")
	// Not worlds: a dir with no level.dat and a plain file
	if err := os.MkdirAll(filepath.Join(worldsDir, "lost+found"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(worldsDir, "notes.txt"), "x")
	writeFile(t, filepath.Join(worldsDir, "alpha", "map-config.yml"), "maps: []\n")

	names, err := GetWorldNames()
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"alpha", "zeta"}; !reflect.DeepEqual(names, want) {
		t.Errorf("GetWorldNames() = %v, want %v", names, want)
	}

	list, err := ListWorlds()
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 || list[0].Name != "alpha" || list[1].Name != "zeta" {
		t.Fatalf("ListWorlds() = %+v, want alpha, zeta", list)
	}
	if !list[0].HasMapConfig || list[1].HasMapConfig {
		t.Errorf("HasMapConfig = %v/%v, want true/false", list[0].HasMapConfig, list[1].HasMapConfig)
	}
	a := list[0]
	if a.Path != filepath.Join(worldsDir, "alpha") || a.Version == "" || a.Difficulty == "Unknown" || a.GameType == "Unknown" {
		t.Errorf("unexpected info %+v", a)
	}
}

func TestListWorldsMissingDir(t *testing.T) {
	setupWorlds(t)
	viper.Set("worlds_dir", filepath.Join(t.TempDir(), "missing"))
	if err := config.Init(""); err != nil {
		t.Fatal(err)
	}
	if _, err := ListWorlds(); err == nil {
		t.Error("ListWorlds() on a missing dir = nil error")
	}
	if _, err := GetWorldNames(); err == nil {
		t.Error("GetWorldNames() on a missing dir = nil error")
	}
}

func TestGetWorldInfoErrors(t *testing.T) {
	worldsDir, _ := setupWorlds(t)
	if _, err := GetWorldInfo("nope"); err == nil || !strings.Contains(err.Error(), "world not found") {
		t.Errorf("GetWorldInfo(missing) = %v, want world not found", err)
	}
	// A corrupt level.dat is reported, and ListWorlds skips the world
	if err := os.MkdirAll(filepath.Join(worldsDir, "bad", "world"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(worldsDir, "bad", "world", "level.dat"), "not nbt")
	if _, err := GetWorldInfo("bad"); err == nil || !strings.Contains(err.Error(), "level.dat") {
		t.Errorf("GetWorldInfo(corrupt) = %v, want level.dat error", err)
	}
	list, err := ListWorlds()
	if err != nil || len(list) != 0 {
		t.Errorf("ListWorlds() = %v, %v; want the corrupt world skipped", list, err)
	}
}

func TestDifficultyAndGameTypeNames(t *testing.T) {
	diffs := []string{"Peaceful", "Easy", "Normal", "Hard", "Unknown"}
	modes := []string{"Survival", "Creative", "Adventure", "Spectator", "Unknown"}
	for i := range diffs {
		if got := difficultyName(int32(i)); got != diffs[i] {
			t.Errorf("difficultyName(%d) = %q, want %q", i, got, diffs[i])
		}
		if got := gameTypeName(int32(i)); got != modes[i] {
			t.Errorf("gameTypeName(%d) = %q, want %q", i, got, modes[i])
		}
	}
}

func TestExpandWorldPattern(t *testing.T) {
	worldsDir, _ := setupWorlds(t)
	for _, w := range []string{"survival", "survival-old", "creative"} {
		addLevelDat(t, worldsDir, w)
	}
	// Matches the glob but isn't a world
	if err := os.MkdirAll(filepath.Join(worldsDir, "survival-empty"), 0o755); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		pattern string
		want    []string
		wantErr string
	}{
		{pattern: "creative", want: []string{"creative"}},
		{pattern: "missing", wantErr: "world not found"},
		{pattern: "surv*", want: []string{"survival", "survival-old"}},
		{pattern: "*", want: []string{"creative", "survival", "survival-old"}},
		{pattern: "x*", wantErr: "no worlds match"},
		{pattern: "[", wantErr: "invalid glob"},
	}
	for _, tt := range tests {
		got, err := ExpandWorldPattern(tt.pattern)
		if tt.wantErr != "" {
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("ExpandWorldPattern(%q) error = %v, want %q", tt.pattern, err, tt.wantErr)
			}
			continue
		}
		if err != nil || !reflect.DeepEqual(got, tt.want) {
			t.Errorf("ExpandWorldPattern(%q) = %v, %v; want %v", tt.pattern, got, err, tt.want)
		}
	}
}

func TestRegisterWorld(t *testing.T) {
	worldsDir, _ := setupWorlds(t)
	addLevelDat(t, worldsDir, "survival")
	calls := fakeSystemctl(t, "exit 0")

	origEnable, origEnableNow := enableUnit, enableUnitNow
	t.Cleanup(func() { enableUnit, enableUnitNow = origEnable, origEnableNow })
	var timers []string
	enableUnit = func(u string) error { timers = append(timers, u); return nil }
	enableUnitNow = func(u string) error { timers = append(timers, u+" --now"); return nil }

	if err := RegisterWorld("survival"); err != nil {
		t.Fatalf("RegisterWorld() = %v", err)
	}
	if got := readCalls(t, calls); !reflect.DeepEqual(got, []string{"daemon-reload"}) {
		t.Errorf("systemctl calls = %v, want [daemon-reload]", got)
	}
	if len(timers) != 3 {
		t.Errorf("timers enabled = %v, want 3", timers)
	}

	if err := RegisterWorld("missing"); err == nil || !strings.Contains(err.Error(), "world not found") {
		t.Errorf("RegisterWorld(missing) = %v, want world not found", err)
	}
}

func TestRegisterWorldDaemonReloadFails(t *testing.T) {
	worldsDir, _ := setupWorlds(t)
	addLevelDat(t, worldsDir, "survival")
	fakeSystemctl(t, "exit 1")
	err := RegisterWorld("survival")
	if err == nil || !strings.Contains(err.Error(), "daemon") {
		t.Errorf("RegisterWorld() = %v, want daemon-reload error", err)
	}
}

func TestGetCurrentVersion(t *testing.T) {
	dir := t.TempDir()
	if _, err := GetCurrentVersion(dir); err == nil {
		t.Error("no server.jar: want error")
	}
	if err := os.Symlink("/opt/minecraft/jars/minecraft_server_1.21.4.jar", filepath.Join(dir, "server.jar")); err != nil {
		t.Fatal(err)
	}
	v, err := GetCurrentVersion(dir)
	if err != nil || v != "1.21.4" {
		t.Errorf("GetCurrentVersion() = %q, %v; want 1.21.4", v, err)
	}

	other := t.TempDir()
	if err := os.Symlink("/opt/paper.jar", filepath.Join(other, "server.jar")); err != nil {
		t.Fatal(err)
	}
	if _, err := GetCurrentVersion(other); err == nil || !strings.Contains(err.Error(), "unexpected jar filename") {
		t.Errorf("GetCurrentVersion(paper) = %v, want unexpected jar filename", err)
	}
}

// setupUpgrade makes a world on 1.21.1 with 1.21.4 downloaded.
func setupUpgrade(t *testing.T) (worldsDir, jarsDir string) {
	t.Helper()
	worldsDir, jarsDir = setupWorlds(t)
	addLevelDat(t, worldsDir, "survival")
	for _, v := range []string{"1.21.1", "1.21.4"} {
		writeFile(t, filepath.Join(jarsDir, "minecraft_server_"+v+".jar"), "")
	}
	if err := os.Symlink(filepath.Join(jarsDir, "minecraft_server_1.21.1.jar"), filepath.Join(worldsDir, "survival", "server.jar")); err != nil {
		t.Fatal(err)
	}
	return worldsDir, jarsDir
}

func assertJar(t *testing.T, worldsDir, jarsDir, version string) {
	t.Helper()
	target, err := os.Readlink(filepath.Join(worldsDir, "survival", "server.jar"))
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(jarsDir, "minecraft_server_"+version+".jar"); target != want {
		t.Errorf("server.jar -> %s, want %s", target, want)
	}
}

func TestUpgradeWorldStopped(t *testing.T) {
	worldsDir, jarsDir := setupUpgrade(t)
	calls := fakeSystemctl(t, `[ "$1" = is-active ] && exit 3; exit 0`)

	res, err := UpgradeWorld("survival", UpgradeOptions{TargetVersion: "1.21.4"})
	if err != nil {
		t.Fatalf("UpgradeWorld() = %v", err)
	}
	want := &UpgradeResult{WorldName: "survival", PreviousVersion: "1.21.1", NewVersion: "1.21.4"}
	if !reflect.DeepEqual(res, want) {
		t.Errorf("result = %+v, want %+v", res, want)
	}
	assertJar(t, worldsDir, jarsDir, "1.21.4")
	if got := readCalls(t, calls); !reflect.DeepEqual(got, []string{"is-active --quiet minecraft@survival.service"}) {
		t.Errorf("systemctl calls = %v", got)
	}
}

func TestUpgradeWorldRunning(t *testing.T) {
	worldsDir, jarsDir := setupUpgrade(t)
	calls := fakeSystemctl(t, "exit 0")

	if _, err := UpgradeWorld("survival", UpgradeOptions{TargetVersion: "1.21.4"}); err == nil || !strings.Contains(err.Error(), "--stop") {
		t.Fatalf("UpgradeWorld(running) = %v, want a --stop hint", err)
	}
	assertJar(t, worldsDir, jarsDir, "1.21.1")

	res, err := UpgradeWorld("survival", UpgradeOptions{TargetVersion: "1.21.4", StopService: true})
	if err != nil {
		t.Fatalf("UpgradeWorld(--stop) = %v", err)
	}
	if !res.ServiceStopped {
		t.Error("ServiceStopped = false")
	}
	assertJar(t, worldsDir, jarsDir, "1.21.4")
	got := readCalls(t, calls)
	if last := got[len(got)-1]; last != "stop minecraft@survival.service" {
		t.Errorf("last systemctl call = %q, want stop", last)
	}
}

func TestUpgradeWorldStopFails(t *testing.T) {
	worldsDir, jarsDir := setupUpgrade(t)
	fakeSystemctl(t, `[ "$1" = stop ] && exit 1; exit 0`)
	if _, err := UpgradeWorld("survival", UpgradeOptions{TargetVersion: "1.21.4", StopService: true}); err == nil || !strings.Contains(err.Error(), "failed to stop") {
		t.Errorf("UpgradeWorld() = %v, want failed to stop", err)
	}
	assertJar(t, worldsDir, jarsDir, "1.21.1")
}

func TestUpgradeWorldRefusals(t *testing.T) {
	worldsDir, jarsDir := setupUpgrade(t)
	fakeSystemctl(t, "exit 3")
	tests := []struct {
		world, version, want string
	}{
		{"missing", "1.21.4", "world not found"},
		{"survival", "1.21.1", "not newer"},
		{"survival", "1.20", "not newer"},
		{"survival", "1.22", "target JAR not found"},
		{"survival", "bogus", "compare versions"},
	}
	for _, tt := range tests {
		_, err := UpgradeWorld(tt.world, UpgradeOptions{TargetVersion: tt.version})
		if err == nil || !strings.Contains(err.Error(), tt.want) {
			t.Errorf("UpgradeWorld(%s, %s) = %v, want %q", tt.world, tt.version, err, tt.want)
		}
	}
	assertJar(t, worldsDir, jarsDir, "1.21.1")
}

func TestRunningWorlds(t *testing.T) {
	setupWorlds(t)
	calls := fakeSystemctl(t, `printf '%s\n' \
  'minecraft@default.service loaded active running Minecraft Server default' \
  'minecraft@old.service loaded activating auto-restart Minecraft Server old'`)

	got, err := RunningWorlds()
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"default", "old"}; !reflect.DeepEqual(got, want) {
		t.Errorf("RunningWorlds() = %v, want %v", got, want)
	}
	call := readCalls(t, calls)[0]
	if !strings.Contains(call, "--state=active,activating,deactivating,reloading") || !strings.HasSuffix(call, "minecraft@*.service") {
		t.Errorf("list-units call = %q", call)
	}

	fakeSystemctl(t, "exit 1")
	if _, err := RunningWorlds(); err == nil {
		t.Error("RunningWorlds() with systemctl failing = nil error")
	}
}

// startRcon starts an in-process RCON server that answers commands with
// reply(command), and points the config at it.
func startRcon(t *testing.T, reply func(cmd string) string) {
	t.Helper()
	srv := rcontest.NewServer(
		rcontest.SetSettings(rcontest.Settings{Password: "secret"}),
		rcontest.SetCommandHandler(func(c *rcontest.Context) {
			_, _ = gorcon.NewPacket(gorcon.SERVERDATA_RESPONSE_VALUE, c.Request().ID, reply(c.Request().Body())).WriteTo(c.Conn())
		}),
	)
	t.Cleanup(srv.Close)
	host, port, err := net.SplitHostPort(srv.Addr())
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("RCON_HOST", host)
	t.Setenv("RCON_PORT", port)
	t.Setenv("RCON_PASSWORD", "secret")
	if err := config.Init(""); err != nil {
		t.Fatal(err)
	}
}

func TestRconPlayersOnline(t *testing.T) {
	setupWorlds(t)
	startRcon(t, func(cmd string) string {
		if cmd == "list" {
			return "There are 3 of a max of 20 players online: a, b, c"
		}
		return "Unknown command"
	})
	n, err := rconPlayersOnline()
	if err != nil || n != 3 {
		t.Errorf("rconPlayersOnline() = %d, %v; want 3", n, err)
	}
	if err := waitForRcon("survival", time.Second); err != nil {
		t.Errorf("waitForRcon() = %v, want nil (server answers)", err)
	}
}

func TestRconPlayersOnlineUnreachable(t *testing.T) {
	setupWorlds(t)
	// Grab a free port and close it so nothing is listening
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := l.Addr().(*net.TCPAddr).Port
	l.Close()
	t.Setenv("RCON_HOST", "127.0.0.1")
	t.Setenv("RCON_PORT", strconv.Itoa(port))
	t.Setenv("RCON_PASSWORD", "secret")
	if err := config.Init(""); err != nil {
		t.Fatal(err)
	}
	if _, err := rconPlayersOnline(); err == nil {
		t.Error("rconPlayersOnline() with no server = nil error")
	}
}
