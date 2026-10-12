package doctor

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fakeServer is a healthy server with one world, "alpha", running
type fakeServer struct {
	worldsDir   string
	owner       int
	mcUID       int
	mcUIDErr    error
	unitStates  map[string]string // default "enabled"
	activeState map[string]string // default "active"
	running     []string
	resticEnv   error
	resticPing  error
	rconPing    error
	noRestic    bool
}

func newFakeServer(t *testing.T) *fakeServer {
	t.Helper()
	dir := t.TempDir()
	mkWorld(t, dir, "alpha")
	mkWorld(t, dir, "beta")
	if err := os.MkdirAll(filepath.Join(dir, "maps"), 0755); err != nil {
		t.Fatal(err)
	}
	return &fakeServer{
		worldsDir:   dir,
		owner:       MinecraftUID,
		mcUID:       MinecraftUID,
		unitStates:  map[string]string{},
		activeState: map[string]string{},
		running:     []string{"alpha"},
	}
}

func mkWorld(t *testing.T, worldsDir, name string) {
	t.Helper()
	level := filepath.Join(worldsDir, name, "world", "level.dat")
	if err := os.MkdirAll(filepath.Dir(level), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(level, []byte("nbt"), 0644); err != nil {
		t.Fatal(err)
	}
}

func (f *fakeServer) env(t *testing.T) Env {
	unmined := filepath.Join(t.TempDir(), "unmined-cli")
	if err := os.WriteFile(unmined, []byte("#!/bin/sh\n"), 0755); err != nil {
		t.Fatal(err)
	}
	return Env{
		WorldsDir:          f.worldsDir,
		OnServer:           true,
		Owner:              func(string) (int, error) { return f.owner, nil },
		LookupMinecraftUID: func() (int, error) { return f.mcUID, f.mcUIDErr },
		UnitFileState: func(unit string) (string, error) {
			if s, ok := f.unitStates[unit]; ok {
				return s, nil
			}
			return "enabled", nil
		},
		ActiveState: func(unit string) string {
			if s, ok := f.activeState[unit]; ok {
				return s
			}
			return "active"
		},
		RunningWorlds:   func() ([]string, error) { return f.running, nil },
		ResticInstalled: func() bool { return !f.noRestic },
		ResticEnv:       func() error { return f.resticEnv },
		ResticPing:      func() error { return f.resticPing },
		RconPing:        func() error { return f.rconPing },
		UnminedPath:     unmined,
	}
}

func find(t *testing.T, results []Result, name string) Result {
	t.Helper()
	for _, r := range results {
		if r.Name == name {
			return r
		}
	}
	t.Fatalf("no %q result in %+v", name, results)
	return Result{}
}

func assertStatus(t *testing.T, results []Result, name string, want Status) Result {
	t.Helper()
	r := find(t, results, name)
	if r.Status != want {
		t.Errorf("%s = %s (%s), want %s", name, r.Status, r.Detail, want)
	}
	return r
}

func TestRunHealthy(t *testing.T) {
	f := newFakeServer(t)
	results := Run(f.env(t))

	for _, r := range results {
		if r.Status != Pass {
			t.Errorf("%s = %s (%s), want PASS", r.Name, r.Status, r.Detail)
		}
	}
	if r := find(t, results, "worlds"); !strings.Contains(r.Detail, "alpha, beta") {
		t.Errorf("worlds detail = %q, want both worlds (and not maps/)", r.Detail)
	}
	if Failed(results, true) {
		t.Error("Failed(strict) = true for a healthy server")
	}
}

func TestConfigError(t *testing.T) {
	f := newFakeServer(t)
	env := f.env(t)
	env.ConfigErr = errors.New("bad yaml")
	assertStatus(t, Run(env), "config", Fail)
}

func TestWorldsDirOwner(t *testing.T) {
	f := newFakeServer(t)
	f.owner = 0
	r := assertStatus(t, Run(f.env(t)), "worlds dir", Fail)
	if !strings.Contains(r.Detail, "uid 0") {
		t.Errorf("detail = %q, want the actual owner", r.Detail)
	}
}

func TestLegacyMinecraftUID(t *testing.T) {
	f := newFakeServer(t)
	f.mcUID, f.owner = 997, 997
	results := Run(f.env(t))
	assertStatus(t, results, "minecraft user", Warn)
	assertStatus(t, results, "worlds dir", Pass)
}

func TestWorldsDirMissing(t *testing.T) {
	f := newFakeServer(t)
	f.worldsDir = filepath.Join(f.worldsDir, "nope")
	assertStatus(t, Run(f.env(t)), "worlds dir", Fail)
}

func TestWorldWithoutLevelDat(t *testing.T) {
	f := newFakeServer(t)
	broken := filepath.Join(f.worldsDir, "broken")
	if err := os.MkdirAll(broken, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(broken, "server.properties"), nil, 0644); err != nil {
		t.Fatal(err)
	}
	// A directory that isn't a server dir is ignored
	if err := os.MkdirAll(filepath.Join(f.worldsDir, "scratch"), 0755); err != nil {
		t.Fatal(err)
	}

	results := Run(f.env(t))
	assertStatus(t, results, "world broken", Fail)
	for _, r := range results {
		if r.Name == "world scratch" || r.Name == "timers broken" {
			t.Errorf("unexpected result %+v", r)
		}
	}
}

func TestUnits(t *testing.T) {
	f := newFakeServer(t)
	f.unitStates["autoshutdown.timer"] = "not-found"
	f.unitStates["minecraft-world-prune.timer"] = "disabled"
	r := assertStatus(t, Run(f.env(t)), "units", Fail)
	if !strings.Contains(r.Detail, "autoshutdown.timer not installed") || !strings.Contains(r.Detail, "minecraft-world-prune.timer is disabled") {
		t.Errorf("units detail = %q", r.Detail)
	}
}

func TestWorldTimers(t *testing.T) {
	t.Run("not enabled fails", func(t *testing.T) {
		f := newFakeServer(t)
		f.unitStates["minecraft-world-backup@beta.timer"] = "disabled"
		r := assertStatus(t, Run(f.env(t)), "timers beta", Fail)
		if !strings.Contains(r.Detail, "world register beta") {
			t.Errorf("detail = %q, want a hint to register", r.Detail)
		}
	})
	t.Run("inactive backup timer warns", func(t *testing.T) {
		f := newFakeServer(t)
		f.activeState["minecraft-map-backup@beta.timer"] = "inactive"
		assertStatus(t, Run(f.env(t)), "timers beta", Warn)
	})
	t.Run("map-build timer only needs to be active for the running world", func(t *testing.T) {
		f := newFakeServer(t)
		f.activeState["minecraft-map-build@beta.timer"] = "inactive"
		f.activeState["minecraft-map-build@alpha.timer"] = "inactive"
		results := Run(f.env(t))
		assertStatus(t, results, "timers beta", Pass)
		assertStatus(t, results, "timers alpha", Warn)
	})
}

func TestRcon(t *testing.T) {
	f := newFakeServer(t)
	f.rconPing = errors.New("connection refused")
	assertStatus(t, Run(f.env(t)), "rcon", Fail)

	f.running = nil
	results := Run(f.env(t))
	assertStatus(t, results, "rcon", Skip)
	assertStatus(t, results, "running world", Pass)
}

func TestRestic(t *testing.T) {
	f := newFakeServer(t)
	f.resticPing = errors.New("timed out after 30s")
	results := Run(f.env(t))
	assertStatus(t, results, "restic env", Pass)
	assertStatus(t, results, "restic repo", Fail)

	f.resticEnv = errors.New("RESTIC_PASSWORD not set")
	results = Run(f.env(t))
	assertStatus(t, results, "restic env", Fail)
	assertStatus(t, results, "restic repo", Skip)

	f.noRestic = true
	assertStatus(t, Run(f.env(t)), "restic", Fail)
}

func TestUnmined(t *testing.T) {
	f := newFakeServer(t)
	env := f.env(t)
	env.UnminedPath = filepath.Join(t.TempDir(), "missing")
	assertStatus(t, Run(env), "unmined", Fail)

	notExec := filepath.Join(t.TempDir(), "unmined-cli")
	if err := os.WriteFile(notExec, nil, 0644); err != nil {
		t.Fatal(err)
	}
	env.UnminedPath = notExec
	assertStatus(t, Run(env), "unmined", Fail)
}

// Away from a server, missing server bits warn or skip rather than fail
func TestNotOnServer(t *testing.T) {
	f := newFakeServer(t)
	f.worldsDir = filepath.Join(f.worldsDir, "nope")
	f.noRestic = true
	env := f.env(t)
	env.OnServer = false
	env.UnminedPath = filepath.Join(t.TempDir(), "missing")
	called := false
	env.UnitFileState = func(string) (string, error) { called = true; return "", errors.New("no systemctl") }
	env.RunningWorlds = func() ([]string, error) { called = true; return nil, errors.New("no systemctl") }

	results := Run(env)
	if called {
		t.Error("systemd was queried off a server")
	}
	assertStatus(t, results, "worlds dir", Warn)
	assertStatus(t, results, "units", Skip)
	assertStatus(t, results, "rcon", Skip)
	assertStatus(t, results, "restic", Warn)
	assertStatus(t, results, "unmined", Warn)
	if Failed(results, false) {
		t.Error("Failed() = true off a server, want warnings only")
	}
	if !Failed(results, true) {
		t.Error("Failed(strict) = false with warnings")
	}
}

func TestFailed(t *testing.T) {
	pass := []Result{{Status: Pass}, {Status: Skip}}
	warn := append(pass, Result{Status: Warn})
	fail := append(warn, Result{Status: Fail})

	cases := []struct {
		name    string
		results []Result
		strict  bool
		want    bool
	}{
		{"pass", pass, false, false},
		{"pass strict", pass, true, false},
		{"warn", warn, false, false},
		{"warn strict", warn, true, true},
		{"fail", fail, false, true},
		{"fail strict", fail, true, true},
	}
	for _, c := range cases {
		if got := Failed(c.results, c.strict); got != c.want {
			t.Errorf("%s: Failed() = %v, want %v", c.name, got, c.want)
		}
	}
}
