package root

import (
	"fmt"
	"strings"

	"github.com/paul/minecraftctl/internal/version"
	"github.com/paul/minecraftctl/pkg/config"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile    string
	verbose    bool
	worldsDir  string
	mapsDir    string
	rconHost   string
	rconPort   int
	rconPass   string
	rootCmd    *cobra.Command
)

func init() {
	rootCmd = newRootCmd()
}

// NewRootCmd creates the root command for minecraftctl (deprecated, use GetRootCmd)
func NewRootCmd() *cobra.Command {
	return newRootCmd()
}

// GetRootCmd returns the root command instance
func GetRootCmd() *cobra.Command {
	return rootCmd
}

// newRootCmd creates the root command for minecraftctl
func newRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "minecraftctl",
		Short: "Minecraft server management tool",
		Long:  "A CLI tool for managing Minecraft worlds, maps, and RCON commands",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// Set up logging level
			if verbose {
				zerolog.SetGlobalLevel(zerolog.DebugLevel)
			} else {
				zerolog.SetGlobalLevel(zerolog.InfoLevel)
			}

			// Initialize config
			if err := config.Init(cfgFile); err != nil {
				return fmt.Errorf("failed to initialize config: %w", err)
			}

			// Bind flags to viper
			viper.BindPFlag("worlds_dir", cmd.Flags().Lookup("worlds-dir"))
			viper.BindPFlag("maps_dir", cmd.Flags().Lookup("maps-dir"))
			viper.BindPFlag("rcon.host", cmd.Flags().Lookup("rcon-host"))
			viper.BindPFlag("rcon.port", cmd.Flags().Lookup("rcon-port"))
			viper.BindPFlag("rcon.password", cmd.Flags().Lookup("rcon-password"))

			return nil
		},
		Version: version.Version,
	}

	// Global flags
	cmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is /etc/minecraftctl.yml or ~/.config/minecraftctl.yml)")
	cmd.PersistentFlags().StringVar(&worldsDir, "worlds-dir", "", "directory containing Minecraft worlds")
	cmd.PersistentFlags().StringVar(&mapsDir, "maps-dir", "", "directory for map output")
	cmd.PersistentFlags().StringVar(&rconHost, "rcon-host", "", "RCON host (default: 127.0.0.1)")
	cmd.PersistentFlags().IntVar(&rconPort, "rcon-port", 0, "RCON port")
	cmd.PersistentFlags().StringVar(&rconPass, "rcon-password", "", "RCON password")
	cmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")

	// Environment variable bindings
	viper.SetEnvPrefix("MINECRAFT")
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))

	// Bind env vars
	viper.BindEnv("worlds_dir", "MINECRAFT_WORLDS_DIR")
	viper.BindEnv("maps_dir", "MINECRAFT_MAPS_DIR")
	viper.BindEnv("rcon.host", "MINECRAFT_RCON_HOST")
	viper.BindEnv("rcon.port", "MINECRAFT_RCON_PORT")
	viper.BindEnv("rcon.password", "MINECRAFT_RCON_PASSWORD")

	// Note: Subcommands are added in their respective files via init()

	return cmd
}

