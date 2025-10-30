package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

const (
	DefaultWorldsDir = "/srv/minecraft-server"
	DefaultMapsDir   = "/srv/minecraft-server/maps"
	DefaultLockFile  = "/tmp/minecraft-map-build.lock"
	DefaultRconHost  = "127.0.0.1"
	DefaultRconPort  = 25575
)

// GlobalConfig holds the application-wide configuration
type GlobalConfig struct {
	WorldsDir string
	MapsDir   string
	LockFile  string
	Rcon      RconConfig
}

// RconConfig holds RCON connection settings
type RconConfig struct {
	Host     string
	Port     int
	Password string
}

// MapConfig represents a per-world map-config.yml file
type MapConfig struct {
	Defaults MapDefaults            `yaml:"defaults" mapstructure:"defaults"`
	Maps     []MapDefinition        `yaml:"maps" mapstructure:"maps"`
}

// MapDefaults holds default settings for all maps
type MapDefaults struct {
	Zoomout         int    `yaml:"zoomout" mapstructure:"zoomout"`
	Zoomin          int    `yaml:"zoomin" mapstructure:"zoomin"`
	ImageFormat     string `yaml:"imageformat" mapstructure:"imageformat"`
	ChunkProcessors int    `yaml:"chunkprocessors" mapstructure:"chunkprocessors"`
}

// MapDefinition defines a single map to render
type MapDefinition struct {
	Name        string        `yaml:"name" mapstructure:"name"`
	Dimension   string        `yaml:"dimension" mapstructure:"dimension"`
	OutputSubdir string       `yaml:"output_subdir" mapstructure:"output_subdir"`
	Zoomout     *int          `yaml:"zoomout,omitempty" mapstructure:"zoomout"`
	Zoomin      *int          `yaml:"zoomin,omitempty" mapstructure:"zoomin"`
	Options     MapOptions    `yaml:"options,omitempty" mapstructure:"options"`
	Ranges      []MapRange    `yaml:"ranges,omitempty" mapstructure:"ranges"`
}

// MapOptions holds optional rendering options
type MapOptions struct {
	TopY     *int    `yaml:"topY,omitempty" mapstructure:"topY"`
	BottomY  *int    `yaml:"bottomY,omitempty" mapstructure:"bottomY"`
	GndXray  *bool   `yaml:"gndxray,omitempty" mapstructure:"gndxray"`
	Shadows  interface{} `yaml:"shadows,omitempty" mapstructure:"shadows"` // can be bool or string
	Night    *bool   `yaml:"night,omitempty" mapstructure:"night"`
	Players  *bool   `yaml:"players,omitempty" mapstructure:"players"`
}

// MapRange defines a focused render area
type MapRange struct {
	Name    string `yaml:"name" mapstructure:"name"`
	Center  [2]int `yaml:"center" mapstructure:"center"`
	Radius  int    `yaml:"radius" mapstructure:"radius"`
	Zoomout *int   `yaml:"zoomout,omitempty" mapstructure:"zoomout"`
	Zoomin  *int   `yaml:"zoomin,omitempty" mapstructure:"zoomin"`
}

var globalConfig *GlobalConfig

// Init initializes the configuration system
func Init(cfgFile string) error {
	// Set defaults
	viper.SetDefault("worlds_dir", DefaultWorldsDir)
	viper.SetDefault("maps_dir", DefaultMapsDir)
	viper.SetDefault("lock_file", DefaultLockFile)
	viper.SetDefault("rcon.host", DefaultRconHost)
	viper.SetDefault("rcon.port", DefaultRconPort)

	// If config file is explicitly set, use it
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		// Search in standard locations
		viper.SetConfigName("minecraftctl")
		viper.SetConfigType("yaml")
		viper.AddConfigPath("/etc")
		viper.AddConfigPath("$HOME/.config")
	}

	// Read config file (optional)
	if err := viper.ReadInConfig(); err != nil {
		// Config file not found is okay, we have defaults
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return fmt.Errorf("error reading config file: %w", err)
		}
	}

	// Load global config
	globalConfig = &GlobalConfig{
		WorldsDir: viper.GetString("worlds_dir"),
		MapsDir:   viper.GetString("maps_dir"),
		LockFile:  viper.GetString("lock_file"),
		Rcon: RconConfig{
			Host:     viper.GetString("rcon.host"),
			Port:     viper.GetInt("rcon.port"),
			Password: viper.GetString("rcon.password"),
		},
	}

	// Expand environment variables in paths
	globalConfig.WorldsDir = expandEnv(globalConfig.WorldsDir)
	globalConfig.MapsDir = expandEnv(globalConfig.MapsDir)
	globalConfig.LockFile = expandEnv(globalConfig.LockFile)

	return nil
}

// Get returns the global configuration
func Get() *GlobalConfig {
	if globalConfig == nil {
		// Return defaults if not initialized
		return &GlobalConfig{
			WorldsDir: DefaultWorldsDir,
			MapsDir:   DefaultMapsDir,
			LockFile:  DefaultLockFile,
			Rcon: RconConfig{
				Host: DefaultRconHost,
				Port: DefaultRconPort,
			},
		}
	}
	return globalConfig
}

// LoadMapConfig loads a per-world map-config.yml file
func LoadMapConfig(worldPath string) (*MapConfig, error) {
	configPath := filepath.Join(worldPath, "map-config.yml")
	
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("map-config.yml not found at %s", configPath)
	}

	v := viper.New()
	v.SetConfigFile(configPath)
	v.SetConfigType("yaml")

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read map-config.yml: %w", err)
	}

	var mapConfig MapConfig
	if err := v.Unmarshal(&mapConfig); err != nil {
		return nil, fmt.Errorf("failed to unmarshal map-config.yml: %w", err)
	}

	// Set defaults if not specified
	if mapConfig.Defaults.Zoomout == 0 {
		mapConfig.Defaults.Zoomout = 2
	}
	if mapConfig.Defaults.Zoomin == 0 {
		mapConfig.Defaults.Zoomin = 1
	}
	if mapConfig.Defaults.ImageFormat == "" {
		mapConfig.Defaults.ImageFormat = "jpeg"
	}
	if mapConfig.Defaults.ChunkProcessors == 0 {
		mapConfig.Defaults.ChunkProcessors = 2
	}

	// Merge defaults with per-map settings
	for i := range mapConfig.Maps {
		if mapConfig.Maps[i].Zoomout == nil {
			zoomout := mapConfig.Defaults.Zoomout
			mapConfig.Maps[i].Zoomout = &zoomout
		}
		if mapConfig.Maps[i].Zoomin == nil {
			zoomin := mapConfig.Defaults.Zoomin
			mapConfig.Maps[i].Zoomin = &zoomin
		}
		if mapConfig.Maps[i].OutputSubdir == "" {
			mapConfig.Maps[i].OutputSubdir = mapConfig.Maps[i].Name
		}
	}

	return &mapConfig, nil
}

// expandEnv expands environment variables in a string
func expandEnv(s string) string {
	return os.ExpandEnv(s)
}

