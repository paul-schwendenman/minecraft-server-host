package main

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/paul/minecraftctl/cmd/minecraftctl/root"
	"github.com/paul/minecraftctl/pkg/systemd"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func init() {
	// Register subcommands with root command
	rootCmd := root.GetRootCmd()
	rootCmd.AddCommand(WorldCmd)
	rootCmd.AddCommand(MapCmd)
	rootCmd.AddCommand(RconCmd)
	rootCmd.AddCommand(ConfigCmd)
	rootCmd.AddCommand(BackupCmd)
	rootCmd.AddCommand(jarCmd)
	rootCmd.AddCommand(StatusCmd)
}

func main() {
	// Set up pretty console logging for development
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	if err := root.GetRootCmd().Execute(); err != nil {
		if errors.Is(err, systemd.ErrPermission) {
			log.Error().Err(err).Msg("insufficient privileges")
			fmt.Fprintf(os.Stderr, "Managing system services requires root. Re-run with:\n  sudo %s\n", strings.Join(os.Args, " "))
		} else {
			log.Error().Err(err).Msg("command failed")
		}
		os.Exit(exitCode(err))
	}
}
