package envfile

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

const DefaultMinecraftEnvPath = "/etc/minecraft.env"

// EnvFile represents a parsed .env file
type EnvFile struct {
	vars map[string]string
	path string
}

// Load reads an env file from the given path
func Load(path string) (*EnvFile, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open env file: %w", err)
	}
	defer file.Close()

	vars := make(map[string]string)
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse KEY=value
		idx := strings.Index(line, "=")
		if idx == -1 {
			continue // Skip malformed lines
		}

		key := strings.TrimSpace(line[:idx])
		value := strings.TrimSpace(line[idx+1:])

		// Remove surrounding quotes if present
		if len(value) >= 2 {
			if (value[0] == '"' && value[len(value)-1] == '"') ||
				(value[0] == '\'' && value[len(value)-1] == '\'') {
				value = value[1 : len(value)-1]
			}
		}

		vars[key] = value
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading env file: %w", err)
	}

	return &EnvFile{
		vars: vars,
		path: path,
	}, nil
}

// LoadWithFallback tries to load from multiple paths, returning the first successful load
func LoadWithFallback(paths ...string) (*EnvFile, error) {
	var lastErr error
	for _, path := range paths {
		ef, err := Load(path)
		if err == nil {
			return ef, nil
		}
		lastErr = err
	}
	if lastErr != nil {
		return nil, lastErr
	}
	return nil, fmt.Errorf("no env file paths provided")
}

// Get returns a value by key
func (e *EnvFile) Get(key string) (string, bool) {
	val, ok := e.vars[key]
	return val, ok
}

// GetInt returns an integer value by key
func (e *EnvFile) GetInt(key string) (int, error) {
	val, ok := e.vars[key]
	if !ok {
		return 0, fmt.Errorf("key not found: %s", key)
	}
	return strconv.Atoi(val)
}

// MustGet returns the value or panics if the key is not found
func (e *EnvFile) MustGet(key string) string {
	val, ok := e.vars[key]
	if !ok {
		panic(fmt.Sprintf("required env key not found: %s", key))
	}
	return val
}

// Export sets all variables as environment variables
func (e *EnvFile) Export() error {
	for key, value := range e.vars {
		if err := os.Setenv(key, value); err != nil {
			return fmt.Errorf("failed to set env var %s: %w", key, err)
		}
	}
	return nil
}

// ExportIfNotSet sets environment variables only if they are not already set
func (e *EnvFile) ExportIfNotSet() error {
	for key, value := range e.vars {
		if os.Getenv(key) == "" {
			if err := os.Setenv(key, value); err != nil {
				return fmt.Errorf("failed to set env var %s: %w", key, err)
			}
		}
	}
	return nil
}

// Path returns the file path that was loaded
func (e *EnvFile) Path() string {
	return e.path
}

// Keys returns all keys in the env file
func (e *EnvFile) Keys() []string {
	keys := make([]string, 0, len(e.vars))
	for k := range e.vars {
		keys = append(keys, k)
	}
	return keys
}

// Len returns the number of variables in the env file
func (e *EnvFile) Len() int {
	return len(e.vars)
}
