// Package commands provides top-level command definitions for minecraftctl.
// These commands are defined here so they can be imported by both the main
// CLI binary and the gendocs tool.
package commands

import (
	"github.com/spf13/cobra"
)

// WorldCmd is the parent command for world management
var WorldCmd = &cobra.Command{
	Use:   "world",
	Short: "Manage Minecraft worlds",
	Long:  "Commands for listing and inspecting Minecraft worlds",
}

// MapCmd is the parent command for map management
var MapCmd = &cobra.Command{
	Use:   "map",
	Short: "Manage maps",
	Long:  "Commands for building and managing Minecraft world maps",
}

// RconCmd is the parent command for RCON operations
var RconCmd = &cobra.Command{
	Use:   "rcon",
	Short: "RCON commands",
	Long:  "Execute RCON commands on the Minecraft server",
}

// ConfigCmd is the parent command for server configuration management
var ConfigCmd = &cobra.Command{
	Use:   "config",
	Short: "Server configuration management",
	Long:  "Manage Minecraft server.properties configuration files",
}

// BackupCmd is the parent command for backup management
var BackupCmd = &cobra.Command{
	Use:   "backup",
	Short: "Manage world backups",
	Long:  "Commands for managing world backups using restic",
}
