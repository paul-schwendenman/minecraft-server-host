package main

import (
	"github.com/paul/minecraftctl/pkg/maps"
	"github.com/spf13/cobra"
)

var mapCmd = &cobra.Command{
	Use:   "map",
	Short: "Manage maps",
	Long:  "Commands for building and managing Minecraft world maps",
}

var mapBuildCmd = &cobra.Command{
	Use:   "build <world>",
	Short: "Build maps for a world",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		worldName := args[0]
		mapName, _ := cmd.Flags().GetString("map")
		force, _ := cmd.Flags().GetBool("force")

		builder := maps.NewBuilder()
		opts := maps.BuildOptions{
			WorldName: worldName,
			MapName:   mapName,
			Force:     force,
		}

		return builder.Build(opts)
	},
}

func init() {
	mapBuildCmd.Flags().String("map", "", "Build only a specific map (by name)")
	mapBuildCmd.Flags().Bool("force", false, "Force rebuild even if map is up to date")

	mapCmd.AddCommand(mapBuildCmd)
}

