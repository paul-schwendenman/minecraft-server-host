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

var mapPreviewCmd = &cobra.Command{
	Use:   "preview <world> <map>",
	Short: "Generate preview image for a map",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		worldName := args[0]
		mapName := args[1]

		builder := maps.NewBuilder()
		return builder.GeneratePreview(worldName, mapName)
	},
}

var mapManifestCmd = &cobra.Command{
	Use:   "manifest <world>",
	Short: "Build manifests for all maps in a world",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		worldName := args[0]

		builder := maps.NewManifestBuilder()
		return builder.BuildManifests(worldName)
	},
}

func init() {
	mapBuildCmd.Flags().String("map", "", "Build only a specific map (by name)")
	mapBuildCmd.Flags().Bool("force", false, "Force rebuild even if map is up to date")

	mapCmd.AddCommand(mapBuildCmd)
	mapCmd.AddCommand(mapPreviewCmd)
	mapCmd.AddCommand(mapManifestCmd)
}

