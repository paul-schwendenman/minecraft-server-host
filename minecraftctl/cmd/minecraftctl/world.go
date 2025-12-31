package main

import (
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/paul/minecraftctl/internal/commands"
	"github.com/paul/minecraftctl/pkg/systemd"
	"github.com/paul/minecraftctl/pkg/worlds"
	"github.com/spf13/cobra"
)

// worldCompletionFunc provides tab completion for world names
func worldCompletionFunc(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) != 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	names, err := worlds.GetWorldNames()
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}
	return names, cobra.ShellCompDirectiveNoFileComp
}

// WorldCmd is an alias for the command defined in internal/commands
var WorldCmd = commands.WorldCmd

var worldListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all worlds",
	Long:  "List all worlds. Use --full to show service status, enabled state, and timer information.",
	RunE: func(cmd *cobra.Command, args []string) error {
		worldList, err := worlds.ListWorlds()
		if err != nil {
			return err
		}

		if len(worldList) == 0 {
			fmt.Println("No worlds found")
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

		if listFull {
			fmt.Fprintln(w, "NAME\tVERSION\tDIFFICULTY\tMAP CONFIG\tSERVICE\tENABLED\tTIMERS")
			for _, world := range worldList {
				mapCfg := "no"
				if world.HasMapConfig {
					mapCfg = "yes"
				}

				// Get service status
				serviceUnit := systemd.FormatUnitName("minecraft", world.Name, systemd.UnitService)
				serviceState := systemd.GetActiveState(serviceUnit)

				// Get enabled status
				enabled, _ := systemd.IsEnabled(serviceUnit)
				enabledStr := "no"
				if enabled {
					enabledStr = "yes"
				}

				// Get timer statuses
				buildTimer := systemd.FormatUnitName("minecraft-map-build", world.Name, systemd.UnitTimer)
				backupTimer := systemd.FormatUnitName("minecraft-world-backup", world.Name, systemd.UnitTimer)

				buildEnabled, _ := systemd.IsEnabled(buildTimer)
				backupEnabled, _ := systemd.IsEnabled(backupTimer)

				timers := ""
				if buildEnabled {
					timers += "✓ build "
				} else {
					timers += "✗ build "
				}
				if backupEnabled {
					timers += "✓ backup"
				} else {
					timers += "✗ backup"
				}

				fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
					world.Name, world.Version, world.Difficulty, mapCfg,
					serviceState, enabledStr, timers)
			}
		} else {
			fmt.Fprintln(w, "NAME\tVERSION\tDIFFICULTY\tMAP CONFIG")
			for _, world := range worldList {
				mapCfg := "no"
				if world.HasMapConfig {
					mapCfg = "yes"
				}
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", world.Name, world.Version, world.Difficulty, mapCfg)
			}
		}
		w.Flush()

		return nil
	},
}

var worldInfoCmd = &cobra.Command{
	Use:               "info <world>",
	Short:             "Show detailed information about a world",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: worldCompletionFunc,
	RunE: func(cmd *cobra.Command, args []string) error {
		worldName := args[0]
		info, err := worlds.GetWorldInfo(worldName)
		if err != nil {
			return err
		}

		fmt.Printf("World: %s\n", info.Name)
		fmt.Printf("Path: %s\n", info.Path)
		fmt.Printf("Version: %s\n", info.Version)
		fmt.Printf("Level Name: %s\n", info.LevelName)
		fmt.Printf("Difficulty: %s\n", info.Difficulty)
		fmt.Printf("Game Type: %s\n", info.GameType)
		fmt.Printf("Spawn: (%d, %d, %d)\n", info.SpawnX, info.SpawnY, info.SpawnZ)
		if !info.LastPlayed.IsZero() {
			fmt.Printf("Last Played: %s\n", info.LastPlayed.Format(time.RFC3339))
		}
		fmt.Printf("Map Config: ")
		if info.HasMapConfig {
			fmt.Println("yes")
		} else {
			fmt.Println("no")
		}

		return nil
	},
}

var (
	createVersion     string
	createSeed        string
	createNoMapConfig bool
	createNoSystemd   bool
)

var (
	upgradeVersion string
	upgradeStop    bool
)

// List command flags
var listFull bool

// Logs command flags
var (
	logsFollow  bool
	logsLines   int
	logsSince   string
	logsOutput  string
	logsNoPager bool
)

var worldCreateCmd = &cobra.Command{
	Use:   "create <world-name>",
	Short: "Create a new Minecraft world",
	Long:  "Create a new Minecraft world directory with server configuration. Requires the Minecraft server jar to be installed at /opt/minecraft/jars/minecraft_server_<version>.jar",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		worldName := args[0]

		if createVersion == "" {
			return fmt.Errorf("--version is required")
		}

		opts := worlds.CreateWorldOptions{
			Version:         createVersion,
			Seed:            createSeed,
			CreateMapConfig: !createNoMapConfig,
			EnableSystemd:   !createNoSystemd,
		}

		if err := worlds.CreateWorld(worldName, opts); err != nil {
			return err
		}

		fmt.Printf("World '%s' created successfully with jar version %s\n", worldName, createVersion)
		if opts.CreateMapConfig {
			fmt.Println("Default map-config.yml created")
		}
		if opts.EnableSystemd {
			fmt.Printf("Systemd service minecraft@%s.service enabled and started\n", worldName)
		}

		return nil
	},
}

var worldRegisterCmd = &cobra.Command{
	Use:               "register <world-name>",
	Short:             "Register an existing world with systemd services",
	Long:              "Register an existing world by enabling systemd services and timers without modifying world files. This is used to \"reattach\" a world from an EBS volume to a new server instance.",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: worldCompletionFunc,
	RunE: func(cmd *cobra.Command, args []string) error {
		worldName := args[0]

		if err := worlds.RegisterWorld(worldName); err != nil {
			return err
		}

		fmt.Printf("World '%s' registered successfully\n", worldName)
		fmt.Printf("Systemd service minecraft@%s.service enabled and started\n", worldName)
		fmt.Printf("Timers enabled: minecraft-map-build@%s.timer, minecraft-world-backup@%s.timer, minecraft-map-backup@%s.timer\n",
			worldName, worldName, worldName)

		return nil
	},
}

var worldUpgradeCmd = &cobra.Command{
	Use:   "upgrade <world-name>",
	Short: "Upgrade a world to a newer Minecraft version",
	Long: `Upgrade a world's server.jar symlink to point to a newer version.

This command:
1. Verifies the world exists
2. Checks that the target version is newer than the current version
3. Verifies the target JAR exists in the jars directory
4. Optionally stops the running server (with --stop flag)
5. Updates the server.jar symlink to point to the new version

Note: This only updates the JAR symlink. Minecraft will automatically upgrade
world data when the server starts with the new version. World upgrades are
one-way and cannot be reverted.`,
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: worldCompletionFunc,
	RunE: func(cmd *cobra.Command, args []string) error {
		worldName := args[0]

		if upgradeVersion == "" {
			return fmt.Errorf("--version is required")
		}

		opts := worlds.UpgradeOptions{
			TargetVersion: upgradeVersion,
			StopService:   upgradeStop,
		}

		result, err := worlds.UpgradeWorld(worldName, opts)
		if err != nil {
			return err
		}

		fmt.Printf("World '%s' upgraded successfully\n", result.WorldName)
		fmt.Printf("  Previous version: %s\n", result.PreviousVersion)
		fmt.Printf("  New version: %s\n", result.NewVersion)
		if result.ServiceStopped {
			fmt.Printf("\nNote: Service minecraft@%s.service was stopped.\n", worldName)
			fmt.Println("Start it with: systemctl start minecraft@" + worldName + ".service")
		}

		return nil
	},
}

// Service management commands
var worldStatusCmd = &cobra.Command{
	Use:               "status <world>",
	Short:             "Show status of the Minecraft server service",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: worldCompletionFunc,
	RunE: func(cmd *cobra.Command, args []string) error {
		unit := systemd.FormatUnitName("minecraft", args[0], systemd.UnitService)
		return systemd.Status(unit)
	},
}

var worldStartCmd = &cobra.Command{
	Use:               "start <world>",
	Short:             "Start the Minecraft server service",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: worldCompletionFunc,
	RunE: func(cmd *cobra.Command, args []string) error {
		unit := systemd.FormatUnitName("minecraft", args[0], systemd.UnitService)
		return systemd.Start(unit)
	},
}

var worldStopCmd = &cobra.Command{
	Use:               "stop <world>",
	Short:             "Stop the Minecraft server service",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: worldCompletionFunc,
	RunE: func(cmd *cobra.Command, args []string) error {
		unit := systemd.FormatUnitName("minecraft", args[0], systemd.UnitService)
		return systemd.Stop(unit)
	},
}

var worldRestartCmd = &cobra.Command{
	Use:               "restart <world>",
	Short:             "Restart the Minecraft server service",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: worldCompletionFunc,
	RunE: func(cmd *cobra.Command, args []string) error {
		unit := systemd.FormatUnitName("minecraft", args[0], systemd.UnitService)
		return systemd.Restart(unit)
	},
}

var worldEnableCmd = &cobra.Command{
	Use:               "enable <world>",
	Short:             "Enable the Minecraft server service to start on boot",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: worldCompletionFunc,
	RunE: func(cmd *cobra.Command, args []string) error {
		unit := systemd.FormatUnitName("minecraft", args[0], systemd.UnitService)
		return systemd.Enable(unit)
	},
}

var worldDisableCmd = &cobra.Command{
	Use:               "disable <world>",
	Short:             "Disable the Minecraft server service from starting on boot",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: worldCompletionFunc,
	RunE: func(cmd *cobra.Command, args []string) error {
		unit := systemd.FormatUnitName("minecraft", args[0], systemd.UnitService)
		return systemd.Disable(unit)
	},
}

var worldLogsCmd = &cobra.Command{
	Use:               "logs <world>",
	Short:             "View logs for the Minecraft server service",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: worldCompletionFunc,
	RunE: func(cmd *cobra.Command, args []string) error {
		unit := systemd.FormatUnitName("minecraft", args[0], systemd.UnitService)
		opts := systemd.LogOptions{
			Follow:  logsFollow,
			Lines:   logsLines,
			Since:   logsSince,
			Output:  logsOutput,
			NoPager: logsNoPager,
		}
		return systemd.Logs(unit, opts)
	},
}

// World backup commands
var worldBackupCmd = &cobra.Command{
	Use:   "backup",
	Short: "Manage world backup service",
}

var worldBackupStatusCmd = &cobra.Command{
	Use:               "status <world>",
	Short:             "Show status of the world backup service and timer",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: worldCompletionFunc,
	RunE: func(cmd *cobra.Command, args []string) error {
		service := systemd.FormatUnitName("minecraft-world-backup", args[0], systemd.UnitService)
		timer := systemd.FormatUnitName("minecraft-world-backup", args[0], systemd.UnitTimer)
		fmt.Println("=== Service ===")
		if err := systemd.Status(service); err != nil {
			// Continue to show timer status even if service status fails
		}
		fmt.Println("\n=== Timer ===")
		return systemd.Status(timer)
	},
}

var worldBackupStartCmd = &cobra.Command{
	Use:               "start <world>",
	Short:             "Trigger a world backup now",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: worldCompletionFunc,
	RunE: func(cmd *cobra.Command, args []string) error {
		unit := systemd.FormatUnitName("minecraft-world-backup", args[0], systemd.UnitService)
		return systemd.Start(unit)
	},
}

var worldBackupStopCmd = &cobra.Command{
	Use:               "stop <world>",
	Short:             "Stop a running world backup",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: worldCompletionFunc,
	RunE: func(cmd *cobra.Command, args []string) error {
		unit := systemd.FormatUnitName("minecraft-world-backup", args[0], systemd.UnitService)
		return systemd.Stop(unit)
	},
}

var worldBackupEnableCmd = &cobra.Command{
	Use:               "enable <world>",
	Short:             "Enable the world backup timer",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: worldCompletionFunc,
	RunE: func(cmd *cobra.Command, args []string) error {
		timer := systemd.FormatUnitName("minecraft-world-backup", args[0], systemd.UnitTimer)
		return systemd.EnableNow(timer)
	},
}

var worldBackupDisableCmd = &cobra.Command{
	Use:               "disable <world>",
	Short:             "Disable the world backup timer",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: worldCompletionFunc,
	RunE: func(cmd *cobra.Command, args []string) error {
		timer := systemd.FormatUnitName("minecraft-world-backup", args[0], systemd.UnitTimer)
		return systemd.Disable(timer)
	},
}

var worldBackupLogsCmd = &cobra.Command{
	Use:               "logs <world>",
	Short:             "View logs for the world backup service",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: worldCompletionFunc,
	RunE: func(cmd *cobra.Command, args []string) error {
		unit := systemd.FormatUnitName("minecraft-world-backup", args[0], systemd.UnitService)
		opts := systemd.LogOptions{
			Follow:  logsFollow,
			Lines:   logsLines,
			Since:   logsSince,
			Output:  logsOutput,
			NoPager: logsNoPager,
		}
		return systemd.Logs(unit, opts)
	},
}

func init() {
	// List command flags
	worldListCmd.Flags().BoolVar(&listFull, "full", false, "Show service status, enabled state, and timer information")

	WorldCmd.AddCommand(worldListCmd)
	WorldCmd.AddCommand(worldInfoCmd)
	WorldCmd.AddCommand(worldCreateCmd)
	WorldCmd.AddCommand(worldRegisterCmd)
	WorldCmd.AddCommand(worldUpgradeCmd)

	// Service management commands
	WorldCmd.AddCommand(worldStatusCmd)
	WorldCmd.AddCommand(worldStartCmd)
	WorldCmd.AddCommand(worldStopCmd)
	WorldCmd.AddCommand(worldRestartCmd)
	WorldCmd.AddCommand(worldEnableCmd)
	WorldCmd.AddCommand(worldDisableCmd)
	WorldCmd.AddCommand(worldLogsCmd)

	// World backup commands
	WorldCmd.AddCommand(worldBackupCmd)
	worldBackupCmd.AddCommand(worldBackupStatusCmd)
	worldBackupCmd.AddCommand(worldBackupStartCmd)
	worldBackupCmd.AddCommand(worldBackupStopCmd)
	worldBackupCmd.AddCommand(worldBackupEnableCmd)
	worldBackupCmd.AddCommand(worldBackupDisableCmd)
	worldBackupCmd.AddCommand(worldBackupLogsCmd)

	// Create command flags
	worldCreateCmd.Flags().StringVar(&createVersion, "version", "", "Minecraft server version (e.g., 1.21.1)")
	worldCreateCmd.MarkFlagRequired("version")
	worldCreateCmd.Flags().StringVar(&createSeed, "seed", "", "World seed (optional)")
	worldCreateCmd.Flags().BoolVar(&createNoMapConfig, "no-map-config", false, "Skip creating map-config.yml")
	worldCreateCmd.Flags().BoolVar(&createNoSystemd, "no-systemd", false, "Skip enabling and starting systemd service")

	// Upgrade command flags
	worldUpgradeCmd.Flags().StringVar(&upgradeVersion, "version", "", "Target Minecraft server version (e.g., 1.21.11)")
	worldUpgradeCmd.MarkFlagRequired("version")
	worldUpgradeCmd.Flags().BoolVar(&upgradeStop, "stop", false, "Automatically stop the server if running")

	// Logs command flags (shared between world logs and world backup logs)
	worldLogsCmd.Flags().BoolVarP(&logsFollow, "follow", "f", false, "Follow log output")
	worldLogsCmd.Flags().IntVarP(&logsLines, "lines", "n", 100, "Number of lines to show")
	worldLogsCmd.Flags().StringVar(&logsSince, "since", "", "Show logs since timestamp (e.g., '1 hour ago')")
	worldLogsCmd.Flags().StringVarP(&logsOutput, "output", "o", "", "Output format (short, json, cat)")
	worldLogsCmd.Flags().BoolVar(&logsNoPager, "no-pager", false, "Disable paging")

	worldBackupLogsCmd.Flags().BoolVarP(&logsFollow, "follow", "f", false, "Follow log output")
	worldBackupLogsCmd.Flags().IntVarP(&logsLines, "lines", "n", 100, "Number of lines to show")
	worldBackupLogsCmd.Flags().StringVar(&logsSince, "since", "", "Show logs since timestamp (e.g., '1 hour ago')")
	worldBackupLogsCmd.Flags().StringVarP(&logsOutput, "output", "o", "", "Output format (short, json, cat)")
	worldBackupLogsCmd.Flags().BoolVar(&logsNoPager, "no-pager", false, "Disable paging")
}
