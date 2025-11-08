package jars

import (
	"github.com/spf13/cobra"
)

var JarCmd = &cobra.Command{
	Use:   "jar",
	Short: "Manage Minecraft server jar files",
	Long:  `Manage and download Minecraft server jar files.`,
}

func init() {
	// Add subcommands here
	JarCmd.AddCommand(downloadCmd)
	JarCmd.AddCommand(listCmd)
	JarCmd.AddCommand(verifyCmd)
}

var downloadCmd = &cobra.Command{
	Use:   "download [version]",
	Short: "Download a Minecraft server jar",
	Long:  `Download a specific version of the Minecraft server jar file.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Implementation will go here
		return nil
	},
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List available Minecraft server versions",
	Long:  `List all available Minecraft server versions that can be downloaded.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Implementation will go here
		return nil
	},
}

var verifyCmd = &cobra.Command{
	Use:   "verify [jarfile]",
	Short: "Verify a jar file's checksum",
	Long:  `Verify the SHA256 checksum of a Minecraft server jar file.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Implementation will go here
		return nil
	},
}
