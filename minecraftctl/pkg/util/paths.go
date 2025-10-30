package util

import (
	"os"
	"path/filepath"
)

// ExpandPath expands tilde and environment variables in a path
func ExpandPath(path string) string {
	if path == "" {
		return path
	}

	// Expand ~
	if path[0] == '~' {
		home, err := os.UserHomeDir()
		if err == nil {
			path = filepath.Join(home, path[1:])
		}
	}

	// Expand environment variables
	path = os.ExpandEnv(path)
	return path
}

// AbsPath returns an absolute path, expanding tilde and env vars first
func AbsPath(path string) (string, error) {
	expanded := ExpandPath(path)
	return filepath.Abs(expanded)
}

