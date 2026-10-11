package backup

import (
	"os"
	"os/user"
	"path/filepath"
	"slices"
	"testing"
)

func TestBackupArgsUseFixedHost(t *testing.T) {
	args := backupArgs("/srv/minecraft-server/default/world", "default")

	i := slices.Index(args, "--host")
	if i < 0 || i+1 >= len(args) || args[i+1] != Host {
		t.Errorf("backupArgs() = %v, want --host %s", args, Host)
	}
	if args[0] != "backup" || args[1] != "/srv/minecraft-server/default/world" {
		t.Errorf("backupArgs() = %v, want backup <path> first", args)
	}
}

func TestOwnedBy(t *testing.T) {
	dir := t.TempDir()
	if !ownedBy(dir, os.Getuid()) {
		t.Errorf("ownedBy(%s, own uid) = false, want true", dir)
	}
	if ownedBy(dir, os.Getuid()+1) {
		t.Errorf("ownedBy(%s, other uid) = true, want false", dir)
	}
	if ownedBy(filepath.Join(dir, "missing"), os.Getuid()) {
		t.Error("ownedBy(missing dir) = true, want false")
	}
}

func TestCommandRunsAsCredential(t *testing.T) {
	c := &Config{Repository: "s3:example/bucket", Password: "pw"}
	cred := &Credential{Uid: 996, Gid: 996, Groups: []uint32{996}, Home: "/home/minecraft"}

	cmd := c.command(cred, "snapshots")

	if cmd.SysProcAttr == nil || cmd.SysProcAttr.Credential == nil {
		t.Fatal("command() didn't set a credential")
	}
	if got := cmd.SysProcAttr.Credential.Uid; got != 996 {
		t.Errorf("credential uid = %d, want 996", got)
	}
	if !slices.Contains(cmd.Env, "HOME=/home/minecraft") {
		t.Errorf("env missing HOME=/home/minecraft: %v", cmd.Env)
	}
}

func TestCommandWithoutCredentialRunsAsCaller(t *testing.T) {
	c := &Config{Repository: "s3:example/bucket", Password: "pw"}

	cmd := c.command(nil, "snapshots")

	if cmd.SysProcAttr != nil {
		t.Errorf("command(nil) set SysProcAttr: %+v", cmd.SysProcAttr)
	}
	if !slices.Contains(cmd.Env, "RESTIC_REPOSITORY=s3:example/bucket") {
		t.Errorf("env missing RESTIC_REPOSITORY: %v", cmd.Env)
	}
}

func TestLookupCredentialCurrentUser(t *testing.T) {
	u, err := user.Current()
	if err != nil {
		t.Skip(err)
	}
	cred, err := LookupCredential(u.Username)
	if err != nil {
		t.Fatalf("LookupCredential(%s): %v", u.Username, err)
	}
	if int(cred.Uid) != os.Getuid() || cred.Home != u.HomeDir {
		t.Errorf("LookupCredential(%s) = %+v, want uid %d home %s", u.Username, cred, os.Getuid(), u.HomeDir)
	}
}

func TestRetryLockFromEnv(t *testing.T) {
	tests := []struct {
		env     string
		want    string
		wantErr bool
	}{
		{"", DefaultRetryLock, false},
		{"5m", "5m", false},
		{" 90s ", "90s", false},
		{"1h30m", "1h30m", false},
		{"0", "", false},
		{"0s", "", false},
		{"soon", "", true},
		{"30", "", true},
		{"-5m", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.env, func(t *testing.T) {
			t.Setenv(RetryLockEnv, tt.env)
			got, err := retryLockFromEnv()
			if (err != nil) != tt.wantErr {
				t.Fatalf("retryLockFromEnv() error = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("retryLockFromEnv() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestLoadConfigRetryLock(t *testing.T) {
	t.Setenv("MC_WORLD_BUCKET", "bucket")
	t.Setenv("RESTIC_PASSWORD", "pw")

	t.Setenv(RetryLockEnv, "")
	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig(): %v", err)
	}
	if cfg.RetryLock != "30m" {
		t.Errorf("default RetryLock = %q, want 30m", cfg.RetryLock)
	}

	t.Setenv(RetryLockEnv, "10m")
	if cfg, err = LoadConfig(); err != nil || cfg.RetryLock != "10m" {
		t.Errorf("LoadConfig() with %s=10m = %+v, %v; want RetryLock 10m", RetryLockEnv, cfg, err)
	}

	t.Setenv(RetryLockEnv, "0")
	if cfg, err = LoadConfig(); err != nil || cfg.RetryLock != "" {
		t.Errorf("LoadConfig() with %s=0 = %+v, %v; want RetryLock off", RetryLockEnv, cfg, err)
	}

	t.Setenv(RetryLockEnv, "forever")
	if _, err = LoadConfig(); err == nil {
		t.Errorf("LoadConfig() with %s=forever: want an error", RetryLockEnv)
	}
}

func TestCommandAddsRetryLock(t *testing.T) {
	c := &Config{Repository: "s3:example/bucket", Password: "pw", RetryLock: "30m"}

	for _, args := range [][]string{
		backupArgs("/srv/minecraft-server/default/world", "default"),
		{"forget", "--keep-daily", "7", "--prune"},
		{"check"},
		{"snapshots"},
	} {
		cmd := c.command(nil, args...)
		want := append([]string{"restic", "--retry-lock", "30m"}, args...)
		if !slices.Equal(cmd.Args, want) {
			t.Errorf("command(%v).Args = %v, want %v", args, cmd.Args, want)
		}
	}
}

func TestCommandWithoutRetryLock(t *testing.T) {
	c := &Config{Repository: "s3:example/bucket", Password: "pw"}

	cmd := c.command(nil, "snapshots")

	if slices.Contains(cmd.Args, "--retry-lock") {
		t.Errorf("command() with RetryLock off = %v, want no --retry-lock", cmd.Args)
	}
}

func TestRemapOwners(t *testing.T) {
	root := t.TempDir()
	for _, dir := range []string{"world/region", "caddy/certs"} {
		if err := os.MkdirAll(filepath.Join(root, dir), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	for _, file := range []string{"world/level.dat", "world/region/r.0.0.mca", "caddy/certs/cert.pem"} {
		if err := os.WriteFile(filepath.Join(root, file), nil, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Symlink("level.dat", filepath.Join(root, "world/link")); err != nil {
		t.Fatal(err)
	}

	type call struct {
		path     string
		uid, gid int
	}
	var calls []call
	orig := lchown
	lchown = func(path string, uid, gid int) error {
		calls = append(calls, call{path, uid, gid})
		return nil
	}
	t.Cleanup(func() { lchown = orig })

	// Every temp file is owned by the test's own ids, so treat those as legacy
	ownUID, ownGID := uint32(os.Getuid()), uint32(os.Getgid())
	from := []uint32{ownUID}
	if ownGID != ownUID {
		from = append(from, ownGID)
	}
	if err := RemapOwners(root, filepath.Join(root, "caddy"), from, 25565, 25566); err != nil {
		t.Fatal(err)
	}

	var got []string
	for _, c := range calls {
		if c.uid != 25565 || c.gid != 25566 {
			t.Errorf("lchown(%s, %d, %d), want 25565, 25566", c.path, c.uid, c.gid)
		}
		rel, _ := filepath.Rel(root, c.path)
		got = append(got, rel)
	}
	want := []string{".", "world", "world/level.dat", "world/link", "world/region", "world/region/r.0.0.mca"}
	if !slices.Equal(got, want) {
		t.Errorf("re-owned %v, want %v (caddy/ skipped)", got, want)
	}

	calls = nil
	if err := RemapOwners(root, "", []uint32{ownUID + 1}, 25565, 25566); err != nil {
		t.Fatal(err)
	}
	if ownGID == ownUID+1 {
		t.Skip("own gid is uid+1, can't test non-matching ids")
	}
	if len(calls) != 0 {
		t.Errorf("re-owned %v with no legacy owners, want none", calls)
	}
}

func TestRestoredWorldsDir(t *testing.T) {
	c := &Config{WorldsDir: "/srv/minecraft-server"}
	for target, want := range map[string]string{
		"/":                "/srv/minecraft-server",
		"/tmp/restore":     "/tmp/restore/srv/minecraft-server",
		"/tmp/restore/":    "/tmp/restore/srv/minecraft-server",
		"relative/restore": "relative/restore/srv/minecraft-server",
	} {
		if got := c.restoredWorldsDir(target); got != want {
			t.Errorf("restoredWorldsDir(%q) = %q, want %q", target, got, want)
		}
	}
}
