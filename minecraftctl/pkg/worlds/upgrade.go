package worlds

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/paul/minecraftctl/pkg/config"
	"github.com/paul/minecraftctl/pkg/systemd"
	"github.com/rs/zerolog/log"
)

// UpgradeOptions holds options for upgrading a world
type UpgradeOptions struct {
	TargetVersion string
	StopService   bool
}

// UpgradeResult contains the result of an upgrade operation
type UpgradeResult struct {
	WorldName       string
	PreviousVersion string
	NewVersion      string
	ServiceStopped  bool
}

// IsServiceRunning checks if the minecraft service for a world is running
func IsServiceRunning(worldName string) (bool, error) {
	serviceName := systemd.FormatUnitName("minecraft", worldName, systemd.UnitService)
	return systemd.IsActive(serviceName)
}

// StopService stops the minecraft service for a world
func StopService(worldName string) error {
	serviceName := systemd.FormatUnitName("minecraft", worldName, systemd.UnitService)
	return systemd.Stop(serviceName)
}

// GetCurrentVersion reads the current version from the server.jar symlink
func GetCurrentVersion(worldPath string) (string, error) {
	serverJarPath := filepath.Join(worldPath, "server.jar")

	target, err := os.Readlink(serverJarPath)
	if err != nil {
		return "", fmt.Errorf("failed to read server.jar symlink: %w", err)
	}

	// Extract version from path like /opt/minecraft/jars/minecraft_server_1.21.1.jar
	base := filepath.Base(target)
	if !strings.HasPrefix(base, "minecraft_server_") || !strings.HasSuffix(base, ".jar") {
		return "", fmt.Errorf("unexpected jar filename format: %s", base)
	}
	version := strings.TrimPrefix(base, "minecraft_server_")
	version = strings.TrimSuffix(version, ".jar")

	return version, nil
}

// UpgradeWorld upgrades a world to a new Minecraft version
func UpgradeWorld(worldName string, opts UpgradeOptions) (*UpgradeResult, error) {
	cfg := config.Get()
	worldPath := filepath.Join(cfg.WorldsDir, worldName)

	// 1. Verify world exists
	if _, err := GetWorldInfo(worldName); err != nil {
		return nil, fmt.Errorf("world not found: %w", err)
	}

	// 2. Get current version from symlink
	currentVersion, err := GetCurrentVersion(worldPath)
	if err != nil {
		return nil, fmt.Errorf("failed to determine current version: %w", err)
	}

	// 3. Verify target version is newer
	cmp, err := CompareVersions(currentVersion, opts.TargetVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to compare versions: %w", err)
	}
	if cmp >= 0 {
		return nil, fmt.Errorf("target version %s is not newer than current version %s (downgrades not supported)",
			opts.TargetVersion, currentVersion)
	}

	// 4. Verify target JAR exists
	targetJarPath := filepath.Join(cfg.JarsDir,
		fmt.Sprintf("minecraft_server_%s.jar", opts.TargetVersion))
	if _, err := os.Stat(targetJarPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("target JAR not found: %s (use 'minecraftctl jar download' first)",
			targetJarPath)
	}

	// 5. Check if service is running
	running, err := IsServiceRunning(worldName)
	if err != nil {
		log.Warn().Err(err).Msg("failed to check service status, assuming not running")
		running = false
	}

	result := &UpgradeResult{
		WorldName:       worldName,
		PreviousVersion: currentVersion,
		NewVersion:      opts.TargetVersion,
		ServiceStopped:  false,
	}

	if running {
		if !opts.StopService {
			return nil, fmt.Errorf("server minecraft@%s.service is running; use --stop to automatically stop it first, or stop it manually", worldName)
		}

		log.Info().Str("world", worldName).Msg("stopping minecraft service")
		if err := StopService(worldName); err != nil {
			return nil, fmt.Errorf("failed to stop service: %w", err)
		}
		result.ServiceStopped = true
	}

	// 6. Update server.jar symlink
	serverJarPath := filepath.Join(worldPath, "server.jar")

	if err := os.Remove(serverJarPath); err != nil {
		return nil, fmt.Errorf("failed to remove existing server.jar symlink: %w", err)
	}

	if err := os.Symlink(targetJarPath, serverJarPath); err != nil {
		return nil, fmt.Errorf("failed to create server.jar symlink: %w", err)
	}

	log.Info().
		Str("world", worldName).
		Str("from", currentVersion).
		Str("to", opts.TargetVersion).
		Msg("world upgraded successfully")

	return result, nil
}
