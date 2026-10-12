package backup

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// recordRunsErr is recordRuns with the result of each run chosen by fn.
func recordRunsErr(t *testing.T, fn func(args []string) error) *[][]string {
	t.Helper()
	var got [][]string
	orig := run
	run = func(cmd *exec.Cmd) error {
		got = append(got, cmd.Args[1:])
		return fn(cmd.Args[1:])
	}
	t.Cleanup(func() { run = orig })
	return &got
}

func TestInitRepository(t *testing.T) {
	c := &Config{Repository: "s3:example/bucket", Password: "pw"}

	// The repo exists: nothing to do
	fakeRestic(t, `[ "$*" = "cat config" ] && exit 0; exit 1`)
	got := recordRuns(t)
	if err := c.InitRepository(); err != nil {
		t.Fatalf("InitRepository() = %v", err)
	}
	if len(*got) != 0 {
		t.Errorf("ran %v, want nothing", *got)
	}

	// No repo yet: restic init
	fakeRestic(t, `echo "Fatal: repository does not exist" >&2; exit 10`)
	if err := c.InitRepository(); err != nil {
		t.Fatalf("InitRepository() = %v", err)
	}
	if len(*got) != 1 || (*got)[0][0] != "init" {
		t.Errorf("ran %v, want [init]", *got)
	}
}

func TestCreateAllWorlds(t *testing.T) {
	fakeRestic(t, "exit 0")
	got := recordRuns(t)
	c := &Config{Repository: "r", Password: "pw", WorldsDir: "/srv/minecraft-server"}

	for _, world := range []string{"", "all"} {
		*got = nil
		if err := c.Create(world); err != nil {
			t.Fatalf("Create(%q) = %v", world, err)
		}
		if len(*got) != 1 {
			t.Fatalf("Create(%q) ran %v", world, *got)
		}
		args := strings.Join((*got)[0], " ")
		for _, want := range []string{"backup /srv/minecraft-server ", "--tag all", "--host minecraft", "--exclude /srv/minecraft-server/caddy"} {
			if !strings.Contains(args, want) {
				t.Errorf("Create(%q) args %q missing %q", world, args, want)
			}
		}
	}
}

func TestCreateOneWorld(t *testing.T) {
	fakeRestic(t, "exit 0")
	got := recordRuns(t)
	worldsDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(worldsDir, "survival", "world"), 0755); err != nil {
		t.Fatal(err)
	}
	c := &Config{Repository: "r", Password: "pw", WorldsDir: worldsDir}

	if err := c.Create("survival"); err != nil {
		t.Fatalf("Create() = %v", err)
	}
	args := (*got)[0]
	if args[1] != worldsDir+"/survival/world" || !slices.Contains(args, "survival") {
		t.Errorf("args = %v", args)
	}
	if slices.Contains(args, filepath.Join(worldsDir, "caddy")) {
		t.Errorf("a single-world backup excludes caddy: %v", args)
	}

	if err := c.Create("missing"); err == nil || !strings.Contains(err.Error(), "world path not found") {
		t.Errorf("Create(missing) = %v", err)
	}
}

func TestCreateErrors(t *testing.T) {
	c := &Config{Repository: "r", Password: "pw", WorldsDir: t.TempDir()}

	// No repo and init fails
	fakeRestic(t, "exit 1")
	recordRunsErr(t, func([]string) error { return errors.New("init failed") })
	if err := c.Create(""); err == nil || !strings.Contains(err.Error(), "failed to initialize repository") {
		t.Errorf("Create() = %v, want an init error", err)
	}

	// The backup itself fails
	fakeRestic(t, "exit 0")
	recordRunsErr(t, func([]string) error { return errors.New("exit status 3") })
	if err := c.Create(""); err == nil || err.Error() != "exit status 3" {
		t.Errorf("Create() = %v, want the restic error", err)
	}
}

func TestListStatsCheck(t *testing.T) {
	got := recordRuns(t)
	c := &Config{Repository: "r", Password: "pw"}
	if err := c.List(""); err != nil {
		t.Fatal(err)
	}
	if err := c.List("survival"); err != nil {
		t.Fatal(err)
	}
	if err := c.Stats(); err != nil {
		t.Fatal(err)
	}
	if err := c.Check(); err != nil {
		t.Fatal(err)
	}
	want := []string{"snapshots", "snapshots --tag survival", "stats", "check"}
	var ran []string
	for _, args := range *got {
		ran = append(ran, strings.Join(args, " "))
	}
	if !slices.Equal(ran, want) {
		t.Errorf("ran %q, want %q", ran, want)
	}
}

func TestRestore(t *testing.T) {
	got := recordRuns(t)
	c := &Config{Repository: "r", Password: "pw", WorldsDir: "/srv/minecraft-server", RetryLock: "30m"}
	target := filepath.Join(t.TempDir(), "restore")

	if err := c.Restore("", target); err != nil {
		t.Fatalf("Restore() = %v", err)
	}
	if info, err := os.Stat(target); err != nil || !info.IsDir() {
		t.Errorf("target dir not created: %v", err)
	}
	want := []string{"--retry-lock", "30m", "restore", "latest", "--target", target}
	if !slices.Equal((*got)[0], want) {
		t.Errorf("args = %v, want %v", (*got)[0], want)
	}

	// Restoring in place doesn't try to create /
	if err := c.Restore("abc123", "/"); err != nil {
		t.Fatalf("Restore(/) = %v", err)
	}
	if args := (*got)[1]; args[3] != "abc123" || args[5] != "/" {
		t.Errorf("args = %v", args)
	}
}

func TestRestoreErrors(t *testing.T) {
	recordRunsErr(t, func([]string) error { return errors.New("restic failed") })
	c := &Config{Repository: "r", Password: "pw"}
	if err := c.Restore("latest", t.TempDir()); err == nil || err.Error() != "restic failed" {
		t.Errorf("Restore() = %v, want the restic error", err)
	}

	file := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(file, nil, 0644); err != nil {
		t.Fatal(err)
	}
	if err := c.Restore("latest", filepath.Join(file, "sub")); err == nil || !strings.Contains(err.Error(), "target directory") {
		t.Errorf("Restore(under a file) = %v", err)
	}
}

func TestResticInstalledAndVersion(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	if IsResticInstalled() {
		t.Error("IsResticInstalled() = true with no restic on PATH")
	}
	if _, err := GetResticVersion(); err == nil {
		t.Error("GetResticVersion() = nil error with no restic")
	}

	fakeRestic(t, `echo "restic 0.18.0 compiled with go1.24 on linux/amd64"`)
	if !IsResticInstalled() {
		t.Error("IsResticInstalled() = false")
	}
	v, err := GetResticVersion()
	if err != nil || v != "restic 0.18.0 compiled with go1.24 on linux/amd64" {
		t.Errorf("GetResticVersion() = %q, %v", v, err)
	}
}
