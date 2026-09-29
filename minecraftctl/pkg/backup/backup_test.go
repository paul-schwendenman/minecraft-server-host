package backup

import (
	"os"
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
