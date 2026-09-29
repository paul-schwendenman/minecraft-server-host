package backup

import (
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
