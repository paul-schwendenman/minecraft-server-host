package main

import (
	"os"

	"github.com/paul/minecraftctl/cmd/minecraftctl/root"
	"github.com/paul/minecraftctl/pkg/jars"
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
	rootCmd.AddCommand(jars.JarCmd)
}

func main() {
	// Set up pretty console logging for development
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	if err := root.GetRootCmd().Execute(); err != nil {
		log.Error().Err(err).Msg("command failed")
		os.Exit(1)
	}
}
