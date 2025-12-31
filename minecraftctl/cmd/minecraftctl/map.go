package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"sync"

	"github.com/AlecAivazis/survey/v2"
	"github.com/paul/minecraftctl/internal/commands"
	"github.com/paul/minecraftctl/pkg/config"
	"github.com/paul/minecraftctl/pkg/maps"
	"github.com/paul/minecraftctl/pkg/systemd"
	"github.com/paul/minecraftctl/pkg/worlds"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// Logs command flags for map services
var (
	mapLogsFollow  bool
	mapLogsLines   int
	mapLogsSince   string
	mapLogsOutput  string
	mapLogsNoPager bool
)

// mapWorldCompletionFunc provides tab completion for world names in map commands
func mapWorldCompletionFunc(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	// For commands that accept multiple worlds, always complete world names
	names, err := worlds.GetWorldNames()
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}
	return names, cobra.ShellCompDirectiveNoFileComp
}

// mapSingleWorldCompletionFunc provides tab completion for first world arg only
func mapSingleWorldCompletionFunc(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) != 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	names, err := worlds.GetWorldNames()
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}
	return names, cobra.ShellCompDirectiveNoFileComp
}

// MapCmd is an alias for the command defined in internal/commands
var MapCmd = commands.MapCmd

var mapBuildCmd = &cobra.Command{
	Use:               "build <world> [worlds...]",
	Short:             "Build maps for one or more worlds",
	Args:              cobra.MinimumNArgs(1),
	ValidArgsFunction: mapWorldCompletionFunc,
	RunE: func(cmd *cobra.Command, args []string) error {
		mapName, _ := cmd.Flags().GetString("map")
		force, _ := cmd.Flags().GetBool("force")
		lockFile, _ := cmd.Flags().GetString("lock-file")
		lockTimeout, _ := cmd.Flags().GetDuration("lock-timeout")
		noLock, _ := cmd.Flags().GetBool("no-lock")
		nonBlocking, _ := cmd.Flags().GetBool("non-blocking")
		parallel, _ := cmd.Flags().GetBool("parallel")
		maxWorkers, _ := cmd.Flags().GetInt("max-workers")
		logLevel, _ := cmd.Flags().GetString("log-level")

		// Expand all patterns to world names
		worldNames := make([]string, 0)
		for _, pattern := range args {
			expanded, err := worlds.ExpandWorldPattern(pattern)
			if err != nil {
				return fmt.Errorf("failed to expand pattern %s: %w", pattern, err)
			}
			worldNames = append(worldNames, expanded...)
		}

		if len(worldNames) == 0 {
			return fmt.Errorf("no worlds found matching patterns")
		}

		if len(worldNames) == 1 && !parallel {
			// Single world, process directly
			builder := maps.NewBuilder()
			opts := maps.BuildOptions{
				WorldName:   worldNames[0],
				MapName:     mapName,
				Force:       force,
				LockFile:    lockFile,
				LockTimeout: lockTimeout,
				NoLock:      noLock,
				NonBlocking: nonBlocking,
				LogLevel:    logLevel,
			}
			return builder.Build(opts)
		}

		// Multiple worlds - batch processing
		return buildBatch(worldNames, maps.BuildOptions{
			MapName:     mapName,
			Force:       force,
			LockFile:    lockFile,
			LockTimeout: lockTimeout,
			NoLock:      noLock,
			NonBlocking: nonBlocking,
			LogLevel:    logLevel,
		}, parallel, maxWorkers)
	},
}

var mapPreviewCmd = &cobra.Command{
	Use:               "preview <world> <map>",
	Short:             "Generate preview image for a map",
	Args:              cobra.ExactArgs(2),
	ValidArgsFunction: mapSingleWorldCompletionFunc,
	RunE: func(cmd *cobra.Command, args []string) error {
		worldName := args[0]
		mapName := args[1]
		logLevel, _ := cmd.Flags().GetString("log-level")

		builder := maps.NewBuilder()
		return builder.GeneratePreview(worldName, mapName, logLevel)
	},
}

var mapManifestCmd = &cobra.Command{
	Use:               "manifest <world> [worlds...]",
	Short:             "Build manifests for all maps in one or more worlds",
	Args:              cobra.MinimumNArgs(1),
	ValidArgsFunction: mapWorldCompletionFunc,
	RunE: func(cmd *cobra.Command, args []string) error {
		noPreview, _ := cmd.Flags().GetBool("no-preview")
		previewOnly, _ := cmd.Flags().GetBool("preview-only")
		parallel, _ := cmd.Flags().GetBool("parallel")
		maxWorkers, _ := cmd.Flags().GetInt("max-workers")
		logLevel, _ := cmd.Flags().GetString("log-level")

		// Expand all patterns to world names
		worldNames := make([]string, 0)
		for _, pattern := range args {
			expanded, err := worlds.ExpandWorldPattern(pattern)
			if err != nil {
				return fmt.Errorf("failed to expand pattern %s: %w", pattern, err)
			}
			worldNames = append(worldNames, expanded...)
		}

		if len(worldNames) == 0 {
			return fmt.Errorf("no worlds found matching patterns")
		}

		updateIndex, _ := cmd.Flags().GetBool("update-index")
		builder := maps.NewManifestBuilder()

		var err error
		if len(worldNames) == 1 && !parallel {
			// Single world, process directly
			opts := maps.ManifestOptions{
				WorldName:        worldNames[0],
				GeneratePreviews: !noPreview,
				PreviewOnly:      previewOnly,
				LogLevel:         logLevel,
			}
			err = builder.BuildManifests(worldNames[0], opts)
		} else {
			// Multiple worlds - batch processing
			err = manifestBatch(worldNames, maps.ManifestOptions{
				GeneratePreviews: !noPreview,
				PreviewOnly:      previewOnly,
				LogLevel:         logLevel,
			}, parallel, maxWorkers)
		}

		if err != nil {
			return err
		}

		// Update aggregate index if requested
		if updateIndex {
			return builder.BuildAggregateIndex()
		}

		return nil
	},
}

var mapIndexCmd = &cobra.Command{
	Use:   "index",
	Short: "Generate aggregate manifest and HTML index page",
	Long:  "Generates world_manifest.json and index.html in the maps directory",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		builder := maps.NewManifestBuilder()
		return builder.BuildAggregateIndex()
	},
}

var mapConfigCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage map configuration files",
	Long:  "Commands for managing map-config.yml files for worlds",
}

var mapConfigGenerateCmd = &cobra.Command{
	Use:               "generate <world>",
	Short:             "Generate a basic map-config.yml file for a world",
	Long:              "Creates a map-config.yml file with default settings and a spawn area zoom region based on NBT spawn coordinates",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: mapSingleWorldCompletionFunc,
	RunE: func(cmd *cobra.Command, args []string) error {
		worldName := args[0]
		force, _ := cmd.Flags().GetBool("force")
		radius, _ := cmd.Flags().GetInt("radius")
		output, _ := cmd.Flags().GetString("output")

		// Get world info to retrieve spawn coordinates
		worldInfo, err := worlds.GetWorldInfo(worldName)
		if err != nil {
			return fmt.Errorf("failed to get world info: %w", err)
		}

		// Determine output path
		worldPath := worldInfo.Path
		configPath := filepath.Join(worldPath, "map-config.yml")
		if output != "" {
			configPath = output
		}

		// Check if config already exists
		if _, err := os.Stat(configPath); err == nil && !force {
			return fmt.Errorf("map-config.yml already exists at %s (use --force to overwrite)", configPath)
		}

		// Create map config with defaults
		zoomoutDefault := 6
		zoominDefault := 0
		zoominSpawn := 2
		zoomoutSpawn := 0
		topYNether := 68
		shadows3d := "3d"
		shadows2d := "2d"
		nightFalse := false

		mapConfig := &config.MapConfig{
			Defaults: config.MapDefaults{
				Zoomout:         zoomoutDefault,
				Zoomin:          zoominDefault,
				ImageFormat:     "jpeg",
				ChunkProcessors: 4,
			},
			Maps: []config.MapDefinition{
				{
					Name:         "overworld",
					Dimension:    "overworld",
					OutputSubdir: "overworld",
					Options: config.MapOptions{
						Shadows: shadows3d,
					},
					Ranges: []config.MapRange{
						{
							Name:    "spawn_area",
							Center:  [2]int{int(worldInfo.SpawnX), int(worldInfo.SpawnZ)},
							Radius:  radius,
							Zoomout: &zoomoutSpawn,
							Zoomin:  &zoominSpawn,
						},
					},
				},
				{
					Name:         "nether",
					Dimension:    "nether",
					OutputSubdir: "nether",
					Options: config.MapOptions{
						TopY:    &topYNether,
						Shadows: shadows2d,
						Night:   &nightFalse,
					},
				},
				{
					Name:         "end",
					Dimension:    "end",
					OutputSubdir: "end",
					Options: config.MapOptions{
						Shadows: shadows2d,
						Night:   &nightFalse,
					},
				},
			},
		}

		// Save the config
		// SaveMapConfig can accept either a directory path or full file path
		if err := config.SaveMapConfig(configPath, mapConfig); err != nil {
			return fmt.Errorf("failed to save map-config.yml: %w", err)
		}

		log.Info().
			Str("world", worldName).
			Str("path", configPath).
			Int32("spawnX", worldInfo.SpawnX).
			Int32("spawnZ", worldInfo.SpawnZ).
			Int("radius", radius).
			Msg("map-config.yml created")

		return nil
	},
}

var mapConfigGetCmd = &cobra.Command{
	Use:               "get <world> [path]",
	Short:             "Get config value(s) from map-config.yml",
	Long:              "Display config values. If path is specified, shows that field. Otherwise shows full config.",
	Args:              cobra.RangeArgs(1, 2),
	ValidArgsFunction: mapSingleWorldCompletionFunc,
	RunE: func(cmd *cobra.Command, args []string) error {
		worldName := args[0]
		var path string
		if len(args) > 1 {
			path = args[1]
		}

		format, _ := cmd.Flags().GetString("format")

		// Get world info
		worldInfo, err := worlds.GetWorldInfo(worldName)
		if err != nil {
			return fmt.Errorf("failed to get world info: %w", err)
		}

		// Load config
		mapConfig, err := config.LoadMapConfig(worldInfo.Path)
		if err != nil {
			return fmt.Errorf("failed to load map-config.yml: %w", err)
		}

		// Get field value
		value, err := config.GetConfigField(path, mapConfig)
		if err != nil {
			return fmt.Errorf("failed to get field: %w", err)
		}

		// Output based on format
		switch format {
		case "json":
			// TODO: implement JSON output
			// For now, just output YAML
			fallthrough
		case "yaml", "":
			data, err := yaml.Marshal(value)
			if err != nil {
				return fmt.Errorf("failed to marshal value: %w", err)
			}
			fmt.Print(string(data))
		default:
			return fmt.Errorf("unsupported format: %s (supported: yaml, json)", format)
		}

		return nil
	},
}

var mapConfigSetCmd = &cobra.Command{
	Use:               "set <world> <path> <value>",
	Short:             "Set a config value in map-config.yml",
	Long:              "Set a specific field value using dot notation path (e.g., defaults.zoomout)",
	Args:              cobra.ExactArgs(3),
	ValidArgsFunction: mapSingleWorldCompletionFunc,
	RunE: func(cmd *cobra.Command, args []string) error {
		worldName := args[0]
		path := args[1]
		valueStr := args[2]

		// Get world info
		worldInfo, err := worlds.GetWorldInfo(worldName)
		if err != nil {
			return fmt.Errorf("failed to get world info: %w", err)
		}

		// Load config
		mapConfig, err := config.LoadMapConfig(worldInfo.Path)
		if err != nil {
			return fmt.Errorf("failed to load map-config.yml: %w", err)
		}

		// Parse value (try int, bool, then string)
		var value interface{}
		if intVal, err := strconv.Atoi(valueStr); err == nil {
			value = intVal
		} else if boolVal, err := strconv.ParseBool(valueStr); err == nil {
			value = boolVal
		} else {
			value = valueStr
		}

		// Set value
		if err := config.SetConfigField(path, value, mapConfig); err != nil {
			return fmt.Errorf("failed to set field: %w", err)
		}

		// Save config
		if err := config.SaveMapConfig(worldInfo.Path, mapConfig); err != nil {
			return fmt.Errorf("failed to save map-config.yml: %w", err)
		}

		log.Info().Str("world", worldName).Str("path", path).Msg("config value updated")
		return nil
	},
}

var mapConfigValidateCmd = &cobra.Command{
	Use:               "validate <world>",
	Short:             "Validate map-config.yml structure and values",
	Long:              "Checks that the map-config.yml file is valid and all values are within acceptable ranges",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: mapSingleWorldCompletionFunc,
	RunE: func(cmd *cobra.Command, args []string) error {
		worldName := args[0]

		// Get world info
		worldInfo, err := worlds.GetWorldInfo(worldName)
		if err != nil {
			return fmt.Errorf("failed to get world info: %w", err)
		}

		// Load config
		mapConfig, err := config.LoadMapConfig(worldInfo.Path)
		if err != nil {
			return fmt.Errorf("failed to load map-config.yml: %w", err)
		}

		// Validate config
		errs := config.ValidateMapConfig(mapConfig)
		if len(errs) > 0 {
			fmt.Fprintf(os.Stderr, "Validation failed with %d error(s):\n", len(errs))
			for _, e := range errs {
				fmt.Fprintf(os.Stderr, "  - %s\n", e)
			}
			return fmt.Errorf("validation failed")
		}

		fmt.Println("✓ map-config.yml is valid")
		return nil
	},
}

var mapConfigEditCmd = &cobra.Command{
	Use:               "edit <world>",
	Short:             "Interactively edit map-config.yml",
	Long:              "Opens an interactive menu-driven editor for map-config.yml",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: mapSingleWorldCompletionFunc,
	RunE: func(cmd *cobra.Command, args []string) error {
		worldName := args[0]

		// Get world info
		worldInfo, err := worlds.GetWorldInfo(worldName)
		if err != nil {
			return fmt.Errorf("failed to get world info: %w", err)
		}

		// Load or create config
		var mapConfig *config.MapConfig
		mapConfig, err = config.LoadMapConfig(worldInfo.Path)
		if err != nil {
			// Config doesn't exist, create a basic one
			fmt.Printf("No map-config.yml found. Creating a basic configuration...\n")
			mapConfig = &config.MapConfig{
				Defaults: config.MapDefaults{
					Zoomout:         6,
					Zoomin:          0,
					ImageFormat:     "jpeg",
					ChunkProcessors: 4,
				},
				Maps: []config.MapDefinition{
					{
						Name:         "overworld",
						Dimension:    "overworld",
						OutputSubdir: "overworld",
					},
				},
			}
		}

		// Run interactive editor
		if err := runInteractiveEditor(worldInfo.Path, mapConfig); err != nil {
			return err
		}

		return nil
	},
}

// Map backup service commands
var mapBackupCmd = &cobra.Command{
	Use:   "backup",
	Short: "Manage map backup service",
}

var mapBackupStatusCmd = &cobra.Command{
	Use:               "status <world>",
	Short:             "Show status of the map backup service and timer",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: mapSingleWorldCompletionFunc,
	RunE: func(cmd *cobra.Command, args []string) error {
		service := systemd.FormatUnitName("minecraft-map-backup", args[0], systemd.UnitService)
		timer := systemd.FormatUnitName("minecraft-map-backup", args[0], systemd.UnitTimer)
		fmt.Println("=== Service ===")
		if err := systemd.Status(service); err != nil {
			// Continue to show timer status even if service status fails
		}
		fmt.Println("\n=== Timer ===")
		return systemd.Status(timer)
	},
}

var mapBackupStartCmd = &cobra.Command{
	Use:               "start <world>",
	Short:             "Trigger a map backup now",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: mapSingleWorldCompletionFunc,
	RunE: func(cmd *cobra.Command, args []string) error {
		unit := systemd.FormatUnitName("minecraft-map-backup", args[0], systemd.UnitService)
		return systemd.Start(unit)
	},
}

var mapBackupStopCmd = &cobra.Command{
	Use:               "stop <world>",
	Short:             "Stop a running map backup",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: mapSingleWorldCompletionFunc,
	RunE: func(cmd *cobra.Command, args []string) error {
		unit := systemd.FormatUnitName("minecraft-map-backup", args[0], systemd.UnitService)
		return systemd.Stop(unit)
	},
}

var mapBackupEnableCmd = &cobra.Command{
	Use:               "enable <world>",
	Short:             "Enable the map backup timer",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: mapSingleWorldCompletionFunc,
	RunE: func(cmd *cobra.Command, args []string) error {
		timer := systemd.FormatUnitName("minecraft-map-backup", args[0], systemd.UnitTimer)
		return systemd.EnableNow(timer)
	},
}

var mapBackupDisableCmd = &cobra.Command{
	Use:               "disable <world>",
	Short:             "Disable the map backup timer",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: mapSingleWorldCompletionFunc,
	RunE: func(cmd *cobra.Command, args []string) error {
		timer := systemd.FormatUnitName("minecraft-map-backup", args[0], systemd.UnitTimer)
		return systemd.Disable(timer)
	},
}

var mapBackupLogsCmd = &cobra.Command{
	Use:               "logs <world>",
	Short:             "View logs for the map backup service",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: mapSingleWorldCompletionFunc,
	RunE: func(cmd *cobra.Command, args []string) error {
		unit := systemd.FormatUnitName("minecraft-map-backup", args[0], systemd.UnitService)
		opts := systemd.LogOptions{
			Follow:  mapLogsFollow,
			Lines:   mapLogsLines,
			Since:   mapLogsSince,
			Output:  mapLogsOutput,
			NoPager: mapLogsNoPager,
		}
		return systemd.Logs(unit, opts)
	},
}

// Map update service commands
var mapUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Manage map update service",
}

var mapUpdateStatusCmd = &cobra.Command{
	Use:               "status <world>",
	Short:             "Show status of the map update service and timer",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: mapSingleWorldCompletionFunc,
	RunE: func(cmd *cobra.Command, args []string) error {
		service := systemd.FormatUnitName("minecraft-map-update", args[0], systemd.UnitService)
		timer := systemd.FormatUnitName("minecraft-map-update", args[0], systemd.UnitTimer)
		fmt.Println("=== Service ===")
		if err := systemd.Status(service); err != nil {
			// Continue to show timer status even if service status fails
		}
		fmt.Println("\n=== Timer ===")
		return systemd.Status(timer)
	},
}

var mapUpdateStartCmd = &cobra.Command{
	Use:               "start <world>",
	Short:             "Trigger a map update now",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: mapSingleWorldCompletionFunc,
	RunE: func(cmd *cobra.Command, args []string) error {
		unit := systemd.FormatUnitName("minecraft-map-update", args[0], systemd.UnitService)
		return systemd.Start(unit)
	},
}

var mapUpdateStopCmd = &cobra.Command{
	Use:               "stop <world>",
	Short:             "Stop a running map update",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: mapSingleWorldCompletionFunc,
	RunE: func(cmd *cobra.Command, args []string) error {
		unit := systemd.FormatUnitName("minecraft-map-update", args[0], systemd.UnitService)
		return systemd.Stop(unit)
	},
}

var mapUpdateEnableCmd = &cobra.Command{
	Use:               "enable <world>",
	Short:             "Enable the map update timer",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: mapSingleWorldCompletionFunc,
	RunE: func(cmd *cobra.Command, args []string) error {
		timer := systemd.FormatUnitName("minecraft-map-update", args[0], systemd.UnitTimer)
		return systemd.EnableNow(timer)
	},
}

var mapUpdateDisableCmd = &cobra.Command{
	Use:               "disable <world>",
	Short:             "Disable the map update timer",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: mapSingleWorldCompletionFunc,
	RunE: func(cmd *cobra.Command, args []string) error {
		timer := systemd.FormatUnitName("minecraft-map-update", args[0], systemd.UnitTimer)
		return systemd.Disable(timer)
	},
}

var mapUpdateLogsCmd = &cobra.Command{
	Use:               "logs <world>",
	Short:             "View logs for the map update service",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: mapSingleWorldCompletionFunc,
	RunE: func(cmd *cobra.Command, args []string) error {
		unit := systemd.FormatUnitName("minecraft-map-update", args[0], systemd.UnitService)
		opts := systemd.LogOptions{
			Follow:  mapLogsFollow,
			Lines:   mapLogsLines,
			Since:   mapLogsSince,
			Output:  mapLogsOutput,
			NoPager: mapLogsNoPager,
		}
		return systemd.Logs(unit, opts)
	},
}

func init() {
	mapBuildCmd.Flags().String("map", "", "Build only a specific map (by name)")
	mapBuildCmd.Flags().Bool("force", false, "Force rebuild even if map is up to date")
	mapBuildCmd.Flags().String("lock-file", "", "Path to lock file (default: from config)")
	mapBuildCmd.Flags().Duration("lock-timeout", 0, "Maximum time to wait for lock (0 = block forever)")
	mapBuildCmd.Flags().Bool("no-lock", false, "Disable file locking")
	mapBuildCmd.Flags().Bool("non-blocking", false, "Exit immediately if lock is held")
	mapBuildCmd.Flags().Bool("parallel", false, "Process multiple worlds in parallel")
	mapBuildCmd.Flags().Int("max-workers", runtime.NumCPU(), "Maximum number of parallel workers")
	mapBuildCmd.Flags().String("log-level", "warning", "uNmINeD log level (verbose, debug, information, warning, error, fatal)")

	mapPreviewCmd.Flags().String("log-level", "warning", "uNmINeD log level (verbose, debug, information, warning, error, fatal)")

	mapManifestCmd.Flags().Bool("no-preview", false, "Skip preview generation")
	mapManifestCmd.Flags().Bool("preview-only", false, "Only generate previews, skip manifest")
	mapManifestCmd.Flags().Bool("parallel", false, "Process multiple worlds in parallel")
	mapManifestCmd.Flags().Int("max-workers", runtime.NumCPU(), "Maximum number of parallel workers")
	mapManifestCmd.Flags().Bool("update-index", false, "Update aggregate manifest and HTML index after manifest generation")
	mapManifestCmd.Flags().String("log-level", "warning", "uNmINeD log level (verbose, debug, information, warning, error, fatal)")

	mapConfigGenerateCmd.Flags().Bool("force", false, "Overwrite existing config file")
	mapConfigGenerateCmd.Flags().Int("radius", 2048, "Radius of the spawn area zoom region")
	mapConfigGenerateCmd.Flags().String("output", "", "Output path for config file (default: <worldPath>/map-config.yml)")

	mapConfigGetCmd.Flags().String("format", "yaml", "Output format (yaml, json)")

	MapCmd.AddCommand(mapBuildCmd)
	MapCmd.AddCommand(mapPreviewCmd)
	MapCmd.AddCommand(mapManifestCmd)
	MapCmd.AddCommand(mapIndexCmd)

	mapConfigCmd.AddCommand(mapConfigGenerateCmd)
	mapConfigCmd.AddCommand(mapConfigGetCmd)
	mapConfigCmd.AddCommand(mapConfigSetCmd)
	mapConfigCmd.AddCommand(mapConfigValidateCmd)
	mapConfigCmd.AddCommand(mapConfigEditCmd)
	MapCmd.AddCommand(mapConfigCmd)

	// Map backup service commands
	MapCmd.AddCommand(mapBackupCmd)
	mapBackupCmd.AddCommand(mapBackupStatusCmd)
	mapBackupCmd.AddCommand(mapBackupStartCmd)
	mapBackupCmd.AddCommand(mapBackupStopCmd)
	mapBackupCmd.AddCommand(mapBackupEnableCmd)
	mapBackupCmd.AddCommand(mapBackupDisableCmd)
	mapBackupCmd.AddCommand(mapBackupLogsCmd)

	// Map update service commands
	MapCmd.AddCommand(mapUpdateCmd)
	mapUpdateCmd.AddCommand(mapUpdateStatusCmd)
	mapUpdateCmd.AddCommand(mapUpdateStartCmd)
	mapUpdateCmd.AddCommand(mapUpdateStopCmd)
	mapUpdateCmd.AddCommand(mapUpdateEnableCmd)
	mapUpdateCmd.AddCommand(mapUpdateDisableCmd)
	mapUpdateCmd.AddCommand(mapUpdateLogsCmd)

	// Logs flags for map backup
	mapBackupLogsCmd.Flags().BoolVarP(&mapLogsFollow, "follow", "f", false, "Follow log output")
	mapBackupLogsCmd.Flags().IntVarP(&mapLogsLines, "lines", "n", 100, "Number of lines to show")
	mapBackupLogsCmd.Flags().StringVar(&mapLogsSince, "since", "", "Show logs since timestamp (e.g., '1 hour ago')")
	mapBackupLogsCmd.Flags().StringVarP(&mapLogsOutput, "output", "o", "", "Output format (short, json, cat)")
	mapBackupLogsCmd.Flags().BoolVar(&mapLogsNoPager, "no-pager", false, "Disable paging")

	// Logs flags for map update
	mapUpdateLogsCmd.Flags().BoolVarP(&mapLogsFollow, "follow", "f", false, "Follow log output")
	mapUpdateLogsCmd.Flags().IntVarP(&mapLogsLines, "lines", "n", 100, "Number of lines to show")
	mapUpdateLogsCmd.Flags().StringVar(&mapLogsSince, "since", "", "Show logs since timestamp (e.g., '1 hour ago')")
	mapUpdateLogsCmd.Flags().StringVarP(&mapLogsOutput, "output", "o", "", "Output format (short, json, cat)")
	mapUpdateLogsCmd.Flags().BoolVar(&mapLogsNoPager, "no-pager", false, "Disable paging")
}

// runInteractiveEditor runs the interactive configuration editor
func runInteractiveEditor(worldPath string, mapConfig *config.MapConfig) error {
	for {
		var choice string
		prompt := &survey.Select{
			Message: "What would you like to edit?",
			Options: []string{
				"Edit defaults",
				"Edit maps",
				"Add new map",
				"Delete map",
				"Validate config",
				"Save and exit",
				"Exit without saving",
			},
		}
		if err := survey.AskOne(prompt, &choice); err != nil {
			return err
		}

		switch choice {
		case "Edit defaults":
			if err := editDefaults(mapConfig); err != nil {
				fmt.Printf("Error: %v\n", err)
			}
		case "Edit maps":
			if err := editMaps(mapConfig); err != nil {
				fmt.Printf("Error: %v\n", err)
			}
		case "Add new map":
			if err := addMap(mapConfig); err != nil {
				fmt.Printf("Error: %v\n", err)
			}
		case "Delete map":
			if err := deleteMap(mapConfig); err != nil {
				fmt.Printf("Error: %v\n", err)
			}
		case "Validate config":
			errs := config.ValidateMapConfig(mapConfig)
			if len(errs) > 0 {
				fmt.Printf("Validation failed with %d error(s):\n", len(errs))
				for _, e := range errs {
					fmt.Printf("  - %s\n", e)
				}
			} else {
				fmt.Println("✓ Config is valid")
			}
		case "Save and exit":
			if err := config.SaveMapConfig(worldPath, mapConfig); err != nil {
				return fmt.Errorf("failed to save config: %w", err)
			}
			fmt.Println("✓ Config saved successfully")
			return nil
		case "Exit without saving":
			fmt.Println("Exiting without saving...")
			return nil
		}
	}
}

func editDefaults(mapConfig *config.MapConfig) error {
	var zoomoutStr string
	var zoominStr string
	var imageFormat string
	var chunkProcessorsStr string

	zoomoutPrompt := &survey.Input{
		Message: "Zoomout (default):",
		Default: fmt.Sprintf("%d", mapConfig.Defaults.Zoomout),
	}
	if err := survey.AskOne(zoomoutPrompt, &zoomoutStr); err != nil {
		return err
	}
	zoomout, err := strconv.Atoi(zoomoutStr)
	if err != nil {
		return fmt.Errorf("invalid zoomout value: %w", err)
	}

	zoominPrompt := &survey.Input{
		Message: "Zoomin (default):",
		Default: fmt.Sprintf("%d", mapConfig.Defaults.Zoomin),
	}
	if err := survey.AskOne(zoominPrompt, &zoominStr); err != nil {
		return err
	}
	zoomin, err := strconv.Atoi(zoominStr)
	if err != nil {
		return fmt.Errorf("invalid zoomin value: %w", err)
	}

	imageFormatPrompt := &survey.Select{
		Message: "Image format:",
		Options: []string{"jpeg", "png", "webp"},
		Default: mapConfig.Defaults.ImageFormat,
	}
	if err := survey.AskOne(imageFormatPrompt, &imageFormat); err != nil {
		return err
	}

	chunkProcessorsPrompt := &survey.Input{
		Message: "Chunk processors:",
		Default: fmt.Sprintf("%d", mapConfig.Defaults.ChunkProcessors),
	}
	if err := survey.AskOne(chunkProcessorsPrompt, &chunkProcessorsStr); err != nil {
		return err
	}
	chunkProcessors, err := strconv.Atoi(chunkProcessorsStr)
	if err != nil {
		return fmt.Errorf("invalid chunkProcessors value: %w", err)
	}

	mapConfig.Defaults.Zoomout = zoomout
	mapConfig.Defaults.Zoomin = zoomin
	mapConfig.Defaults.ImageFormat = imageFormat
	mapConfig.Defaults.ChunkProcessors = chunkProcessors

	fmt.Println("✓ Defaults updated")
	return nil
}

func editMaps(mapConfig *config.MapConfig) error {
	if len(mapConfig.Maps) == 0 {
		fmt.Println("No maps defined. Add a map first.")
		return nil
	}

	// Create map selection list
	mapNames := make([]string, len(mapConfig.Maps))
	for i, m := range mapConfig.Maps {
		mapNames[i] = fmt.Sprintf("%s (%s)", m.Name, m.Dimension)
	}

	var selected string
	prompt := &survey.Select{
		Message: "Select map to edit:",
		Options: mapNames,
	}
	if err := survey.AskOne(prompt, &selected); err != nil {
		return err
	}

	// Find selected map
	var mapIdx int
	for i, m := range mapConfig.Maps {
		if fmt.Sprintf("%s (%s)", m.Name, m.Dimension) == selected {
			mapIdx = i
			break
		}
	}

	// Edit the map
	return editMap(mapConfig, mapIdx)
}

func editMap(mapConfig *config.MapConfig, idx int) error {
	if idx < 0 || idx >= len(mapConfig.Maps) {
		return fmt.Errorf("invalid map index")
	}

	m := &mapConfig.Maps[idx]

	// Edit name
	var name string
	namePrompt := &survey.Input{
		Message: "Map name:",
		Default: m.Name,
	}
	if err := survey.AskOne(namePrompt, &name); err != nil {
		return err
	}
	m.Name = name

	// Edit dimension
	var dimension string
	dimPrompt := &survey.Select{
		Message: "Dimension:",
		Options: []string{"overworld", "nether", "end"},
		Default: m.Dimension,
	}
	if err := survey.AskOne(dimPrompt, &dimension); err != nil {
		return err
	}
	m.Dimension = dimension

	// Edit output subdir
	var outputSubdir string
	outputPrompt := &survey.Input{
		Message: "Output subdir:",
		Default: m.OutputSubdir,
	}
	if err := survey.AskOne(outputPrompt, &outputSubdir); err != nil {
		return err
	}
	m.OutputSubdir = outputSubdir

	fmt.Println("✓ Map updated")
	return nil
}

func addMap(mapConfig *config.MapConfig) error {
	var name string
	var dimension string
	var outputSubdir string

	namePrompt := &survey.Input{
		Message: "Map name:",
	}
	if err := survey.AskOne(namePrompt, &name); err != nil {
		return err
	}

	dimPrompt := &survey.Select{
		Message: "Dimension:",
		Options: []string{"overworld", "nether", "end"},
	}
	if err := survey.AskOne(dimPrompt, &dimension); err != nil {
		return err
	}

	outputPrompt := &survey.Input{
		Message: "Output subdir:",
		Default: name,
	}
	if err := survey.AskOne(outputPrompt, &outputSubdir); err != nil {
		return err
	}

	mapConfig.Maps = append(mapConfig.Maps, config.MapDefinition{
		Name:         name,
		Dimension:    dimension,
		OutputSubdir: outputSubdir,
	})

	fmt.Printf("✓ Map '%s' added\n", name)
	return nil
}

func deleteMap(mapConfig *config.MapConfig) error {
	if len(mapConfig.Maps) == 0 {
		fmt.Println("No maps to delete.")
		return nil
	}

	// Create map selection list
	mapNames := make([]string, len(mapConfig.Maps))
	for i, m := range mapConfig.Maps {
		mapNames[i] = fmt.Sprintf("%s (%s)", m.Name, m.Dimension)
	}

	var selected string
	prompt := &survey.Select{
		Message: "Select map to delete:",
		Options: mapNames,
	}
	if err := survey.AskOne(prompt, &selected); err != nil {
		return err
	}

	// Confirm deletion
	var confirm bool
	confirmPrompt := &survey.Confirm{
		Message: "Are you sure you want to delete this map?",
		Default: false,
	}
	if err := survey.AskOne(confirmPrompt, &confirm); err != nil {
		return err
	}

	if !confirm {
		return nil
	}

	// Find and delete
	for i, m := range mapConfig.Maps {
		if fmt.Sprintf("%s (%s)", m.Name, m.Dimension) == selected {
			mapConfig.Maps = append(mapConfig.Maps[:i], mapConfig.Maps[i+1:]...)
			fmt.Printf("✓ Map '%s' deleted\n", m.Name)
			return nil
		}
	}

	return nil
}

// buildBatch processes multiple worlds in batch
func buildBatch(worldNames []string, baseOpts maps.BuildOptions, parallel bool, maxWorkers int) error {
	if !parallel {
		// Sequential processing
		var firstErr error
		for _, worldName := range worldNames {
			builder := maps.NewBuilder()
			opts := baseOpts
			opts.WorldName = worldName
			if err := builder.Build(opts); err != nil {
				log.Error().Err(err).Str("world", worldName).Msg("failed to build maps")
				if firstErr == nil {
					firstErr = err
				}
			}
		}
		return firstErr
	}

	// Parallel processing with worker pool
	if maxWorkers <= 0 {
		maxWorkers = runtime.NumCPU()
	}
	if maxWorkers > len(worldNames) {
		maxWorkers = len(worldNames)
	}

	var wg sync.WaitGroup
	semaphore := make(chan struct{}, maxWorkers)
	errors := make([]error, len(worldNames))

	for i, worldName := range worldNames {
		wg.Add(1)
		go func(idx int, name string) {
			defer wg.Done()
			semaphore <- struct{}{}        // Acquire
			defer func() { <-semaphore }() // Release

			builder := maps.NewBuilder()
			opts := baseOpts
			opts.WorldName = name
			errors[idx] = builder.Build(opts)
		}(i, worldName)
	}

	wg.Wait()

	// Return first error if any
	for _, err := range errors {
		if err != nil {
			return err
		}
	}
	return nil
}

// manifestBatch processes multiple worlds for manifest generation
func manifestBatch(worldNames []string, baseOpts maps.ManifestOptions, parallel bool, maxWorkers int) error {
	if !parallel {
		// Sequential processing
		var firstErr error
		for _, worldName := range worldNames {
			builder := maps.NewManifestBuilder()
			opts := baseOpts
			opts.WorldName = worldName
			if err := builder.BuildManifests(worldName, opts); err != nil {
				log.Error().Err(err).Str("world", worldName).Msg("failed to build manifests")
				if firstErr == nil {
					firstErr = err
				}
			}
		}
		return firstErr
	}

	// Parallel processing with worker pool
	if maxWorkers <= 0 {
		maxWorkers = runtime.NumCPU()
	}
	if maxWorkers > len(worldNames) {
		maxWorkers = len(worldNames)
	}

	var wg sync.WaitGroup
	semaphore := make(chan struct{}, maxWorkers)
	errors := make([]error, len(worldNames))

	for i, worldName := range worldNames {
		wg.Add(1)
		go func(idx int, name string) {
			defer wg.Done()
			semaphore <- struct{}{}        // Acquire
			defer func() { <-semaphore }() // Release

			builder := maps.NewManifestBuilder()
			opts := baseOpts
			opts.WorldName = name
			errors[idx] = builder.BuildManifests(name, opts)
		}(i, worldName)
	}

	wg.Wait()

	// Return first error if any
	for _, err := range errors {
		if err != nil {
			return err
		}
	}
	return nil
}
