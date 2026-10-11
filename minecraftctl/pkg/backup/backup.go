// Package backup provides functions for managing world backups using restic.
package backup

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/paul/minecraftctl/pkg/envfile"
	"github.com/rs/zerolog/log"
)

const (
	defaultRegion    = "us-east-2"
	defaultWorldsDir = "/srv/minecraft-server"

	// Host is recorded as the host of every snapshot, instead of the machine's
	// hostname. Every new instance gets a new hostname, and restic picks the
	// parent snapshot (so it can skip unchanged files) and groups snapshots
	// for `forget` by host. Each bucket belongs to one environment, so one
	// name is enough.
	Host = "minecraft"

	// CacheDir is restic's cache when it's owned by the user restic runs as.
	// restic's default is ~/.cache, but minecraft@.service runs the on-stop
	// backup with ProtectHome, which hides it. Other users get restic's
	// default, so none of them can leave files there its owner can't update.
	CacheDir = "/var/cache/restic"

	// RunAsUser is who restic runs as when minecraftctl runs as root, so
	// every run shares CacheDir, and restored files belong to the user that
	// runs the worlds instead of the numeric owner recorded in the snapshot.
	RunAsUser = "minecraft"

	// DefaultRetryLock is how long restic waits for a lock another restic
	// process holds (the weekly prune's exclusive lock, or a backup's
	// shared one while prune starts) before giving up. systemd only orders
	// a prune after backups that are queued with it, so a backup timer that
	// fires mid-prune would otherwise fail straight away.
	DefaultRetryLock = "30m"

	// RetryLockEnv overrides DefaultRetryLock; "0" turns retrying off
	RetryLockEnv = "MC_RESTIC_RETRY_LOCK"
)

// LegacyIDs are the uids/gids RunAsUser had on older AMIs. Snapshots taken
// then record them, and they now belong to other accounts (dhcpcd, crontab),
// so Restore hands files carrying them back to RunAsUser.
var LegacyIDs = []uint32{996, 997}

// Config holds the backup configuration
type Config struct {
	Repository string
	Password   string
	WorldsDir  string
	// RunAs is who restic runs as, or nil for the current user
	RunAs *Credential
	// RetryLock is passed to restic's --retry-lock, or "" to leave it off
	RetryLock string
}

// LoadConfig loads backup configuration from environment
func LoadConfig() (*Config, error) {
	// Try to load from env file if not already in environment
	if os.Getenv("MC_WORLD_BUCKET") == "" || os.Getenv("RESTIC_PASSWORD") == "" {
		if ef, err := envfile.Load(envfile.DefaultMinecraftEnvPath); err == nil {
			ef.ExportIfNotSet()
		}
	}

	bucket := os.Getenv("MC_WORLD_BUCKET")
	if bucket == "" {
		return nil, fmt.Errorf("MC_WORLD_BUCKET not set")
	}

	password := os.Getenv("RESTIC_PASSWORD")
	if password == "" {
		return nil, fmt.Errorf("RESTIC_PASSWORD not set")
	}

	region := os.Getenv("AWS_REGION")
	if region == "" {
		region = defaultRegion
	}

	worldsDir := os.Getenv("MINECRAFT_WORLDS_DIR")
	if worldsDir == "" {
		worldsDir = defaultWorldsDir
	}

	retryLock, err := retryLockFromEnv()
	if err != nil {
		return nil, err
	}

	cfg := &Config{
		Repository: fmt.Sprintf("s3:s3.%s.amazonaws.com/%s", region, bucket),
		Password:   password,
		WorldsDir:  worldsDir,
		RetryLock:  retryLock,
	}
	if os.Geteuid() == 0 {
		cred, err := LookupCredential(RunAsUser)
		if err != nil {
			log.Warn().Err(err).Msgf("can't run restic as %s, running it as root", RunAsUser)
		} else {
			cfg.RunAs = cred
		}
	}
	return cfg, nil
}

// retryLockFromEnv returns the --retry-lock value from RetryLockEnv
// (default DefaultRetryLock), or "" when it's a zero duration
func retryLockFromEnv() (string, error) {
	value := strings.TrimSpace(os.Getenv(RetryLockEnv))
	if value == "" {
		return DefaultRetryLock, nil
	}
	d, err := time.ParseDuration(value)
	if err != nil || d < 0 {
		return "", fmt.Errorf("%s=%q is not a duration like 30m or 0", RetryLockEnv, value)
	}
	if d == 0 {
		return "", nil
	}
	return value, nil
}

// command builds a restic command, run as runAs (nil: the current user)
func (c *Config) command(runAs *Credential, args ...string) *exec.Cmd {
	if c.RetryLock != "" {
		// A global flag, so it goes before the subcommand and applies to
		// every one (backup, forget --prune, check, restore, snapshots...)
		args = append([]string{"--retry-lock", c.RetryLock}, args...)
	}
	cmd := exec.Command("restic", args...)
	env := append(os.Environ(),
		"RESTIC_REPOSITORY="+c.Repository,
		"RESTIC_PASSWORD="+c.Password,
	)
	uid := os.Getuid()
	if runAs != nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{Credential: &syscall.Credential{
			Uid:    runAs.Uid,
			Gid:    runAs.Gid,
			Groups: runAs.Groups,
		}}
		uid = int(runAs.Uid)
		// sudo leaves HOME as root's, which restic can't use as runAs
		env = append(env, "HOME="+runAs.Home)
	}
	if os.Getenv("RESTIC_CACHE_DIR") == "" && ownedBy(CacheDir, uid) {
		env = append(env, "RESTIC_CACHE_DIR="+CacheDir)
	}
	cmd.Env = env
	return cmd
}

// ownedBy reports whether path is a directory owned by uid
func ownedBy(path string, uid int) bool {
	info, err := os.Stat(path)
	if err != nil || !info.IsDir() {
		return false
	}
	st, ok := info.Sys().(*syscall.Stat_t)
	return ok && int(st.Uid) == uid
}

func run(cmd *exec.Cmd) error {
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

// runRestic executes a restic command as c.RunAs
func (c *Config) runRestic(args ...string) error {
	return run(c.command(c.RunAs, args...))
}

// runResticOutput executes a restic command as c.RunAs and returns the output
func (c *Config) runResticOutput(args ...string) (string, error) {
	out, err := c.command(c.RunAs, args...).CombinedOutput()
	return string(out), err
}

// InitRepository initializes the restic repository if it doesn't exist
func (c *Config) InitRepository() error {
	// Check if repo exists by reading its config, which (unlike listing
	// snapshots) doesn't grow with the repository
	_, err := c.runResticOutput("cat", "config")
	if err == nil {
		return nil // Repo already exists
	}

	fmt.Println("Initializing restic repository...")
	return c.runRestic("init")
}

// List shows available snapshots, optionally filtered by tag
func (c *Config) List(tag string) error {
	args := []string{"snapshots"}
	if tag != "" {
		args = append(args, "--tag", tag)
	}
	return c.runRestic(args...)
}

// Create creates a new backup
func (c *Config) Create(world string) error {
	if err := c.InitRepository(); err != nil {
		return fmt.Errorf("failed to initialize repository: %w", err)
	}

	var backupPath string
	var tag string

	if world == "" || world == "all" {
		backupPath = c.WorldsDir
		tag = "all"
		fmt.Printf("Backing up all worlds in %s...\n", backupPath)
	} else {
		backupPath = fmt.Sprintf("%s/%s/world", c.WorldsDir, world)
		tag = world
		if _, err := os.Stat(backupPath); os.IsNotExist(err) {
			return fmt.Errorf("world path not found: %s", backupPath)
		}
		fmt.Printf("Backing up world: %s...\n", world)
	}

	args := backupArgs(backupPath, tag)
	if tag == "all" {
		// Caddy's TLS certificates: re-issued if lost, and only readable by
		// the caddy user, so including them makes every `all` backup fail
		// (restic exits 3 on unreadable files)
		args = append(args, "--exclude", filepath.Join(c.WorldsDir, "caddy"))
	}
	if err := c.runRestic(args...); err != nil {
		return err
	}

	fmt.Println("Backup complete.")
	return nil
}

func backupArgs(path, tag string) []string {
	return []string{"backup", path,
		"--host", Host,
		"--tag", tag,
		"--exclude", "*.log",
		"--exclude", "logs/",
		"--exclude", "crash-reports/",
	}
}

// Restore restores a snapshot
func (c *Config) Restore(snapshot string, target string) error {
	if snapshot == "" {
		snapshot = "latest"
	}

	args := []string{"restore", snapshot, "--target", target}

	if target == "/" {
		fmt.Printf("Restoring snapshot %s to original location...\n", snapshot)
	} else {
		fmt.Printf("Restoring snapshot %s to %s...\n", snapshot, target)
		if err := os.MkdirAll(target, 0755); err != nil {
			return fmt.Errorf("failed to create target directory: %w", err)
		}
	}

	// Run as the caller, not c.RunAs: restic also restores the metadata of
	// every directory above the world (/, /srv), which fails for anyone but
	// root. Restored files keep the owners recorded in the snapshot, apart
	// from LegacyIDs, which are handed back to RunAsUser below.
	if err := run(c.command(nil, args...)); err != nil {
		return err
	}

	if os.Geteuid() != 0 {
		return nil
	}
	owner, err := LookupCredential(RunAsUser)
	if err != nil {
		log.Warn().Err(err).Msgf("can't re-own files from old %s ids", RunAsUser)
		return nil
	}
	root := target
	if target == "/" {
		root = c.WorldsDir
	}
	// caddy/ isn't backed up, and its files belong to caddy, which an older
	// AMI gave one of the legacy ids
	skip := filepath.Join(root, "caddy")
	return RemapOwners(root, skip, LegacyIDs, owner.Uid, owner.Gid)
}

// lchown is os.Lchown, swapped out by tests
var lchown = os.Lchown

// RemapOwners gives every file under root (except the skip directory) whose
// uid or gid is one of from the uid or gid to, leaving other owners alone.
// Symlinks are re-owned themselves, not followed.
func RemapOwners(root, skip string, from []uint32, uid, gid uint32) error {
	legacy := make(map[uint32]bool, len(from))
	for _, id := range from {
		legacy[id] = true
	}
	return filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && path == skip {
			return filepath.SkipDir
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		st, ok := info.Sys().(*syscall.Stat_t)
		if !ok {
			return nil
		}
		newUID, newGID := -1, -1
		if legacy[st.Uid] {
			newUID = int(uid)
		}
		if legacy[st.Gid] {
			newGID = int(gid)
		}
		if newUID == -1 && newGID == -1 {
			return nil
		}
		if err := lchown(path, newUID, newGID); err != nil {
			return fmt.Errorf("re-own %s: %w", path, err)
		}
		return nil
	})
}

// Prune removes old snapshots according to retention policy
func (c *Config) Prune() error {
	fmt.Println("Pruning old snapshots...")
	err := c.runRestic("forget",
		"--keep-daily", "7",
		"--keep-weekly", "4",
		"--keep-monthly", "3",
		"--prune",
	)
	if err != nil {
		return err
	}

	fmt.Println("\nChecking repository integrity...")
	if err := c.runRestic("check"); err != nil {
		return err
	}

	fmt.Println("\nRepository statistics:")
	return c.runRestic("stats")
}

// Stats shows repository statistics
func (c *Config) Stats() error {
	return c.runRestic("stats")
}

// Check verifies repository integrity
func (c *Config) Check() error {
	return c.runRestic("check")
}

// IsResticInstalled checks if restic is available
func IsResticInstalled() bool {
	_, err := exec.LookPath("restic")
	return err == nil
}

// GetResticVersion returns the installed restic version
func GetResticVersion() (string, error) {
	cmd := exec.Command("restic", "version")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}
