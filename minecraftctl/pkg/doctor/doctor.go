// Package doctor runs read-only health checks on a Minecraft server host.
//
// Every check goes through an Env, so tests can stand in for systemd, file
// ownership, restic and RCON.
package doctor

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/paul/minecraftctl/pkg/worlds"
)

// Status is the outcome of one check
type Status string

const (
	Pass Status = "PASS"
	Warn Status = "WARN"
	Fail Status = "FAIL"
	// Skip means the check didn't apply (no world running, not a server)
	Skip Status = "SKIP"
)

// Result is one check's outcome
type Result struct {
	Status Status `json:"status"`
	Name   string `json:"name"`
	Detail string `json:"detail"`
}

// MinecraftUID is the uid the minecraft user has on current AMIs. Older AMIs
// used 996/997 (see backup.LegacyIDs).
const MinecraftUID = 25565

// GlobalUnits are the units the AMI enables that aren't per world
var GlobalUnits = []string{
	"autoshutdown.timer",
	"minecraft-active.service",
	"minecraft-world-watch.timer",
	"minecraft-world-backup.timer",
	"minecraft-world-prune.timer",
}

// notWorlds are directories in the worlds dir that aren't worlds
var notWorlds = map[string]bool{"maps": true, "caddy": true, "lost+found": true}

// Env is everything the checks look at
type Env struct {
	// ConfigErr is the error loading the config, if any
	ConfigErr error
	// ConfigFile is the config file used ("" for defaults only)
	ConfigFile string
	WorldsDir  string
	// OnServer is false where there's no systemd (e.g. a Mac). Checks that
	// need the server then warn or skip instead of failing.
	OnServer bool

	// Owner returns the uid that owns path
	Owner func(path string) (int, error)
	// LookupMinecraftUID returns the minecraft user's uid
	LookupMinecraftUID func() (int, error)
	// UnitFileState is `systemctl is-enabled` (enabled, disabled, not-found...)
	UnitFileState func(unit string) (string, error)
	// ActiveState is systemd's ActiveState (active, inactive, failed...)
	ActiveState func(unit string) string
	// RunningWorlds lists the worlds whose minecraft@ service is up
	RunningWorlds func() ([]string, error)
	// ResticInstalled reports whether restic is on PATH
	ResticInstalled func() bool
	// ResticEnv loads the restic settings (MC_WORLD_BUCKET, RESTIC_PASSWORD)
	ResticEnv func() error
	// ResticPing opens the repository, with a timeout
	ResticPing func() error
	// RconPing connects to RCON and runs a command
	RconPing func() error
	// UnminedPath is the uNmINeD CLI binary
	UnminedPath string
}

type checker struct {
	env     Env
	results []Result
}

func (c *checker) add(status Status, name, format string, args ...any) {
	c.results = append(c.results, Result{Status: status, Name: name, Detail: fmt.Sprintf(format, args...)})
}

// failOrWarn is Fail on a server and Warn elsewhere
func (c *checker) failOrWarn() Status {
	if c.env.OnServer {
		return Fail
	}
	return Warn
}

// Run runs every check, in order
func Run(env Env) []Result {
	c := &checker{env: env}
	c.checkConfig()
	worldNames := c.checkWorldsDir()
	running := c.checkUnits(worldNames)
	c.checkRcon(running)
	c.checkRestic()
	c.checkUnmined()
	return c.results
}

func (c *checker) checkConfig() {
	switch {
	case c.env.ConfigErr != nil:
		c.add(Fail, "config", "%v", c.env.ConfigErr)
	case c.env.ConfigFile != "":
		c.add(Pass, "config", "loaded %s", c.env.ConfigFile)
	default:
		c.add(Pass, "config", "no config file, using defaults")
	}
}

// checkWorldsDir checks the worlds dir and each world in it, returning the
// world names
func (c *checker) checkWorldsDir() []string {
	dir := c.env.WorldsDir
	info, err := os.Stat(dir)
	if err != nil {
		c.add(c.failOrWarn(), "worlds dir", "%v", err)
		return nil
	}
	if !info.IsDir() {
		c.add(Fail, "worlds dir", "%s is not a directory", dir)
		return nil
	}

	wantUID, lookupErr := c.env.LookupMinecraftUID()
	switch {
	case lookupErr != nil:
		c.add(c.failOrWarn(), "minecraft user", "%v", lookupErr)
		wantUID = MinecraftUID
	case wantUID != MinecraftUID:
		c.add(Warn, "minecraft user", "uid %d, expected %d (an older AMI?)", wantUID, MinecraftUID)
	default:
		c.add(Pass, "minecraft user", "uid %d", wantUID)
	}

	owner, err := c.env.Owner(dir)
	switch {
	case err != nil:
		c.add(Warn, "worlds dir", "%s: can't read owner: %v", dir, err)
	case owner != wantUID:
		status := Fail
		if lookupErr != nil {
			status = Warn
		}
		c.add(status, "worlds dir", "%s is owned by uid %d, want minecraft (uid %d)", dir, owner, wantUID)
	default:
		c.add(Pass, "worlds dir", "%s, owned by minecraft", dir)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		c.add(Fail, "worlds", "%v", err)
		return nil
	}
	var names []string
	for _, e := range entries {
		name := e.Name()
		if !e.IsDir() || notWorlds[name] || strings.HasPrefix(name, ".") {
			continue
		}
		worldDir := filepath.Join(dir, name)
		if exists(filepath.Join(worldDir, "world", "level.dat")) {
			names = append(names, name)
			continue
		}
		// Only directories that look like a server dir are broken worlds
		if exists(filepath.Join(worldDir, "server.properties")) || exists(filepath.Join(worldDir, "server.jar")) {
			c.add(Fail, "world "+name, "no world/level.dat in %s", worldDir)
		}
	}
	sort.Strings(names)
	if len(names) == 0 {
		c.add(Warn, "worlds", "no worlds (no */world/level.dat) in %s", dir)
	} else {
		c.add(Pass, "worlds", "%d with level.dat: %s", len(names), strings.Join(names, ", "))
	}
	return names
}

func exists(path string) bool {
	_, err := os.Lstat(path)
	return err == nil
}

// checkUnits checks the AMI's units and each world's timers, returning the
// running world ("" if none or unknown)
func (c *checker) checkUnits(worldNames []string) string {
	if !c.env.OnServer {
		c.add(Skip, "units", "no systemctl (not a server?)")
		c.add(Skip, "running world", "no systemctl (not a server?)")
		return ""
	}

	var problems []string
	for _, unit := range GlobalUnits {
		if p := c.enabledProblem(unit); p != "" {
			problems = append(problems, p)
		}
	}
	if len(problems) > 0 {
		c.add(Fail, "units", "%s", strings.Join(problems, "; "))
	} else {
		c.add(Pass, "units", "%s enabled", strings.Join(GlobalUnits, ", "))
	}

	running := ""
	worldsUp, err := c.env.RunningWorlds()
	switch {
	case err != nil:
		c.add(Warn, "running world", "%v", err)
	case len(worldsUp) == 0:
		c.add(Pass, "running world", "none")
	case len(worldsUp) > 1:
		c.add(Warn, "running world", "more than one: %s", strings.Join(worldsUp, ", "))
	default:
		running = worldsUp[0]
		c.add(Pass, "running world", "%s", running)
	}

	for _, world := range worldNames {
		c.checkWorldTimers(world, world == running)
	}
	return running
}

// enabledProblem describes what's wrong with unit's enablement, or ""
func (c *checker) enabledProblem(unit string) string {
	state, err := c.env.UnitFileState(unit)
	switch {
	case err != nil:
		return fmt.Sprintf("%s: %v", unit, err)
	case state == "not-found":
		return unit + " not installed"
	case state != "enabled":
		return fmt.Sprintf("%s is %s", unit, state)
	}
	return ""
}

// checkWorldTimers checks a world's timers are enabled and, where they
// should be, active. The backup timers run whether or not the world does;
// the map-build timer only runs with its world.
func (c *checker) checkWorldTimers(world string, isRunning bool) {
	name := "timers " + world
	var fails, warns []string
	for _, timer := range worlds.WorldTimers(world) {
		if p := c.enabledProblem(timer); p != "" {
			fails = append(fails, p)
			continue
		}
		mapBuild := strings.HasPrefix(timer, "minecraft-map-build@")
		if mapBuild && !isRunning {
			continue
		}
		if state := c.env.ActiveState(timer); state != "active" {
			warns = append(warns, fmt.Sprintf("%s is %s", timer, state))
		}
	}
	switch {
	case len(fails) > 0:
		c.add(Fail, name, "%s (run: minecraftctl world register %s)", strings.Join(fails, "; "), world)
	case len(warns) > 0:
		c.add(Warn, name, "%s", strings.Join(warns, "; "))
	default:
		c.add(Pass, name, "enabled")
	}
}

func (c *checker) checkRcon(running string) {
	if running == "" {
		c.add(Skip, "rcon", "no world running")
		return
	}
	if err := c.env.RconPing(); err != nil {
		c.add(Fail, "rcon", "%s: %v", running, err)
		return
	}
	c.add(Pass, "rcon", "%s answers", running)
}

func (c *checker) checkRestic() {
	if !c.env.ResticInstalled() {
		c.add(c.failOrWarn(), "restic", "restic not found on PATH")
		return
	}
	if err := c.env.ResticEnv(); err != nil {
		c.add(c.failOrWarn(), "restic env", "%v", err)
		c.add(Skip, "restic repo", "no restic settings")
		return
	}
	c.add(Pass, "restic env", "MC_WORLD_BUCKET and RESTIC_PASSWORD set")
	if err := c.env.ResticPing(); err != nil {
		c.add(Fail, "restic repo", "%v", err)
		return
	}
	c.add(Pass, "restic repo", "reachable")
}

func (c *checker) checkUnmined() {
	info, err := os.Stat(c.env.UnminedPath)
	switch {
	case err != nil:
		c.add(c.failOrWarn(), "unmined", "%v", err)
	case info.IsDir() || info.Mode()&0111 == 0:
		c.add(Fail, "unmined", "%s is not executable", c.env.UnminedPath)
	default:
		c.add(Pass, "unmined", "%s", c.env.UnminedPath)
	}
}

// Failed reports whether results should make doctor exit non-zero: any
// FAIL, or with strict any WARN too
func Failed(results []Result, strict bool) bool {
	for _, r := range results {
		if r.Status == Fail || (strict && r.Status == Warn) {
			return true
		}
	}
	return false
}

// Counts returns how many results have each status
func Counts(results []Result) map[Status]int {
	counts := map[Status]int{}
	for _, r := range results {
		counts[r.Status]++
	}
	return counts
}
