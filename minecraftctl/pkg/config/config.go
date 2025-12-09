package config

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

const (
	DefaultWorldsDir = "/srv/minecraft-server"
	DefaultMapsDir   = "/srv/minecraft-server/maps"
	DefaultJarsDir   = "/opt/minecraft/jars"
	DefaultLockFile  = "/tmp/minecraft-map-build.lock"
	DefaultRconHost  = "127.0.0.1"
	DefaultRconPort  = 25575
)

// GlobalConfig holds the application-wide configuration
type GlobalConfig struct {
	WorldsDir string
	MapsDir   string
	JarsDir   string
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
	Defaults MapDefaults     `yaml:"defaults" mapstructure:"defaults"`
	Maps     []MapDefinition `yaml:"maps" mapstructure:"maps"`
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
	Name         string     `yaml:"name" mapstructure:"name"`
	Dimension    string     `yaml:"dimension" mapstructure:"dimension"`
	OutputSubdir string     `yaml:"output_subdir" mapstructure:"output_subdir"`
	Zoomout      *int       `yaml:"zoomout,omitempty" mapstructure:"zoomout"`
	Zoomin       *int       `yaml:"zoomin,omitempty" mapstructure:"zoomin"`
	Options      MapOptions `yaml:"options,omitempty" mapstructure:"options"`
	Ranges       []MapRange `yaml:"ranges,omitempty" mapstructure:"ranges"`
}

// MapOptions holds optional rendering options
type MapOptions struct {
	TopY    *int        `yaml:"topY,omitempty" mapstructure:"topY"`
	BottomY *int        `yaml:"bottomY,omitempty" mapstructure:"bottomY"`
	GndXray *bool       `yaml:"gndxray,omitempty" mapstructure:"gndxray"`
	Shadows interface{} `yaml:"shadows,omitempty" mapstructure:"shadows"` // can be bool or string
	Night   *bool       `yaml:"night,omitempty" mapstructure:"night"`
	Players *bool       `yaml:"players,omitempty" mapstructure:"players"`
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
	viper.SetDefault("jars_dir", DefaultJarsDir)
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

	// Enable automatic environment variable reading
	viper.AutomaticEnv()

	// Set env prefix for MINECRAFT_ prefixed vars
	viper.SetEnvPrefix("MINECRAFT")

	// Bind environment variables - support both prefixed and non-prefixed variants
	// For nested keys like "rcon.password", Viper will look for MINECRAFT_RCON_PASSWORD
	// by default, but we also want to support RCON_PASSWORD directly
	viper.BindEnv("rcon.password", "RCON_PASSWORD")
	viper.BindEnv("rcon.host", "RCON_HOST")
	viper.BindEnv("rcon.port", "RCON_PORT")
	viper.BindEnv("worlds_dir", "WORLDS_DIR")
	viper.BindEnv("maps_dir", "MAPS_DIR")
	viper.BindEnv("jars_dir", "MINECRAFT_JARS_DIR")
	viper.BindEnv("lock_file", "LOCK_FILE")

	// Load global config
	globalConfig = &GlobalConfig{
		WorldsDir: viper.GetString("worlds_dir"),
		MapsDir:   viper.GetString("maps_dir"),
		JarsDir:   viper.GetString("jars_dir"),
		LockFile:  viper.GetString("lock_file"),
		Rcon: RconConfig{
			Host:     viper.GetString("rcon.host"),
			Port:     viper.GetInt("rcon.port"),
			Password: viper.GetString("rcon.password"),
		},
	}

	// Check environment variables directly (overrides Viper values)
	// This supports both RCON_PASSWORD and MINECRAFT_RCON_PASSWORD
	// Priority: RCON_PASSWORD > MINECRAFT_RCON_PASSWORD > config file > defaults
	if pwd := os.Getenv("RCON_PASSWORD"); pwd != "" {
		globalConfig.Rcon.Password = pwd
	} else if pwd := os.Getenv("MINECRAFT_RCON_PASSWORD"); pwd != "" {
		globalConfig.Rcon.Password = pwd
	}

	// Similar direct checks for other RCON settings
	if host := os.Getenv("RCON_HOST"); host != "" {
		globalConfig.Rcon.Host = host
	} else if host := os.Getenv("MINECRAFT_RCON_HOST"); host != "" {
		globalConfig.Rcon.Host = host
	}

	if portStr := os.Getenv("RCON_PORT"); portStr != "" {
		if port, err := strconv.Atoi(portStr); err == nil {
			globalConfig.Rcon.Port = port
		}
	} else if portStr := os.Getenv("MINECRAFT_RCON_PORT"); portStr != "" {
		if port, err := strconv.Atoi(portStr); err == nil {
			globalConfig.Rcon.Port = port
		}
	}

	// Expand environment variables in paths and password
	globalConfig.WorldsDir = expandEnv(globalConfig.WorldsDir)
	globalConfig.MapsDir = expandEnv(globalConfig.MapsDir)
	globalConfig.JarsDir = expandEnv(globalConfig.JarsDir)
	globalConfig.LockFile = expandEnv(globalConfig.LockFile)
	globalConfig.Rcon.Password = expandEnv(globalConfig.Rcon.Password)

	return nil
}

// Get returns the global configuration
// It always reads from Viper to ensure flags override config
func Get() *GlobalConfig {
	if globalConfig == nil {
		// Return defaults if not initialized
		return &GlobalConfig{
			WorldsDir: DefaultWorldsDir,
			MapsDir:   DefaultMapsDir,
			JarsDir:   DefaultJarsDir,
			LockFile:  DefaultLockFile,
			Rcon: RconConfig{
				Host: DefaultRconHost,
				Port: DefaultRconPort,
			},
		}
	}

	// Re-read from Viper to pick up any flags that were bound after Init()
	// This allows CLI flags to override the initial config
	cfg := &GlobalConfig{
		WorldsDir: viper.GetString("worlds_dir"),
		MapsDir:   viper.GetString("maps_dir"),
		JarsDir:   viper.GetString("jars_dir"),
		LockFile:  viper.GetString("lock_file"),
		Rcon: RconConfig{
			Host:     viper.GetString("rcon.host"),
			Port:     viper.GetInt("rcon.port"),
			Password: viper.GetString("rcon.password"),
		},
	}

	// Expand environment variables in paths and password
	cfg.WorldsDir = expandEnv(cfg.WorldsDir)
	cfg.MapsDir = expandEnv(cfg.MapsDir)
	cfg.JarsDir = expandEnv(cfg.JarsDir)
	cfg.LockFile = expandEnv(cfg.LockFile)
	cfg.Rcon.Password = expandEnv(cfg.Rcon.Password)

	return cfg
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

// SaveMapConfig saves a MapConfig to a YAML file
// If the path ends with ".yml" or ".yaml", it's treated as a full file path.
// Otherwise, it's treated as a directory and "map-config.yml" is appended.
func SaveMapConfig(path string, mapConfig *MapConfig) error {
	var configPath string
	baseName := strings.ToLower(filepath.Base(path))
	if strings.HasSuffix(baseName, ".yml") || strings.HasSuffix(baseName, ".yaml") {
		// Path is already a full file path
		configPath = path
	} else {
		// Path is a directory, append map-config.yml
		configPath = filepath.Join(path, "map-config.yml")
	}

	// Create parent directory if needed
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Marshal to YAML
	data, err := yaml.Marshal(mapConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal map-config.yml: %w", err)
	}

	// Write file
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write map-config.yml: %w", err)
	}

	return nil
}

// capitalizeFieldName converts a field name from YAML (lowercase) to Go struct field name
func capitalizeFieldName(name string) string {
	if len(name) == 0 {
		return name
	}
	return strings.ToUpper(name[:1]) + name[1:]
}

// GetConfigField retrieves a field value from MapConfig using dot notation path
// Examples: "defaults.zoomout", "maps[0].name", "maps[0].options.shadows"
func GetConfigField(path string, mapConfig *MapConfig) (interface{}, error) {
	if mapConfig == nil {
		return nil, fmt.Errorf("mapConfig cannot be nil")
	}
	if path == "" {
		return mapConfig, nil
	}

	parts := strings.Split(path, ".")
	current := reflect.ValueOf(mapConfig).Elem()

	for i, part := range parts {
		// Handle array indexing like maps[0]
		if idx := strings.Index(part, "["); idx != -1 {
			fieldName := part[:idx]
			// Extract index
			re := regexp.MustCompile(`\[(\d+)\]`)
			matches := re.FindStringSubmatch(part)
			if len(matches) < 2 {
				return nil, fmt.Errorf("invalid array index in path: %s", part)
			}
			index, err := strconv.Atoi(matches[1])
			if err != nil {
				return nil, fmt.Errorf("invalid array index: %w", err)
			}

			// Get field
			field := current.FieldByName(capitalizeFieldName(fieldName))
			if !field.IsValid() {
				return nil, fmt.Errorf("field not found: %s", fieldName)
			}

			// Navigate to array element
			if field.Kind() != reflect.Slice {
				return nil, fmt.Errorf("field is not a slice: %s", fieldName)
			}
			if index < 0 || index >= field.Len() {
				return nil, fmt.Errorf("array index out of range: %d (length: %d)", index, field.Len())
			}
			current = field.Index(index)
		} else {
			// Regular field access
			field := current.FieldByName(capitalizeFieldName(part))
			if !field.IsValid() {
				return nil, fmt.Errorf("field not found: %s", part)
			}
			current = field
		}

		// Follow pointer if needed
		if current.Kind() == reflect.Ptr {
			if current.IsNil() {
				return nil, fmt.Errorf("field is nil: %s", strings.Join(parts[:i+1], "."))
			}
			current = current.Elem()
		}
	}

	return current.Interface(), nil
}

// SetConfigField sets a field value in MapConfig using dot notation path
// Examples: "defaults.zoomout", "maps[0].options.shadows", "maps[0].ranges[0].radius"
func SetConfigField(path string, value interface{}, mapConfig *MapConfig) error {
	if mapConfig == nil {
		return fmt.Errorf("mapConfig cannot be nil")
	}
	if path == "" {
		return fmt.Errorf("path cannot be empty")
	}

	parts := strings.Split(path, ".")
	current := reflect.ValueOf(mapConfig).Elem()
	var lastPart string
	var lastIndex int = -1

	for i, part := range parts {
		isLast := i == len(parts)-1

		// Handle array indexing like maps[0]
		if idx := strings.Index(part, "["); idx != -1 {
			fieldName := part[:idx]
			// Extract index
			re := regexp.MustCompile(`\[(\d+)\]`)
			matches := re.FindStringSubmatch(part)
			if len(matches) < 2 {
				return fmt.Errorf("invalid array index in path: %s", part)
			}
			index, err := strconv.Atoi(matches[1])
			if err != nil {
				return fmt.Errorf("invalid array index: %w", err)
			}

			// Get field
			field := current.FieldByName(capitalizeFieldName(fieldName))
			if !field.IsValid() {
				return fmt.Errorf("field not found: %s", fieldName)
			}

			if !isLast {
				// Navigate to array element
				if field.Kind() != reflect.Slice {
					return fmt.Errorf("field is not a slice: %s", fieldName)
				}
				if index < 0 || index >= field.Len() {
					return fmt.Errorf("array index out of range: %d (length: %d)", index, field.Len())
				}
				current = field.Index(index)
				// Follow pointer if needed
				if current.Kind() == reflect.Ptr {
					if current.IsNil() {
						return fmt.Errorf("field is nil: %s", strings.Join(parts[:i+1], "."))
					}
					current = current.Elem()
				}
			} else {
				// This is the last part, we need to set a value in the array
				if field.Kind() != reflect.Slice {
					return fmt.Errorf("field is not a slice: %s", fieldName)
				}
				if index < 0 || index >= field.Len() {
					return fmt.Errorf("array index out of range: %d (length: %d)", index, field.Len())
				}
				elem := field.Index(index)
				if elem.Kind() == reflect.Ptr {
					if elem.IsNil() {
						return fmt.Errorf("array element is nil at index %d", index)
					}
					elem = elem.Elem()
				}
				current = elem
				lastPart = ""
				lastIndex = index
			}
		} else {
			if !isLast {
				// Regular field access (not last part)
				field := current.FieldByName(capitalizeFieldName(part))
				if !field.IsValid() {
					return fmt.Errorf("field not found: %s", part)
				}
				current = field
				// Follow pointer if needed
				if current.Kind() == reflect.Ptr {
					if current.IsNil() {
						return fmt.Errorf("field is nil: %s", strings.Join(parts[:i+1], "."))
					}
					current = current.Elem()
				}
			} else {
				// Last part - we'll set this field
				lastPart = part
			}
		}
	}

	// Set the value
	if lastPart != "" {
		field := current.FieldByName(capitalizeFieldName(lastPart))
		if !field.IsValid() {
			return fmt.Errorf("field not found: %s", lastPart)
		}
		if !field.CanSet() {
			return fmt.Errorf("field cannot be set: %s", lastPart)
		}

		val := reflect.ValueOf(value)
		// Convert value to appropriate type
		if field.Kind() != val.Kind() {
			// Try to convert
			if val.CanConvert(field.Type()) {
				val = val.Convert(field.Type())
			} else {
				return fmt.Errorf("cannot convert value type %v to field type %v", val.Type(), field.Type())
			}
		}

		// Handle pointer fields
		if field.Kind() == reflect.Ptr {
			// Create a new pointer of the correct type
			ptr := reflect.New(field.Type().Elem())
			if val.CanConvert(field.Type().Elem()) {
				ptr.Elem().Set(val.Convert(field.Type().Elem()))
			} else {
				return fmt.Errorf("cannot convert value type %v to pointer element type %v", val.Type(), field.Type().Elem())
			}
			field.Set(ptr)
		} else {
			field.Set(val)
		}
	} else if lastIndex >= 0 {
		// Setting an entire array element - this would require more complex logic
		// For now, we don't support setting entire array elements
		return fmt.Errorf("setting entire array elements is not supported")
	}

	return nil
}

// ValidateMapConfig validates a MapConfig and returns a list of error messages
func ValidateMapConfig(mapConfig *MapConfig) []string {
	var errs []string

	// Validate defaults
	if mapConfig.Defaults.Zoomout < 0 {
		errs = append(errs, "defaults.zoomout must be >= 0")
	}
	if mapConfig.Defaults.Zoomin < 0 {
		errs = append(errs, "defaults.zoomin must be >= 0")
	}
	if mapConfig.Defaults.ImageFormat != "" {
		validFormats := map[string]bool{"jpeg": true, "png": true, "webp": true}
		if !validFormats[strings.ToLower(mapConfig.Defaults.ImageFormat)] {
			errs = append(errs, fmt.Sprintf("defaults.imageformat must be one of: jpeg, png, webp (got: %s)", mapConfig.Defaults.ImageFormat))
		}
	}
	if mapConfig.Defaults.ChunkProcessors < 1 {
		errs = append(errs, "defaults.chunkprocessors must be >= 1")
	}

	// Validate maps
	if len(mapConfig.Maps) == 0 {
		errs = append(errs, "at least one map definition is required")
	}

	validDimensions := map[string]bool{"overworld": true, "nether": true, "end": true}
	validShadowValues := map[string]bool{"true": true, "false": true, "2d": true, "3d": true, "3do": true}

	for i, m := range mapConfig.Maps {
		if m.Name == "" {
			errs = append(errs, fmt.Sprintf("maps[%d].name is required", i))
		}
		if m.Dimension == "" {
			errs = append(errs, fmt.Sprintf("maps[%d].dimension is required", i))
		} else if !validDimensions[m.Dimension] {
			errs = append(errs, fmt.Sprintf("maps[%d].dimension must be one of: overworld, nether, end (got: %s)", i, m.Dimension))
		}

		// Validate shadow value if set
		if m.Options.Shadows != nil {
			var shadowStr string
			switch v := m.Options.Shadows.(type) {
			case string:
				shadowStr = v
			case bool:
				if v {
					shadowStr = "true"
				} else {
					shadowStr = "false"
				}
			}
			if shadowStr != "" && !validShadowValues[strings.ToLower(shadowStr)] {
				errs = append(errs, fmt.Sprintf("maps[%d].options.shadows must be one of: true, false, 2d, 3d, 3do (got: %v)", i, m.Options.Shadows))
			}
		}

		// Validate zoom levels
		if m.Zoomout != nil && *m.Zoomout < 0 {
			errs = append(errs, fmt.Sprintf("maps[%d].zoomout must be >= 0", i))
		}
		if m.Zoomin != nil && *m.Zoomin < 0 {
			errs = append(errs, fmt.Sprintf("maps[%d].zoomin must be >= 0", i))
		}

		// Validate ranges
		for j, r := range m.Ranges {
			if r.Name == "" {
				errs = append(errs, fmt.Sprintf("maps[%d].ranges[%d].name is required", i, j))
			}
			if r.Radius <= 0 {
				errs = append(errs, fmt.Sprintf("maps[%d].ranges[%d].radius must be > 0", i, j))
			}
			if r.Zoomout != nil && *r.Zoomout < 0 {
				errs = append(errs, fmt.Sprintf("maps[%d].ranges[%d].zoomout must be >= 0", i, j))
			}
			if r.Zoomin != nil && *r.Zoomin < 0 {
				errs = append(errs, fmt.Sprintf("maps[%d].ranges[%d].zoomin must be >= 0", i, j))
			}
		}
	}

	return errs
}

// expandEnv expands environment variables in a string
func expandEnv(s string) string {
	return os.ExpandEnv(s)
}
