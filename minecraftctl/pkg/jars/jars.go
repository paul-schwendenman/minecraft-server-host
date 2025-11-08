package jars

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
)

const (
	checksumsFileName = "checksums.txt"
	jarPrefix         = "minecraft_server_"
	jarSuffix         = ".jar"
)

// JarInfo contains information about a Minecraft server JAR
type JarInfo struct {
	Version      string
	Path         string
	Size         int64
	Checksum     string
	InstalledAt  time.Time
}

// ListJars returns a list of all installed JARs in the jars directory
func ListJars(jarsDir string) ([]JarInfo, error) {
	entries, err := os.ReadDir(jarsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read jars directory: %w", err)
	}

	var jars []JarInfo
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		// Check if it matches the pattern minecraft_server_*.jar
		if !strings.HasPrefix(name, jarPrefix) || !strings.HasSuffix(name, jarSuffix) {
			continue
		}

		// Extract version from filename: minecraft_server_<version>.jar
		version := strings.TrimPrefix(name, jarPrefix)
		version = strings.TrimSuffix(version, jarSuffix)

		info, err := GetJarInfo(version, jarsDir)
		if err != nil {
			log.Warn().Err(err).Str("version", version).Msg("failed to get jar info")
			continue
		}

		jars = append(jars, *info)
	}

	return jars, nil
}

// GetJarInfo retrieves information about a specific JAR
func GetJarInfo(version, jarsDir string) (*JarInfo, error) {
	jarPath := filepath.Join(jarsDir, fmt.Sprintf("%s%s%s", jarPrefix, version, jarSuffix))

	fileInfo, err := os.Stat(jarPath)
	if err != nil {
		return nil, fmt.Errorf("jar not found: %s", jarPath)
	}

	// Compute checksum
	checksum, err := computeSHA256(jarPath)
	if err != nil {
		return nil, fmt.Errorf("failed to compute checksum: %w", err)
	}

	return &JarInfo{
		Version:     version,
		Path:        jarPath,
		Size:        fileInfo.Size(),
		Checksum:    checksum,
		InstalledAt: fileInfo.ModTime(),
	}, nil
}

// DownloadJar downloads a Minecraft server JAR from a URL and verifies its checksum
func DownloadJar(version, url, jarsDir string, expectedSHA256 string) error {
	jarPath := filepath.Join(jarsDir, fmt.Sprintf("%s%s%s", jarPrefix, version, jarSuffix))

	// Check if JAR already exists
	if _, err := os.Stat(jarPath); err == nil {
		return fmt.Errorf("jar already exists: %s", jarPath)
	}

	// Create temporary file for download
	tmpFile, err := os.CreateTemp(jarsDir, fmt.Sprintf("%s%s-*.tmp", jarPrefix, version))
	if err != nil {
		return fmt.Errorf("failed to create temporary file: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer func() {
		tmpFile.Close()
		os.Remove(tmpPath)
	}()

	// Download the file
	log.Info().Str("version", version).Str("url", url).Msg("downloading jar")
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to download jar: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with status %d", resp.StatusCode)
	}

	// Write to temporary file
	_, err = io.Copy(tmpFile, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to write jar file: %w", err)
	}
	tmpFile.Close()

	// Compute checksum
	actualSHA256, err := computeSHA256(tmpPath)
	if err != nil {
		return fmt.Errorf("failed to compute checksum: %w", err)
	}

	// Verify checksum if expected is provided
	if expectedSHA256 != "" {
		if actualSHA256 != expectedSHA256 {
			return fmt.Errorf("checksum mismatch: expected %s, got %s", expectedSHA256, actualSHA256)
		}
		log.Info().Str("version", version).Msg("checksum verified")
	}

	// Move temporary file to final location
	if err := os.Rename(tmpPath, jarPath); err != nil {
		return fmt.Errorf("failed to move jar to final location: %w", err)
	}

	// Update checksums.txt
	if err := SaveChecksum(version, actualSHA256, jarsDir); err != nil {
		log.Warn().Err(err).Msg("failed to update checksums.txt")
		// Don't fail the download if checksum file update fails
	}

	log.Info().Str("version", version).Str("path", jarPath).Msg("jar downloaded successfully")
	return nil
}

// VerifyJar verifies a JAR's checksum against the checksums.txt file
func VerifyJar(version, jarsDir string) error {
	jarPath := filepath.Join(jarsDir, fmt.Sprintf("%s%s%s", jarPrefix, version, jarSuffix))

	// Check if JAR exists
	if _, err := os.Stat(jarPath); err != nil {
		return fmt.Errorf("jar not found: %s", jarPath)
	}

	// Load checksums
	checksums, err := LoadChecksums(jarsDir)
	if err != nil {
		return fmt.Errorf("failed to load checksums: %w", err)
	}

	// Get expected checksum
	jarFileName := filepath.Base(jarPath)
	expectedSHA256, ok := checksums[jarFileName]
	if !ok {
		return fmt.Errorf("checksum not found in checksums.txt for %s", jarFileName)
	}

	// Compute actual checksum
	actualSHA256, err := computeSHA256(jarPath)
	if err != nil {
		return fmt.Errorf("failed to compute checksum: %w", err)
	}

	// Verify
	if actualSHA256 != expectedSHA256 {
		return fmt.Errorf("checksum mismatch: expected %s, got %s", expectedSHA256, actualSHA256)
	}

	log.Info().Str("version", version).Msg("jar checksum verified")
	return nil
}

// LoadChecksums loads checksums from the checksums.txt file
func LoadChecksums(jarsDir string) (map[string]string, error) {
	checksumsPath := filepath.Join(jarsDir, checksumsFileName)
	checksums := make(map[string]string)

	file, err := os.Open(checksumsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return checksums, nil // Return empty map if file doesn't exist
		}
		return nil, fmt.Errorf("failed to open checksums file: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue // Skip empty lines and comments
		}

		// Parse sha256sum format: <checksum>  <filename>
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue // Skip malformed lines
		}

		checksum := parts[0]
		// Filename might have spaces, so join everything after the checksum
		filename := strings.Join(parts[1:], " ")
		// Remove leading asterisk or space if present (sha256sum format)
		filename = strings.TrimPrefix(filename, "*")
		filename = strings.TrimSpace(filename)

		checksums[filename] = checksum
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read checksums file: %w", err)
	}

	return checksums, nil
}

// SaveChecksum saves or updates a checksum in the checksums.txt file
func SaveChecksum(version, sha256, jarsDir string) error {
	checksumsPath := filepath.Join(jarsDir, checksumsFileName)
	jarFileName := fmt.Sprintf("%s%s%s", jarPrefix, version, jarSuffix)

	// Load existing checksums
	checksums, err := LoadChecksums(jarsDir)
	if err != nil {
		return fmt.Errorf("failed to load existing checksums: %w", err)
	}

	// Update or add the checksum
	checksums[jarFileName] = sha256

	// Write back to file
	file, err := os.Create(checksumsPath)
	if err != nil {
		return fmt.Errorf("failed to create checksums file: %w", err)
	}
	defer file.Close()

	// Write all checksums in sha256sum format
	for filename, checksum := range checksums {
		// sha256sum format: <checksum>  <filename>
		// Use two spaces for compatibility
		_, err := fmt.Fprintf(file, "%s  %s\n", checksum, filename)
		if err != nil {
			return fmt.Errorf("failed to write checksum: %w", err)
		}
	}

	return nil
}

// computeSHA256 computes the SHA256 checksum of a file
func computeSHA256(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

