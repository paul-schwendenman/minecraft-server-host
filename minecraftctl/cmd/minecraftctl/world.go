package main

import (
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/paul/minecraftctl/pkg/worlds"
	"github.com/spf13/cobra"
)

var worldCmd = &cobra.Command{
	Use:   "world",
	Short: "Manage Minecraft worlds",
	Long:  "Commands for listing and inspecting Minecraft worlds",
}

var worldListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all worlds",
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
		fmt.Fprintln(w, "NAME\tVERSION\tDIFFICULTY\tMAP CONFIG")
		for _, world := range worldList {
			mapCfg := "no"
			if world.HasMapConfig {
				mapCfg = "yes"
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", world.Name, world.Version, world.Difficulty, mapCfg)
		}
		w.Flush()

		return nil
	},
}

var worldInfoCmd = &cobra.Command{
	Use:   "info <world>",
	Short: "Show detailed information about a world",
	Args:  cobra.ExactArgs(1),
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

func init() {
	worldCmd.AddCommand(worldListCmd)
	worldCmd.AddCommand(worldInfoCmd)
}

