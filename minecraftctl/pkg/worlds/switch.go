package worlds

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/paul/minecraftctl/pkg/config"
	"github.com/paul/minecraftctl/pkg/lock"
	"github.com/paul/minecraftctl/pkg/properties"
	"github.com/paul/minecraftctl/pkg/rcon"
	"github.com/paul/minecraftctl/pkg/systemd"
	"github.com/rs/zerolog/log"
)

// SwitchLockPath is held for the whole of a switch. autoshutdown.sh skips its
// idle check while it's held, since mid-switch no world may be running. It
// lives on tmpfs so a crash can't leave it behind across a reboot, and flock
// releases it if the process dies.
const SwitchLockPath = "/run/minecraft-switch.lock"

// Switch outcomes, for callers (the phase 3 tag watcher) that need to tell
// them apart. Each maps to its own exit code in the CLI.
var (
	// ErrNotStartable: the target world can't start. Nothing was changed.
	ErrNotStartable = errors.New("world can't start")
	// ErrSwitchRefused: players are online, the player count couldn't be
	// read, or another switch is in progress. Nothing was changed; retry later.
	ErrSwitchRefused = errors.New("switch refused")
	// ErrStartFailed: the target world didn't come up. It was stopped and the
	// previous world (if any) is running again.
	ErrStartFailed = errors.New("world didn't start")
	// ErrRollbackFailed: the target world didn't come up and the previous
	// world didn't come back either. Nothing is running.
	ErrRollbackFailed = errors.New("world didn't start and the previous world didn't come back")
)

var worldNameRe = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)

// CheckStartable reports why a world can't start, or nil if it can. It doesn't
// require world/level.dat: Minecraft writes that on a world's first start, so
// a world fresh from `world create` has none yet.
//
//   - eula.txt: without eula=true Minecraft exits straight away.
//   - enable-rcon=true: Minecraft would start without it, but autoshutdown
//     reads the player count over RCON (and would power off with people
//     playing), and ExecStop saves and stops the world over RCON.
//   - server.jar must resolve, which also catches a jar missing from
//     /opt/minecraft/jars.
func CheckStartable(worldName string) error {
	return checkStartable(config.Get().WorldsDir, worldName)
}

func checkStartable(worldsDir, worldName string) error {
	if !worldNameRe.MatchString(worldName) {
		return fmt.Errorf("%w: %q is not a valid world name", ErrNotStartable, worldName)
	}
	dir := filepath.Join(worldsDir, worldName)
	if info, err := os.Stat(dir); err != nil || !info.IsDir() {
		return fmt.Errorf("%w: world not found: %s", ErrNotStartable, dir)
	}

	eula, err := properties.Load(filepath.Join(dir, "eula.txt"))
	if err != nil || !propertyIs(eula, "eula", "true") {
		return fmt.Errorf("%w: %s doesn't accept the EULA", ErrNotStartable, filepath.Join(dir, "eula.txt"))
	}

	props, err := properties.Load(filepath.Join(dir, "server.properties"))
	if err != nil || !propertyIs(props, "enable-rcon", "true") {
		return fmt.Errorf("%w: RCON not enabled in %s", ErrNotStartable, filepath.Join(dir, "server.properties"))
	}

	// Stat follows the symlink
	if info, err := os.Stat(filepath.Join(dir, "server.jar")); err != nil || !info.Mode().IsRegular() {
		return fmt.Errorf("%w: %s is missing or points to a missing jar", ErrNotStartable, filepath.Join(dir, "server.jar"))
	}
	return nil
}

func propertyIs(p *properties.Properties, key, want string) bool {
	v, ok := p.Get(key)
	return ok && strings.TrimSpace(v) == want
}

// RunningWorlds returns the worlds whose minecraft@ service isn't stopped:
// running, starting, stopping or waiting to restart after a crash. Normally
// there's at most one.
func RunningWorlds() ([]string, error) {
	units, err := listWorldUnits()
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(units))
	for _, u := range units {
		names = append(names, u.world)
	}
	return names, nil
}

type worldUnit struct {
	world  string
	active string // systemd ActiveState
}

func listWorldUnits() ([]worldUnit, error) {
	units, err := systemd.ListUnits("minecraft@*.service", "active", "activating", "deactivating", "reloading")
	if err != nil {
		return nil, err
	}
	var out []worldUnit
	for _, u := range units {
		name := strings.TrimSuffix(strings.TrimPrefix(u.Name, "minecraft@"), ".service")
		out = append(out, worldUnit{world: name, active: u.Active})
	}
	return out, nil
}

// SwitchOptions controls SwitchWorld.
type SwitchOptions struct {
	// Force switches even with players online, after warning them.
	Force bool
	// WarnDelay is how long players get between the warning and the stop.
	WarnDelay time.Duration
	// Timeout is how long to wait for the new world to answer RCON.
	Timeout time.Duration
	// DryRun runs the checks and reports what would happen, without changing
	// anything.
	DryRun bool
}

// SwitchResult describes what SwitchWorld did (or, with DryRun, would do).
type SwitchResult struct {
	World   string
	Stopped []string // worlds stopped (or that would be)
	Started bool     // false if World was already running
}

// SwitchWorld stops whatever world is running and starts worldName, waiting
// until it answers RCON. If it doesn't come up within opts.Timeout, it's
// stopped and the previous world is started again. If worldName is already
// running, any other running world is stopped and nothing else happens.
//
// The returned error wraps one of ErrNotStartable, ErrSwitchRefused,
// ErrStartFailed or ErrRollbackFailed when it's one of those outcomes.
func SwitchWorld(worldName string, opts SwitchOptions) (*SwitchResult, error) {
	return defaultSwitcher().switchWorld(worldName, opts)
}

// switcher holds the side effects of a switch, so tests can replace them.
type switcher struct {
	worldsDir     string
	lock          func() (unlock func(), err error)
	units         func() ([]worldUnit, error)
	start         func(world string) error
	stop          func(world string) error
	playersOnline func() (int, error)
	say           func(msg string) error
	waitUp        func(world string, timeout time.Duration) error
	sleep         func(time.Duration)
}

func defaultSwitcher() *switcher {
	return &switcher{
		worldsDir: config.Get().WorldsDir,
		lock:      lockSwitch,
		units:     listWorldUnits,
		start: func(world string) error {
			return systemd.Start(systemd.FormatUnitName("minecraft", world, systemd.UnitService))
		},
		stop: func(world string) error {
			return systemd.Stop(systemd.FormatUnitName("minecraft", world, systemd.UnitService))
		},
		playersOnline: rconPlayersOnline,
		say: func(msg string) error {
			_, err := rconSend("say " + msg)
			return err
		},
		waitUp: waitForRcon,
		sleep:  time.Sleep,
	}
}

func (s *switcher) switchWorld(target string, opts SwitchOptions) (*SwitchResult, error) {
	if err := checkStartable(s.worldsDir, target); err != nil {
		return nil, err
	}

	if !opts.DryRun {
		unlock, err := s.lock()
		if err != nil {
			return nil, err
		}
		defer unlock()
	}

	units, err := s.units()
	if err != nil {
		return nil, fmt.Errorf("failed to list running worlds: %w", err)
	}

	result := &SwitchResult{World: target, Started: true}
	var previous string // the world to go back to if target doesn't come up
	serving := false    // a world is up and could have players on it
	for _, u := range units {
		if u.world == target && u.active == "active" {
			result.Started = false
			continue
		}
		result.Stopped = append(result.Stopped, u.world)
		if u.active == "active" || u.active == "reloading" {
			serving = true
			if previous == "" {
				previous = u.world
			}
		}
	}
	if len(result.Stopped) > 1 {
		log.Warn().Strs("worlds", result.Stopped).Msg("more than one world is running; stopping all of them")
	}

	// Only a world that's up can have players. One that's crash-looping or
	// already stopping can't, and isn't worth going back to.
	if serving {
		if err := s.checkPlayers(target, opts); err != nil {
			return nil, err
		}
	}

	if opts.DryRun {
		return result, nil
	}

	for _, w := range result.Stopped {
		log.Info().Str("world", w).Msg("stopping")
		if err := s.stop(w); err != nil {
			return nil, fmt.Errorf("failed to stop %s: %w", w, err)
		}
	}

	if !result.Started {
		return result, nil
	}

	log.Info().Str("world", target).Msg("starting")
	startErr := s.start(target)
	if startErr == nil {
		startErr = s.waitUp(target, opts.Timeout)
	}
	if startErr == nil {
		return result, nil
	}

	// Roll back: stop the target so it doesn't sit in a restart loop, and
	// bring back what was running before.
	log.Error().Err(startErr).Str("world", target).Msg("world didn't come up, rolling back")
	if err := s.stop(target); err != nil {
		log.Error().Err(err).Str("world", target).Msg("failed to stop world")
	}
	if previous == "" {
		return nil, fmt.Errorf("%w: %s: %v", ErrStartFailed, target, startErr)
	}
	log.Info().Str("world", previous).Msg("starting previous world again")
	if err := s.start(previous); err == nil {
		err = s.waitUp(previous, opts.Timeout)
		if err == nil {
			return nil, fmt.Errorf("%w: %s: %v (%s is running again)", ErrStartFailed, target, startErr, previous)
		}
		log.Error().Err(err).Str("world", previous).Msg("previous world didn't come back")
	} else {
		log.Error().Err(err).Str("world", previous).Msg("failed to start previous world")
	}
	if err := s.stop(previous); err != nil {
		log.Error().Err(err).Str("world", previous).Msg("failed to stop world")
	}
	return nil, fmt.Errorf("%w: %s: %v", ErrRollbackFailed, target, startErr)
}

// checkPlayers refuses the switch if anyone's online, unless opts.Force, in
// which case it warns them and waits opts.WarnDelay first.
func (s *switcher) checkPlayers(target string, opts SwitchOptions) error {
	count, err := s.playersOnline()
	if err != nil {
		if !opts.Force {
			return fmt.Errorf("%w: can't read the player count: %v", ErrSwitchRefused, err)
		}
		log.Warn().Err(err).Msg("can't read the player count, switching anyway (--force)")
		return nil
	}
	if count == 0 {
		return nil
	}
	if !opts.Force {
		return fmt.Errorf("%w: %d player(s) online (use --force to switch anyway)", ErrSwitchRefused, count)
	}
	if opts.DryRun {
		return nil
	}
	log.Warn().Int("players", count).Msg("players online, switching anyway (--force)")
	msg := fmt.Sprintf("Switching to world %s in %s. You'll be disconnected.", target, opts.WarnDelay)
	if err := s.say(msg); err != nil {
		log.Warn().Err(err).Msg("failed to warn players")
	}
	s.sleep(opts.WarnDelay)
	return nil
}

func lockSwitch() (func(), error) {
	fl := lock.NewFileLock(SwitchLockPath)
	// A short wait rides out autoshutdown's momentary check of the lock.
	if err := fl.TryLock(5 * time.Second); err != nil {
		if errors.Is(err, fs.ErrPermission) {
			return nil, fmt.Errorf("failed to lock %s: %w", SwitchLockPath, systemd.ErrPermission)
		}
		if strings.Contains(err.Error(), "timeout") {
			return nil, fmt.Errorf("%w: another switch is in progress", ErrSwitchRefused)
		}
		return nil, err
	}
	return func() { _ = fl.Unlock() }, nil
}

func rconSend(command string) (string, error) {
	client, err := rcon.NewClient()
	if err != nil {
		return "", err
	}
	defer client.Close()
	return client.Send(command)
}

var playerCountRe = regexp.MustCompile(`There are (\d+)`)

func rconPlayersOnline() (int, error) {
	out, err := rconSend("list")
	if err != nil {
		return 0, err
	}
	return parsePlayerCount(out)
}

// parsePlayerCount reads the count from the reply to `list`, e.g. "There are
// 2 of a max of 20 players online: a, b".
func parsePlayerCount(out string) (int, error) {
	m := playerCountRe.FindStringSubmatch(out)
	if m == nil {
		return 0, fmt.Errorf("unexpected reply to list: %q", out)
	}
	return strconv.Atoi(m[1])
}

// waitForRcon waits for a world to answer RCON, which Minecraft only starts
// once the world has loaded. It gives up early if the unit stops or fails.
func waitForRcon(world string, timeout time.Duration) error {
	unit := systemd.FormatUnitName("minecraft", world, systemd.UnitService)
	deadline := time.Now().Add(timeout)
	for {
		if _, err := rconSend("list"); err == nil {
			log.Info().Str("world", world).Msg("world is up")
			return nil
		}
		switch state := systemd.GetActiveState(unit); state {
		case "inactive", "failed":
			return fmt.Errorf("%s is %s", unit, state)
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("%s didn't answer RCON within %s", world, timeout)
		}
		time.Sleep(5 * time.Second)
	}
}
