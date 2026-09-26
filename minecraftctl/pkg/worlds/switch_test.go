package worlds

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

// makeWorld creates a world as `world create` leaves it: eula.txt,
// server.properties with RCON enabled and server.jar linked into a jars dir,
// but no world/level.dat yet.
func makeWorld(t *testing.T, worldsDir, name string) string {
	t.Helper()
	dir := filepath.Join(worldsDir, name)
	jars := filepath.Join(worldsDir, "..", "jars")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(jars, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(dir, "eula.txt"), "#By changing the setting below to TRUE...\neula=true\n")
	writeFile(t, filepath.Join(dir, "server.properties"), "enable-rcon=true\nrcon.port=25575\nlevel-name=world\n")
	jar := filepath.Join(jars, "minecraft_server_1.21.jar")
	writeFile(t, jar, "")
	if err := os.Symlink(jar, filepath.Join(dir, "server.jar")); err != nil {
		t.Fatal(err)
	}
	return dir
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestCheckStartable(t *testing.T) {
	tests := []struct {
		name    string
		world   string
		modify  func(dir string)
		wantErr string // "" for startable
	}{
		{name: "fresh world without level.dat", world: "fresh"},
		{name: "dotted name", world: "world.bak-1.19.2"},
		{name: "path traversal", world: "../escape", wantErr: "not a valid world name"},
		{name: "hidden dir", world: ".hidden", wantErr: "not a valid world name"},
		{name: "empty name", world: "", wantErr: "not a valid world name"},
		{name: "missing world", world: "nope", modify: func(dir string) { os.RemoveAll(dir) }, wantErr: "world not found"},
		{name: "eula false", world: "old", modify: func(dir string) {
			os.WriteFile(filepath.Join(dir, "eula.txt"), []byte("eula=false\n"), 0o644)
		}, wantErr: "doesn't accept the EULA"},
		{name: "eula missing", world: "old", modify: func(dir string) {
			os.Remove(filepath.Join(dir, "eula.txt"))
		}, wantErr: "doesn't accept the EULA"},
		{name: "rcon disabled", world: "old", modify: func(dir string) {
			os.WriteFile(filepath.Join(dir, "server.properties"), []byte("enable-rcon=false\n"), 0o644)
		}, wantErr: "RCON not enabled"},
		{name: "server.properties missing", world: "old", modify: func(dir string) {
			os.Remove(filepath.Join(dir, "server.properties"))
		}, wantErr: "RCON not enabled"},
		{name: "jar missing", world: "old", modify: func(dir string) {
			os.Remove(filepath.Join(dir, "server.jar"))
			os.Symlink(filepath.Join(dir, "minecraft_server_1.16.4.jar"), filepath.Join(dir, "server.jar"))
		}, wantErr: "points to a missing jar"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			worldsDir := filepath.Join(t.TempDir(), "srv")
			if worldNameRe.MatchString(tt.world) {
				dir := makeWorld(t, worldsDir, tt.world)
				if tt.modify != nil {
					tt.modify(dir)
				}
			} else {
				// A startable-looking dir the name would escape to
				makeWorld(t, worldsDir, "escape")
			}

			err := checkStartable(worldsDir, tt.world)
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("checkStartable() = %v, want nil", err)
				}
				return
			}
			if !errors.Is(err, ErrNotStartable) {
				t.Fatalf("checkStartable() = %v, want ErrNotStartable", err)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("checkStartable() = %q, want it to contain %q", err, tt.wantErr)
			}
		})
	}
}

func TestParsePlayerCount(t *testing.T) {
	tests := []struct {
		out  string
		want int
	}{
		{"There are 0 of a max of 20 players online: ", 0},
		{"There are 2 of a max of 20 players online: alice, bob", 2},
		{"There are 3/20 players online:", 3},
	}
	for _, tt := range tests {
		got, err := parsePlayerCount(tt.out)
		if err != nil || got != tt.want {
			t.Errorf("parsePlayerCount(%q) = %d, %v; want %d", tt.out, got, err, tt.want)
		}
	}
	if _, err := parsePlayerCount("Unknown command"); err == nil {
		t.Error("parsePlayerCount(garbage) = nil error, want error")
	}
}

// fakeSystem records what a switch does. Worlds in failing never come up.
type fakeSystem struct {
	units   []worldUnit
	players int
	playErr error
	failing map[string]bool
	calls   []string
}

func newTestSwitcher(t *testing.T, f *fakeSystem, worlds ...string) *switcher {
	t.Helper()
	worldsDir := filepath.Join(t.TempDir(), "srv")
	for _, w := range worlds {
		makeWorld(t, worldsDir, w)
	}
	return &switcher{
		worldsDir: worldsDir,
		lock: func() (func(), error) {
			f.calls = append(f.calls, "lock")
			return func() { f.calls = append(f.calls, "unlock") }, nil
		},
		units: func() ([]worldUnit, error) { return f.units, nil },
		start: func(w string) error { f.calls = append(f.calls, "start "+w); return nil },
		stop:  func(w string) error { f.calls = append(f.calls, "stop "+w); return nil },
		playersOnline: func() (int, error) {
			f.calls = append(f.calls, "players")
			return f.players, f.playErr
		},
		say: func(msg string) error { f.calls = append(f.calls, "say"); return nil },
		waitUp: func(w string, _ time.Duration) error {
			f.calls = append(f.calls, "wait "+w)
			if f.failing[w] {
				return errors.New("no RCON")
			}
			return nil
		},
		sleep: func(d time.Duration) { f.calls = append(f.calls, "sleep "+d.String()) },
	}
}

func assertCalls(t *testing.T, f *fakeSystem, want ...string) {
	t.Helper()
	if !reflect.DeepEqual(f.calls, want) {
		t.Errorf("calls:\n got %q\nwant %q", f.calls, want)
	}
}

func TestSwitchFromRunningWorld(t *testing.T) {
	f := &fakeSystem{units: []worldUnit{{"default", "active"}}}
	s := newTestSwitcher(t, f, "default", "old")

	res, err := s.switchWorld("old", SwitchOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if !res.Started || !reflect.DeepEqual(res.Stopped, []string{"default"}) {
		t.Errorf("result = %+v", res)
	}
	assertCalls(t, f, "lock", "players", "stop default", "start old", "wait old", "unlock")
}

func TestSwitchWithNothingRunning(t *testing.T) {
	f := &fakeSystem{}
	s := newTestSwitcher(t, f, "old")

	if _, err := s.switchWorld("old", SwitchOptions{}); err != nil {
		t.Fatal(err)
	}
	// Nobody can be online, so no player check (RCON isn't up at boot)
	assertCalls(t, f, "lock", "start old", "wait old", "unlock")
}

func TestSwitchToRunningWorldIsNoop(t *testing.T) {
	f := &fakeSystem{units: []worldUnit{{"old", "active"}}}
	s := newTestSwitcher(t, f, "old")

	res, err := s.switchWorld("old", SwitchOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Started || len(res.Stopped) != 0 {
		t.Errorf("result = %+v, want no-op", res)
	}
	assertCalls(t, f, "lock", "unlock")
}

func TestSwitchStopsEveryOtherWorld(t *testing.T) {
	f := &fakeSystem{units: []worldUnit{{"default", "active"}, {"old", "active"}}}
	s := newTestSwitcher(t, f, "default", "old")

	res, err := s.switchWorld("old", SwitchOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Started {
		t.Error("old was already running, want Started = false")
	}
	assertCalls(t, f, "lock", "players", "stop default", "unlock")
}

func TestSwitchRestartsCrashLoopingTarget(t *testing.T) {
	f := &fakeSystem{units: []worldUnit{{"old", "activating"}}}
	s := newTestSwitcher(t, f, "old")

	if _, err := s.switchWorld("old", SwitchOptions{}); err != nil {
		t.Fatal(err)
	}
	// Not up, so no player check; stopped and started fresh
	assertCalls(t, f, "lock", "stop old", "start old", "wait old", "unlock")
}

func TestSwitchRefusesNotStartable(t *testing.T) {
	f := &fakeSystem{units: []worldUnit{{"default", "active"}}}
	s := newTestSwitcher(t, f, "default")

	_, err := s.switchWorld("nope", SwitchOptions{Force: true})
	if !errors.Is(err, ErrNotStartable) {
		t.Fatalf("err = %v, want ErrNotStartable", err)
	}
	assertCalls(t, f) // checked before taking the lock
}

func TestSwitchRefusesWithPlayersOnline(t *testing.T) {
	f := &fakeSystem{units: []worldUnit{{"default", "active"}}, players: 2}
	s := newTestSwitcher(t, f, "default", "old")

	_, err := s.switchWorld("old", SwitchOptions{})
	if !errors.Is(err, ErrSwitchRefused) {
		t.Fatalf("err = %v, want ErrSwitchRefused", err)
	}
	assertCalls(t, f, "lock", "players", "unlock")
}

func TestSwitchRefusesWhenPlayerCountUnknown(t *testing.T) {
	f := &fakeSystem{units: []worldUnit{{"default", "active"}}, playErr: errors.New("connection refused")}
	s := newTestSwitcher(t, f, "default", "old")

	_, err := s.switchWorld("old", SwitchOptions{})
	if !errors.Is(err, ErrSwitchRefused) {
		t.Fatalf("err = %v, want ErrSwitchRefused", err)
	}
}

func TestSwitchForceWarnsPlayers(t *testing.T) {
	f := &fakeSystem{units: []worldUnit{{"default", "active"}}, players: 2}
	s := newTestSwitcher(t, f, "default", "old")

	if _, err := s.switchWorld("old", SwitchOptions{Force: true, WarnDelay: 10 * time.Second}); err != nil {
		t.Fatal(err)
	}
	assertCalls(t, f, "lock", "players", "say", "sleep 10s", "stop default", "start old", "wait old", "unlock")
}

func TestSwitchDryRunChangesNothing(t *testing.T) {
	f := &fakeSystem{units: []worldUnit{{"default", "active"}}, players: 2}
	s := newTestSwitcher(t, f, "default", "old")

	res, err := s.switchWorld("old", SwitchOptions{DryRun: true, Force: true})
	if err != nil {
		t.Fatal(err)
	}
	if !res.Started || !reflect.DeepEqual(res.Stopped, []string{"default"}) {
		t.Errorf("result = %+v", res)
	}
	assertCalls(t, f, "players")

	// Without --force, a dry run reports the refusal
	f.calls = nil
	if _, err := s.switchWorld("old", SwitchOptions{DryRun: true}); !errors.Is(err, ErrSwitchRefused) {
		t.Fatalf("err = %v, want ErrSwitchRefused", err)
	}
}

func TestSwitchRollsBackWhenTargetFails(t *testing.T) {
	f := &fakeSystem{units: []worldUnit{{"default", "active"}}, failing: map[string]bool{"old": true}}
	s := newTestSwitcher(t, f, "default", "old")

	_, err := s.switchWorld("old", SwitchOptions{})
	if !errors.Is(err, ErrStartFailed) {
		t.Fatalf("err = %v, want ErrStartFailed", err)
	}
	assertCalls(t, f, "lock", "players", "stop default", "start old", "wait old",
		"stop old", "start default", "wait default", "unlock")
}

func TestSwitchFailsWithNothingToRollBackTo(t *testing.T) {
	f := &fakeSystem{failing: map[string]bool{"old": true}}
	s := newTestSwitcher(t, f, "old")

	_, err := s.switchWorld("old", SwitchOptions{})
	if !errors.Is(err, ErrStartFailed) {
		t.Fatalf("err = %v, want ErrStartFailed", err)
	}
	assertCalls(t, f, "lock", "start old", "wait old", "stop old", "unlock")
}

func TestSwitchRollbackFails(t *testing.T) {
	f := &fakeSystem{units: []worldUnit{{"default", "active"}}, failing: map[string]bool{"old": true, "default": true}}
	s := newTestSwitcher(t, f, "default", "old")

	_, err := s.switchWorld("old", SwitchOptions{})
	if !errors.Is(err, ErrRollbackFailed) {
		t.Fatalf("err = %v, want ErrRollbackFailed", err)
	}
	assertCalls(t, f, "lock", "players", "stop default", "start old", "wait old",
		"stop old", "start default", "wait default", "stop default", "unlock")
}
