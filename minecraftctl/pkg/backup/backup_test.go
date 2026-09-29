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
