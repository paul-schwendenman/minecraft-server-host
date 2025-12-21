package worlds

import (
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/paul/minecraftctl/pkg/config"
	"github.com/paul/minecraftctl/pkg/envfile"
	"github.com/paul/minecraftctl/pkg/nbt"
	"github.com/rs/zerolog/log"
)

// WorldInfo contains information about a Minecraft world
type WorldInfo struct {
	Name         string
	Path         string
	Version      string
	SpawnX       int32
	SpawnY       int32
	SpawnZ       int32
	LastPlayed   time.Time
	Difficulty   string
	GameType     string
	LevelName    string
	HasMapConfig bool
}

// ListWorlds returns a list of all worlds in the configured worlds directory
func ListWorlds() ([]WorldInfo, error) {
	cfg := config.Get()
	worldsDir := cfg.WorldsDir

	entries, err := os.ReadDir(worldsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read worlds directory: %w", err)
	}

	var worlds []WorldInfo
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		worldPath := filepath.Join(worldsDir, entry.Name())
		levelDatPath := filepath.Join(worldPath, "world", "level.dat")

		// Check if this looks like a world directory
		if _, err := os.Stat(levelDatPath); os.IsNotExist(err) {
			continue
		}

		info, err := GetWorldInfo(entry.Name())
		if err != nil {
			log.Warn().Err(err).Str("world", entry.Name()).Msg("failed to get world info")
			continue
		}

		worlds = append(worlds, *info)
	}

	// Sort by name
	sort.Slice(worlds, func(i, j int) bool {
		return worlds[i].Name < worlds[j].Name
	})

	return worlds, nil
}

// GetWorldInfo retrieves information about a specific world
func GetWorldInfo(worldName string) (*WorldInfo, error) {
	cfg := config.Get()
	worldPath := filepath.Join(cfg.WorldsDir, worldName)
	levelDatPath := filepath.Join(worldPath, "world", "level.dat")

	if _, err := os.Stat(levelDatPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("world not found: %s (no level.dat at %s)", worldName, levelDatPath)
	}

	levelInfo, err := nbt.ReadLevelDat(levelDatPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read level.dat: %w", err)
	}

	// Check for map-config.yml
	mapConfigPath := filepath.Join(worldPath, "map-config.yml")
	hasMapConfig := true
	if _, err := os.Stat(mapConfigPath); os.IsNotExist(err) {
		hasMapConfig = false
	}

	difficulty := difficultyName(levelInfo.Difficulty)
	gameType := gameTypeName(levelInfo.GameType)

	return &WorldInfo{
		Name:         worldName,
		Path:         worldPath,
		Version:      levelInfo.GetVersionName(),
		SpawnX:       levelInfo.SpawnX,
		SpawnY:       levelInfo.SpawnY,
		SpawnZ:       levelInfo.SpawnZ,
		LastPlayed:   time.Unix(levelInfo.LastPlayed/1000, 0),
		Difficulty:   difficulty,
		GameType:     gameType,
		LevelName:    levelInfo.LevelName,
		HasMapConfig: hasMapConfig,
	}, nil
}

func difficultyName(diff int32) string {
	switch diff {
	case 0:
		return "Peaceful"
	case 1:
		return "Easy"
	case 2:
		return "Normal"
	case 3:
		return "Hard"
	default:
		return "Unknown"
	}
}

func gameTypeName(gt int32) string {
	switch gt {
	case 0:
		return "Survival"
	case 1:
		return "Creative"
	case 2:
		return "Adventure"
	case 3:
		return "Spectator"
	default:
		return "Unknown"
	}
}

// ExpandWorldPattern expands a world name pattern (with glob characters) to a list of matching world names
func ExpandWorldPattern(pattern string) ([]string, error) {
	cfg := config.Get()
	worldsDir := cfg.WorldsDir

	// Check if pattern contains glob characters
	hasGlob := strings.ContainsAny(pattern, "*?[]")
	if !hasGlob {
		// No glob characters, return pattern as-is if it's a valid world
		worldPath := filepath.Join(worldsDir, pattern)
		levelDatPath := filepath.Join(worldPath, "world", "level.dat")
		if _, err := os.Stat(levelDatPath); os.IsNotExist(err) {
			return nil, fmt.Errorf("world not found: %s", pattern)
		}
		return []string{pattern}, nil
	}

	// Build a glob pattern that matches world directories
	globPattern := filepath.Join(worldsDir, pattern, "world", "level.dat")

	// Find all matching paths
	matches, err := filepath.Glob(globPattern)
	if err != nil {
		return nil, fmt.Errorf("invalid glob pattern: %w", err)
	}

	if len(matches) == 0 {
		return nil, fmt.Errorf("no worlds match pattern: %s", pattern)
	}

	// Extract world names from matched paths
	worldNames := make([]string, 0, len(matches))
	for _, match := range matches {
		// match is: /path/to/worlds/<worldname>/world/level.dat
		rel, err := filepath.Rel(worldsDir, match)
		if err != nil {
			continue
		}
		worldName := strings.Split(rel, string(filepath.Separator))[0]
		worldNames = append(worldNames, worldName)
	}

	// Remove duplicates and sort
	seen := make(map[string]bool)
	result := make([]string, 0, len(worldNames))
	for _, name := range worldNames {
		if !seen[name] {
			seen[name] = true
			result = append(result, name)
		}
	}
	sort.Strings(result)

	return result, nil
}

// CreateWorldOptions holds options for creating a new world
type CreateWorldOptions struct {
	Version         string
	Seed            string
	CreateMapConfig bool
	EnableSystemd   bool
}

// CreateWorld creates a new Minecraft world directory with the necessary configuration files
func CreateWorld(worldName string, opts CreateWorldOptions) error {
	cfg := config.Get()
	worldDir := filepath.Join(cfg.WorldsDir, worldName)
	jarPath := fmt.Sprintf("/opt/minecraft/jars/minecraft_server_%s.jar", opts.Version)

	// Check if world already exists
	if _, err := os.Stat(worldDir); err == nil {
		// Check if it's actually a world (has level.dat)
		levelDatPath := filepath.Join(worldDir, "world", "level.dat")
		if _, err := os.Stat(levelDatPath); err == nil {
			return fmt.Errorf("world already exists: %s", worldName)
		}
	}

	// Validate that jar exists
	if _, err := os.Stat(jarPath); os.IsNotExist(err) {
		return fmt.Errorf("Minecraft server jar not found: %s", jarPath)
	}

	// Create world directory
	if err := os.MkdirAll(worldDir, 0755); err != nil {
		return fmt.Errorf("failed to create world directory: %w", err)
	}

	// Link the server jar
	serverJarPath := filepath.Join(worldDir, "server.jar")
	// Remove existing symlink if it exists
	if _, err := os.Lstat(serverJarPath); err == nil {
		if err := os.Remove(serverJarPath); err != nil {
			return fmt.Errorf("failed to remove existing server.jar: %w", err)
		}
	}
	if err := os.Symlink(jarPath, serverJarPath); err != nil {
		return fmt.Errorf("failed to create symlink to server jar: %w", err)
	}

	// Create eula.txt
	eulaPath := filepath.Join(worldDir, "eula.txt")
	if err := os.WriteFile(eulaPath, []byte("eula=true\n"), 0644); err != nil {
		return fmt.Errorf("failed to create eula.txt: %w", err)
	}

	// Load RCON settings from /etc/minecraft.env
	rconPort := cfg.Rcon.Port
	rconPassword := cfg.Rcon.Password
	if ef, err := envfile.Load(envfile.DefaultMinecraftEnvPath); err == nil {
		if port, err := ef.GetInt("RCON_PORT"); err == nil {
			rconPort = port
		}
		if pwd, ok := ef.Get("RCON_PASSWORD"); ok {
			rconPassword = pwd
		}
	} else {
		log.Warn().Msg("/etc/minecraft.env not found, using config values for RCON")
	}

	// If RCON password is still empty, try to get it from config
	if rconPassword == "" {
		rconPassword = cfg.Rcon.Password
	}

	// Create server.properties
	serverPropsPath := filepath.Join(worldDir, "server.properties")
	if _, err := os.Stat(serverPropsPath); os.IsNotExist(err) {
		var props strings.Builder
		props.WriteString("enable-rcon=true\n")
		props.WriteString(fmt.Sprintf("rcon.port=%d\n", rconPort))
		props.WriteString(fmt.Sprintf("rcon.password=%s\n", rconPassword))
		props.WriteString(fmt.Sprintf("motd=Welcome to %s\n", worldName))
		props.WriteString("level-name=world\n")
		if opts.Seed != "" {
			props.WriteString(fmt.Sprintf("level-seed=%s\n", opts.Seed))
		}

		if err := os.WriteFile(serverPropsPath, []byte(props.String()), 0644); err != nil {
			return fmt.Errorf("failed to create server.properties: %w", err)
		}
	}

	// Create map-config.yml if requested
	if opts.CreateMapConfig {
		mapConfigPath := filepath.Join(worldDir, "map-config.yml")
		if _, err := os.Stat(mapConfigPath); os.IsNotExist(err) {
			mapConfig := `# Default map configuration for uNmINeD
# Adjust zoom levels, dimensions, and regions as needed

defaults:
  zoomout: 2
  zoomin: 1
  imageformat: jpeg
  chunkprocessors: 4

maps:
  - name: overworld
    dimension: overworld
    output_subdir: overworld
    options:
      shadows: 3d
    ranges:
      - name: spawn_area
        center: [0, 0]
        radius: 2048
        zoomout: 2
        zoomin: 4

  - name: nether
    dimension: nether
    output_subdir: nether
    options:
      topY: 68
      shadows: 2d
      night: false
    ranges:
      - name: hub
        center: [0, 0]
        radius: 1024
        zoomout: 3
        zoomin: 2

  - name: end
    dimension: end
    output_subdir: end
    options:
      shadows: 2d
`

			if err := os.WriteFile(mapConfigPath, []byte(mapConfig), 0644); err != nil {
				return fmt.Errorf("failed to create map-config.yml: %w", err)
			}
		}
	}

	// Fix permissions: chown all created files to minecraft:minecraft
	// This ensures the systemd service (which runs as minecraft user) can write to these files
	if err := chownToMinecraftUser(worldDir); err != nil {
		log.Warn().Err(err).Str("world", worldName).Msg("failed to chown world directory to minecraft user, continuing")
	}

	// Enable and start systemd service if requested
	if opts.EnableSystemd {
		serviceName := fmt.Sprintf("minecraft@%s.service", worldName)

		// Enable service
		enableCmd := exec.Command("systemctl", "enable", serviceName)
		if err := enableCmd.Run(); err != nil {
			return fmt.Errorf("failed to enable systemd service %s: %w", serviceName, err)
		}

		// Start service
		startCmd := exec.Command("systemctl", "start", serviceName)
		if err := startCmd.Run(); err != nil {
			return fmt.Errorf("failed to start systemd service %s: %w", serviceName, err)
		}
	}

	return nil
}

// chownToMinecraftUser recursively changes ownership of a directory to minecraft:minecraft
func chownToMinecraftUser(path string) error {
	minecraftUser, err := user.Lookup("minecraft")
	if err != nil {
		return fmt.Errorf("failed to lookup minecraft user: %w", err)
	}

	minecraftGroup, err := user.LookupGroup("minecraft")
	if err != nil {
		return fmt.Errorf("failed to lookup minecraft group: %w", err)
	}

	uid, err := strconv.Atoi(minecraftUser.Uid)
	if err != nil {
		return fmt.Errorf("failed to parse minecraft user UID: %w", err)
	}

	gid, err := strconv.Atoi(minecraftGroup.Gid)
	if err != nil {
		return fmt.Errorf("failed to parse minecraft group GID: %w", err)
	}

	// Recursively chown the directory and all its contents
	return filepath.Walk(path, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		return os.Chown(filePath, uid, gid)
	})
}

// RegisterWorld registers an existing world with systemd services and timers
// without modifying any world files. This is used to "reattach" a world from
// an EBS volume to a new server instance.
func RegisterWorld(worldName string) error {
	cfg := config.Get()
	worldDir := filepath.Join(cfg.WorldsDir, worldName)
	levelDatPath := filepath.Join(worldDir, "world", "level.dat")

	// Validate that the world exists
	if _, err := os.Stat(levelDatPath); os.IsNotExist(err) {
		return fmt.Errorf("world not found: %s (no level.dat at %s)", worldName, levelDatPath)
	}

	// Reload systemd daemon first to ensure all service files are recognized
	reloadCmd := exec.Command("systemctl", "daemon-reload")
	if err := reloadCmd.Run(); err != nil {
		return fmt.Errorf("failed to reload systemd daemon: %w", err)
	}

	// Enable and start the main service
	serviceName := fmt.Sprintf("minecraft@%s.service", worldName)
	enableCmd := exec.Command("systemctl", "enable", serviceName)
	if err := enableCmd.Run(); err != nil {
		return fmt.Errorf("failed to enable systemd service %s: %w", serviceName, err)
	}

	startCmd := exec.Command("systemctl", "start", serviceName)
	if err := startCmd.Run(); err != nil {
		return fmt.Errorf("failed to start systemd service %s: %w", serviceName, err)
	}

	// Enable timers (but don't start them - they'll start on their schedule)
	timers := []string{
		fmt.Sprintf("minecraft-map-rebuild@%s.timer", worldName),
		fmt.Sprintf("minecraft-world-backup@%s.timer", worldName),
		fmt.Sprintf("minecraft-map-backup@%s.timer", worldName),
	}

	for _, timerName := range timers {
		enableTimerCmd := exec.Command("systemctl", "enable", timerName)
		if err := enableTimerCmd.Run(); err != nil {
			// Log warning but don't fail - timers might not be installed
			log.Warn().Err(err).Str("timer", timerName).Msg("failed to enable timer, continuing")
		}
	}

	return nil
}
