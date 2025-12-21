package main

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/paul/minecraftctl/pkg/config"
	"github.com/paul/minecraftctl/pkg/envfile"
	"github.com/paul/minecraftctl/pkg/properties"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Server configuration management",
	Long:  "Manage Minecraft server.properties configuration files",
}

var configCheckCmd = &cobra.Command{
	Use:   "check <world>",
	Short: "Check RCON settings in server.properties",
	Long:  "Verify that RCON settings in server.properties match minecraft.env",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		worldName := args[0]
		cfg := config.Get()
		propsPath := filepath.Join(cfg.WorldsDir, worldName, "server.properties")

		// Load server.properties
		props, err := properties.Load(propsPath)
		if err != nil {
			return fmt.Errorf("failed to load server.properties: %w", err)
		}

		// Load minecraft.env
		envPath, _ := cmd.Flags().GetString("env-file")
		if envPath == "" {
			envPath = envfile.DefaultMinecraftEnvPath
		}

		ef, err := envfile.Load(envPath)
		if err != nil {
			return fmt.Errorf("failed to load minecraft.env: %w", err)
		}

		fmt.Printf("Checking RCON configuration for '%s'...\n\n", worldName)
		fmt.Printf("server.properties: %s\n", propsPath)
		fmt.Printf("minecraft.env: %s\n\n", ef.Path())

		hasErrors := false

		// Check enable-rcon
		enableRcon, ok := props.Get("enable-rcon")
		if !ok {
			fmt.Printf("  enable-rcon:   (missing) - should be true\n")
			hasErrors = true
		} else if enableRcon != "true" {
			fmt.Printf("  enable-rcon:   %s (MISMATCH - should be true)\n", enableRcon)
			hasErrors = true
		} else {
			fmt.Printf("  enable-rcon:   true (OK)\n")
		}

		// Check rcon.port
		envPort, _ := ef.GetInt("RCON_PORT")
		propsPort, err := props.GetInt("rcon.port")
		if err != nil {
			fmt.Printf("  rcon.port:     (missing or invalid)\n")
			hasErrors = true
		} else if envPort != 0 && propsPort != envPort {
			fmt.Printf("  rcon.port:     %d (MISMATCH - minecraft.env has %d)\n", propsPort, envPort)
			hasErrors = true
		} else {
			fmt.Printf("  rcon.port:     %d (OK)\n", propsPort)
		}

		// Check rcon.password
		envPassword, hasEnvPassword := ef.Get("RCON_PASSWORD")
		propsPassword, hasPropsPassword := props.Get("rcon.password")
		if !hasPropsPassword {
			fmt.Printf("  rcon.password: (missing)\n")
			hasErrors = true
		} else if hasEnvPassword && propsPassword != envPassword {
			fmt.Printf("  rcon.password: ******** (MISMATCH - does not match minecraft.env)\n")
			hasErrors = true
		} else {
			fmt.Printf("  rcon.password: ******** (OK)\n")
		}

		fmt.Println()
		if hasErrors {
			fmt.Println("Run 'minecraftctl config sync " + worldName + "' to fix these issues.")
			return fmt.Errorf("RCON configuration has issues")
		}

		fmt.Println("All RCON settings are correctly configured.")
		return nil
	},
}

var configSyncCmd = &cobra.Command{
	Use:   "sync <world>",
	Short: "Sync RCON settings from minecraft.env to server.properties",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		worldName := args[0]
		dryRun, _ := cmd.Flags().GetBool("dry-run")
		cfg := config.Get()
		propsPath := filepath.Join(cfg.WorldsDir, worldName, "server.properties")

		// Load server.properties
		props, err := properties.Load(propsPath)
		if err != nil {
			return fmt.Errorf("failed to load server.properties: %w", err)
		}

		// Load minecraft.env
		envPath, _ := cmd.Flags().GetString("env-file")
		if envPath == "" {
			envPath = envfile.DefaultMinecraftEnvPath
		}

		ef, err := envfile.Load(envPath)
		if err != nil {
			return fmt.Errorf("failed to load minecraft.env: %w", err)
		}

		fmt.Printf("Syncing RCON configuration for '%s'...\n\n", worldName)
		if dryRun {
			fmt.Println("(dry-run mode - no changes will be made)")
			fmt.Println()
		}

		changes := false

		// Sync enable-rcon
		enableRcon, _ := props.Get("enable-rcon")
		if enableRcon != "true" {
			fmt.Printf("  enable-rcon:   %s -> true\n", enableRcon)
			props.Set("enable-rcon", "true")
			changes = true
		}

		// Sync rcon.port
		if envPort, err := ef.GetInt("RCON_PORT"); err == nil {
			propsPort, _ := props.GetInt("rcon.port")
			if propsPort != envPort {
				fmt.Printf("  rcon.port:     %d -> %d\n", propsPort, envPort)
				props.SetInt("rcon.port", envPort)
				changes = true
			}
		}

		// Sync rcon.password
		if envPassword, ok := ef.Get("RCON_PASSWORD"); ok {
			propsPassword, _ := props.Get("rcon.password")
			if propsPassword != envPassword {
				fmt.Printf("  rcon.password: <old> -> <new>\n")
				props.Set("rcon.password", envPassword)
				changes = true
			}
		}

		if !changes {
			fmt.Println("\nNo changes needed - RCON settings already in sync.")
			return nil
		}

		if dryRun {
			fmt.Println("\nDry-run complete. Use --dry-run=false to apply changes.")
			return nil
		}

		if err := props.Save(); err != nil {
			return fmt.Errorf("failed to save server.properties: %w", err)
		}

		fmt.Println("\nChanges saved. Restart the server for changes to take effect:")
		fmt.Printf("  minecraftctl world restart %s\n", worldName)
		return nil
	},
}

var configGetCmd = &cobra.Command{
	Use:   "get <world> <key>",
	Short: "Get a property value from server.properties",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		worldName := args[0]
		key := args[1]
		cfg := config.Get()
		propsPath := filepath.Join(cfg.WorldsDir, worldName, "server.properties")

		props, err := properties.Load(propsPath)
		if err != nil {
			return fmt.Errorf("failed to load server.properties: %w", err)
		}

		val, ok := props.Get(key)
		if !ok {
			return fmt.Errorf("key not found: %s", key)
		}

		fmt.Println(val)
		return nil
	},
}

var configSetCmd = &cobra.Command{
	Use:   "set <world> <key> <value>",
	Short: "Set a property value in server.properties",
	Args:  cobra.ExactArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		worldName := args[0]
		key := args[1]
		value := args[2]
		cfg := config.Get()
		propsPath := filepath.Join(cfg.WorldsDir, worldName, "server.properties")

		props, err := properties.Load(propsPath)
		if err != nil {
			return fmt.Errorf("failed to load server.properties: %w", err)
		}

		oldVal, existed := props.Get(key)
		props.Set(key, value)

		if err := props.Save(); err != nil {
			return fmt.Errorf("failed to save server.properties: %w", err)
		}

		if existed {
			log.Info().Str("key", key).Str("old", oldVal).Str("new", value).Msg("updated property")
		} else {
			log.Info().Str("key", key).Str("value", value).Msg("added property")
		}

		fmt.Printf("Set %s=%s in server.properties\n", key, value)
		return nil
	},
}

var configListCmd = &cobra.Command{
	Use:   "list <world>",
	Short: "List all properties from server.properties",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		worldName := args[0]
		showPasswords, _ := cmd.Flags().GetBool("show-passwords")
		cfg := config.Get()
		propsPath := filepath.Join(cfg.WorldsDir, worldName, "server.properties")

		props, err := properties.Load(propsPath)
		if err != nil {
			return fmt.Errorf("failed to load server.properties: %w", err)
		}

		for _, key := range props.Keys() {
			val, _ := props.Get(key)

			// Mask password values unless --show-passwords is set
			if !showPasswords && strings.Contains(strings.ToLower(key), "password") {
				val = "********"
			}

			fmt.Printf("%s=%s\n", key, val)
		}

		return nil
	},
}

func init() {
	// Add env-file flag to commands that need it
	configCheckCmd.Flags().String("env-file", "", "path to minecraft.env file")
	configSyncCmd.Flags().String("env-file", "", "path to minecraft.env file")
	configSyncCmd.Flags().Bool("dry-run", false, "show what would change without modifying files")
	configListCmd.Flags().Bool("show-passwords", false, "show password values (default: masked)")

	configCmd.AddCommand(configCheckCmd)
	configCmd.AddCommand(configSyncCmd)
	configCmd.AddCommand(configGetCmd)
	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configListCmd)
}
