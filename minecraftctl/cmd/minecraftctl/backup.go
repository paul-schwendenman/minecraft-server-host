package main

import (
	"fmt"

	"github.com/paul/minecraftctl/internal/commands"
	"github.com/paul/minecraftctl/pkg/backup"
	"github.com/paul/minecraftctl/pkg/worlds"
	"github.com/spf13/cobra"
)

// BackupCmd is an alias for the command defined in internal/commands
var BackupCmd = commands.BackupCmd

// backupWorldCompletionFunc provides tab completion for world names
func backupWorldCompletionFunc(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) != 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	names, err := worlds.GetWorldNames()
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}
	// Add "all" as an option
	names = append([]string{"all"}, names...)
	return names, cobra.ShellCompDirectiveNoFileComp
}

var backupListCmd = &cobra.Command{
	Use:   "list [tag]",
	Short: "List available backup snapshots",
	Long:  "List all available backup snapshots. Optionally filter by tag (world name or 'all').",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if !backup.IsResticInstalled() {
			return fmt.Errorf("restic is not installed")
		}

		cfg, err := backup.LoadConfig()
		if err != nil {
			return err
		}

		tag := ""
		if len(args) > 0 {
			tag = args[0]
		}

		return cfg.List(tag)
	},
}

var backupCreateCmd = &cobra.Command{
	Use:               "create [world]",
	Short:             "Create a new backup",
	Long:              "Create a new backup of the specified world, or all worlds if no world is specified.",
	Args:              cobra.MaximumNArgs(1),
	ValidArgsFunction: backupWorldCompletionFunc,
	RunE: func(cmd *cobra.Command, args []string) error {
		if !backup.IsResticInstalled() {
			return fmt.Errorf("restic is not installed")
		}

		cfg, err := backup.LoadConfig()
		if err != nil {
			return err
		}

		world := "all"
		if len(args) > 0 {
			world = args[0]
		}

		return cfg.Create(world)
	},
}

var restoreTarget string

var backupRestoreCmd = &cobra.Command{
	Use:   "restore <snapshot>",
	Short: "Restore a backup snapshot",
	Long: `Restore a backup snapshot.

Use 'latest' to restore the most recent snapshot.
Use --target to restore to a custom location instead of the original path.

Examples:
  minecraftctl backup restore latest
  minecraftctl backup restore abc123
  minecraftctl backup restore latest --target /tmp/restore-test`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if !backup.IsResticInstalled() {
			return fmt.Errorf("restic is not installed")
		}

		cfg, err := backup.LoadConfig()
		if err != nil {
			return err
		}

		snapshot := args[0]
		target := restoreTarget
		if target == "" {
			target = "/"
		}

		return cfg.Restore(snapshot, target)
	},
}

var backupPruneCmd = &cobra.Command{
	Use:   "prune",
	Short: "Remove old backup snapshots",
	Long: `Remove old backup snapshots according to the retention policy.

Retention policy:
  - Keep 7 daily snapshots
  - Keep 4 weekly snapshots
  - Keep 3 monthly snapshots

This command also verifies repository integrity after pruning.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if !backup.IsResticInstalled() {
			return fmt.Errorf("restic is not installed")
		}

		cfg, err := backup.LoadConfig()
		if err != nil {
			return err
		}

		return cfg.Prune()
	},
}

var backupStatsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Show backup repository statistics",
	Long:  "Display statistics about the backup repository including size and deduplication.",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if !backup.IsResticInstalled() {
			return fmt.Errorf("restic is not installed")
		}

		cfg, err := backup.LoadConfig()
		if err != nil {
			return err
		}

		return cfg.Stats()
	},
}

var backupCheckCmd = &cobra.Command{
	Use:   "check",
	Short: "Verify backup repository integrity",
	Long:  "Check the backup repository for errors and verify data integrity.",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if !backup.IsResticInstalled() {
			return fmt.Errorf("restic is not installed")
		}

		cfg, err := backup.LoadConfig()
		if err != nil {
			return err
		}

		return cfg.Check()
	},
}

func init() {
	// Add flags
	backupRestoreCmd.Flags().StringVarP(&restoreTarget, "target", "t", "", "Target directory for restore (default: original location)")

	// Add subcommands to backup command
	BackupCmd.AddCommand(backupListCmd)
	BackupCmd.AddCommand(backupCreateCmd)
	BackupCmd.AddCommand(backupRestoreCmd)
	BackupCmd.AddCommand(backupPruneCmd)
	BackupCmd.AddCommand(backupStatsCmd)
	BackupCmd.AddCommand(backupCheckCmd)
}
