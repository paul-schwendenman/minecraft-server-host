package doctor

import (
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"strconv"
	"syscall"
	"time"

	"github.com/paul/minecraftctl/pkg/backup"
	"github.com/paul/minecraftctl/pkg/maps"
	"github.com/paul/minecraftctl/pkg/rcon"
	"github.com/paul/minecraftctl/pkg/systemd"
	"github.com/paul/minecraftctl/pkg/worlds"
)

// ResticTimeout is how long the restic repo check waits
const ResticTimeout = 30 * time.Second

// SystemEnv returns an Env that checks this machine, with config settings
// taken from the caller
func SystemEnv(worldsDir string, configFile string, configErr error) Env {
	_, err := exec.LookPath("systemctl")
	return Env{
		ConfigErr:          configErr,
		ConfigFile:         configFile,
		WorldsDir:          worldsDir,
		OnServer:           err == nil,
		Owner:              fileOwner,
		LookupMinecraftUID: lookupMinecraftUID,
		UnitFileState:      systemd.UnitFileState,
		ActiveState:        systemd.GetActiveState,
		RunningWorlds:      worlds.RunningWorlds,
		ResticInstalled:    backup.IsResticInstalled,
		ResticEnv: func() error {
			_, err := backup.LoadConfig()
			return err
		},
		ResticPing: func() error {
			cfg, err := backup.LoadConfig()
			if err != nil {
				return err
			}
			return cfg.Ping(ResticTimeout)
		},
		RconPing: func() error {
			client, err := rcon.NewClient()
			if err != nil {
				return err
			}
			defer client.Close()
			_, err = client.Send("list")
			return err
		},
		UnminedPath: maps.DefaultUnminedPath,
	}
}

func fileOwner(path string) (int, error) {
	info, err := os.Stat(path)
	if err != nil {
		return 0, err
	}
	st, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return 0, fmt.Errorf("no owner information")
	}
	return int(st.Uid), nil
}

func lookupMinecraftUID() (int, error) {
	u, err := user.Lookup(backup.RunAsUser)
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(u.Uid)
}
