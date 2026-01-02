// Package backup provides functions for managing world backups using restic.
package backup

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/paul/minecraftctl/pkg/envfile"
)

const (
	defaultRegion    = "us-east-2"
	defaultWorldsDir = "/srv/minecraft-server"
)

// Config holds the backup configuration
type Config struct {
	Repository string
	Password   string
	WorldsDir  string
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

	return &Config{
		Repository: fmt.Sprintf("s3:s3.%s.amazonaws.com/%s", region, bucket),
		Password:   password,
		WorldsDir:  worldsDir,
	}, nil
}

// runRestic executes a restic command with the configured environment
func (c *Config) runRestic(args ...string) error {
	cmd := exec.Command("restic", args...)
	cmd.Env = append(os.Environ(),
		"RESTIC_REPOSITORY="+c.Repository,
		"RESTIC_PASSWORD="+c.Password,
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

// runResticOutput executes a restic command and returns the output
func (c *Config) runResticOutput(args ...string) (string, error) {
	cmd := exec.Command("restic", args...)
	cmd.Env = append(os.Environ(),
		"RESTIC_REPOSITORY="+c.Repository,
		"RESTIC_PASSWORD="+c.Password,
	)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// InitRepository initializes the restic repository if it doesn't exist
func (c *Config) InitRepository() error {
	// Check if repo exists by trying to list snapshots
	_, err := c.runResticOutput("snapshots", "--quiet")
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

	err := c.runRestic("backup", backupPath,
		"--tag", tag,
		"--exclude", "*.log",
		"--exclude", "logs/",
		"--exclude", "crash-reports/",
	)
	if err != nil {
		return err
	}

	fmt.Println("\nBackup complete. Recent snapshots:")
	return c.runRestic("snapshots", "--latest", "3", "--tag", tag)
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

	return c.runRestic(args...)
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
