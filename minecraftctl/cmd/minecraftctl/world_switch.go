package main

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/paul/minecraftctl/pkg/worlds"
	"github.com/spf13/cobra"
)

// Exit codes for `world switch`, so a caller like the tag watcher can tell
// the outcomes apart without parsing output. 1 is any other error.
const (
	exitNotStartable  = 2 // the world can't start; nothing changed
	exitSwitchRefused = 3 // players online or another switch running; retry later
	exitStartFailed   = 4 // the world didn't come up; the previous world is back
	exitRollbackFail  = 5 // the world didn't come up and neither did the previous one
)

// exitCode maps an error returned by a command to the process exit code.
func exitCode(err error) int {
	switch {
	case errors.Is(err, worlds.ErrNotStartable):
		return exitNotStartable
	case errors.Is(err, worlds.ErrSwitchRefused):
		return exitSwitchRefused
	case errors.Is(err, worlds.ErrRollbackFailed):
		return exitRollbackFail
	case errors.Is(err, worlds.ErrStartFailed):
		return exitStartFailed
	default:
		return 1
	}
}

var (
	switchForce     bool
	switchWarnDelay time.Duration
	switchTimeout   time.Duration
	switchDryRun    bool
)

var worldSwitchCmd = &cobra.Command{
	Use:   "switch <world>",
	Short: "Stop the running world and start another",
	Long: `Stop whatever world is running and start <world>, waiting until it answers
RCON. Only one world can run at a time: they all use the same ports.

Refuses if players are online, unless --force is given, in which case players
are warned and get --warn-delay before the world stops. If <world> doesn't come
up within --timeout, it's stopped and the previous world is started again. If
<world> is the only world running, nothing happens. If nothing is running (at boot),
<world> is just started.

This doesn't change which world starts at boot; on the AWS hosts that's the
ActiveWorld instance tag.

Exit codes:
  0  switched, or <world> was already running
  1  any other error
  2  <world> can't start (not a world, no EULA, RCON off, missing jar); nothing changed
  3  refused: players online, or another switch is running; nothing changed
  4  <world> didn't come up; the previous world is running again
  5  <world> didn't come up and neither did the previous world; nothing is running`,
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: worldCompletionFunc,
	RunE: func(cmd *cobra.Command, args []string) error {
		res, err := worlds.SwitchWorld(args[0], worlds.SwitchOptions{
			Force:     switchForce,
			WarnDelay: switchWarnDelay,
			Timeout:   switchTimeout,
			DryRun:    switchDryRun,
		})
		if err != nil {
			return err
		}

		stopped := "nothing"
		if len(res.Stopped) > 0 {
			stopped = strings.Join(res.Stopped, ", ")
		}
		switch {
		case !res.Started:
			fmt.Printf("%s is already running; nothing to do\n", res.World)
		case switchDryRun:
			fmt.Printf("Would stop %s and start %s\n", stopped, res.World)
		default:
			fmt.Printf("Switched to %s (stopped %s)\n", res.World, stopped)
		}
		return nil
	},
}

// refuseIfOtherWorldRunning stops `world start` and `world restart` from
// starting a second world alongside a running one; they'd fight over the
// same ports.
func refuseIfOtherWorldRunning(world string) error {
	running, err := worlds.RunningWorlds()
	if err != nil {
		return err
	}
	for _, w := range running {
		if w != world {
			return fmt.Errorf("world %s is running; use `minecraftctl world switch %s` to switch to %s", w, world, world)
		}
	}
	return nil
}

func init() {
	worldSwitchCmd.Flags().BoolVar(&switchForce, "force", false, "Switch even with players online (they're warned first)")
	worldSwitchCmd.Flags().DurationVar(&switchWarnDelay, "warn-delay", 10*time.Second, "With --force, how long players get between the warning and the stop")
	worldSwitchCmd.Flags().DurationVar(&switchTimeout, "timeout", 5*time.Minute, "How long to wait for the world to answer RCON before rolling back")
	worldSwitchCmd.Flags().BoolVar(&switchDryRun, "dry-run", false, "Run the checks and print what would happen, without changing anything")

	WorldCmd.AddCommand(worldSwitchCmd)
}
