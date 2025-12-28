package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/paul/minecraftctl/pkg/rcon"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

// RconCmd is the parent command for RCON operations
var RconCmd = &cobra.Command{
	Use:   "rcon",
	Short: "RCON commands",
	Long:  "Execute RCON commands on the Minecraft server",
}

var rconStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check server status via RCON",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := rcon.NewClient()
		if err != nil {
			return fmt.Errorf("failed to create RCON client: %w", err)
		}
		defer client.Close()

		status, err := client.Status()
		if err != nil {
			return err
		}

		fmt.Println(status)
		return nil
	},
}

var rconSendCmd = &cobra.Command{
	Use:   "send <command>",
	Short: "Send a command via RCON",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := rcon.NewClient()
		if err != nil {
			return fmt.Errorf("failed to create RCON client: %w", err)
		}
		defer client.Close()

		command := args[0]
		for i := 1; i < len(args); i++ {
			command += " " + args[i]
		}

		response, err := client.Send(command)
		if err != nil {
			return err
		}

		fmt.Println(response)
		return nil
	},
}

var rconExecCmd = &cobra.Command{
	Use:   "exec <file>",
	Short: "Execute commands from a file via RCON",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		filePath := args[0]
		data, err := os.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("failed to read file: %w", err)
		}

		client, err := rcon.NewClient()
		if err != nil {
			return fmt.Errorf("failed to create RCON client: %w", err)
		}
		defer client.Close()

		// Execute each line as a command
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}

			log.Info().Str("command", line).Msg("executing RCON command")
			response, err := client.Send(line)
			if err != nil {
				log.Error().Err(err).Str("command", line).Msg("command failed")
				continue
			}
			if response != "" {
				fmt.Println(response)
			}
		}

		return nil
	},
}

func init() {
	RconCmd.AddCommand(rconStatusCmd)
	RconCmd.AddCommand(rconSendCmd)
	RconCmd.AddCommand(rconExecCmd)
}
