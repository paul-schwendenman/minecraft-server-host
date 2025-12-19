package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/paul/minecraftctl/pkg/config"
	"github.com/paul/minecraftctl/pkg/jars"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// setupTestConfig resets viper and initializes config with test directories
func setupTestConfig(t *testing.T, jarsDir string) {
	t.Helper()
	viper.Reset()
	viper.Set("jars_dir", jarsDir)
	// Initialize config so Get() reads from Viper
	if err := config.Init(""); err != nil {
		t.Fatalf("config.Init() failed: %v", err)
	}
}

// createTestJarFile creates a test JAR file with known content
func createTestJarFile(t *testing.T, dir, version, content string) string {
	t.Helper()
	jarPath := filepath.Join(dir, fmt.Sprintf("minecraft_server_%s.jar", version))
	if err := os.WriteFile(jarPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test jar: %v", err)
	}
	return jarPath
}

// sha256sumContent computes SHA256 of a string
func sha256sumContent(content string) string {
	hash := sha256.Sum256([]byte(content))
	return hex.EncodeToString(hash[:])
}

// executeCommand runs a command and captures output
func executeCommand(cmd *cobra.Command, args ...string) (string, error) {
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	// Reset the command's Run functions to avoid nil pointer issues
	// and create a fresh command for execution
	cmd.SetArgs(args)

	// For commands with RunE, we need to execute through the command tree
	// We'll call the command's RunE directly for isolated testing
	if cmd.RunE != nil {
		err := cmd.RunE(cmd, args)
		return buf.String(), err
	}
	return buf.String(), nil
}

// captureStdout captures stdout during a function execution
func captureStdout(t *testing.T, fn func() error) (string, error) {
	t.Helper()
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := fn()

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	buf.ReadFrom(r)
	return buf.String(), err
}

func TestJarListCmdExecution(t *testing.T) {
	t.Run("empty directory shows no jars message", func(t *testing.T) {
		dir := t.TempDir()
		setupTestConfig(t, dir)

		output, err := captureStdout(t, func() error {
			return jarListCmd.RunE(jarListCmd, []string{})
		})

		if err != nil {
			t.Fatalf("jarListCmd.RunE() failed: %v", err)
		}

		if !strings.Contains(output, "No JARs found") {
			t.Errorf("Expected 'No JARs found' message, got: %s", output)
		}
	})

	t.Run("lists jars with versions", func(t *testing.T) {
		dir := t.TempDir()
		setupTestConfig(t, dir)

		// Create test jars
		createTestJarFile(t, dir, "1.20.1", "content1")
		createTestJarFile(t, dir, "1.20.4", "content2")
		createTestJarFile(t, dir, "1.21", "content3")

		output, err := captureStdout(t, func() error {
			return jarListCmd.RunE(jarListCmd, []string{})
		})

		if err != nil {
			t.Fatalf("jarListCmd.RunE() failed: %v", err)
		}

		// Check header
		if !strings.Contains(output, "VERSION") {
			t.Errorf("Expected VERSION header, got: %s", output)
		}

		// Check versions are listed
		for _, version := range []string{"1.20.1", "1.20.4", "1.21"} {
			if !strings.Contains(output, version) {
				t.Errorf("Expected version %s in output, got: %s", version, output)
			}
		}
	})

	t.Run("shows checksum prefix", func(t *testing.T) {
		dir := t.TempDir()
		setupTestConfig(t, dir)

		createTestJarFile(t, dir, "1.20.1", "test content")

		output, err := captureStdout(t, func() error {
			return jarListCmd.RunE(jarListCmd, []string{})
		})

		if err != nil {
			t.Fatalf("jarListCmd.RunE() failed: %v", err)
		}

		// Check that CHECKSUM header is present
		if !strings.Contains(output, "CHECKSUM") {
			t.Errorf("Expected CHECKSUM header, got: %s", output)
		}

		// Check checksum is truncated (ends with ...)
		if !strings.Contains(output, "...") {
			t.Errorf("Expected truncated checksum with '...', got: %s", output)
		}
	})
}

func TestJarInfoCmdExecution(t *testing.T) {
	t.Run("shows jar info", func(t *testing.T) {
		dir := t.TempDir()
		setupTestConfig(t, dir)

		content := "test jar content for info"
		createTestJarFile(t, dir, "1.20.4", content)

		output, err := captureStdout(t, func() error {
			return jarInfoCmd.RunE(jarInfoCmd, []string{"1.20.4"})
		})

		if err != nil {
			t.Fatalf("jarInfoCmd.RunE() failed: %v", err)
		}

		// Check output contains expected fields
		expectedFields := []string{
			"Version: 1.20.4",
			"Path:",
			"Size:",
			"Checksum:",
			"Installed:",
		}
		for _, field := range expectedFields {
			if !strings.Contains(output, field) {
				t.Errorf("Expected %q in output, got: %s", field, output)
			}
		}

		// Verify checksum is correct
		expectedChecksum := sha256sumContent(content)
		if !strings.Contains(output, expectedChecksum) {
			t.Errorf("Expected checksum %s in output", expectedChecksum)
		}
	})

	t.Run("shows correct size", func(t *testing.T) {
		dir := t.TempDir()
		setupTestConfig(t, dir)

		content := "a"
		createTestJarFile(t, dir, "1.20.4", content)

		output, err := captureStdout(t, func() error {
			return jarInfoCmd.RunE(jarInfoCmd, []string{"1.20.4"})
		})

		if err != nil {
			t.Fatalf("jarInfoCmd.RunE() failed: %v", err)
		}

		// Size should be 1 B
		if !strings.Contains(output, "1 B") {
			t.Errorf("Expected size '1 B' in output, got: %s", output)
		}
	})

	t.Run("error for nonexistent jar", func(t *testing.T) {
		dir := t.TempDir()
		setupTestConfig(t, dir)

		err := jarInfoCmd.RunE(jarInfoCmd, []string{"1.99.99"})
		if err == nil {
			t.Error("Expected error for nonexistent jar")
		}
		if !strings.Contains(err.Error(), "not found") {
			t.Errorf("Error should mention 'not found': %v", err)
		}
	})
}

func TestJarVerifyCmdExecution(t *testing.T) {
	t.Run("verifies valid checksum", func(t *testing.T) {
		dir := t.TempDir()
		setupTestConfig(t, dir)

		content := "test jar content for verification"
		createTestJarFile(t, dir, "1.20.1", content)

		// Save the correct checksum
		checksum := sha256sumContent(content)
		jars.SaveChecksum("1.20.1", checksum, dir)

		output, err := captureStdout(t, func() error {
			return jarVerifyCmd.RunE(jarVerifyCmd, []string{"1.20.1"})
		})

		if err != nil {
			t.Fatalf("jarVerifyCmd.RunE() failed: %v", err)
		}

		if !strings.Contains(output, "verified") {
			t.Errorf("Expected 'verified' in output, got: %s", output)
		}
	})

	t.Run("fails with invalid checksum", func(t *testing.T) {
		dir := t.TempDir()
		setupTestConfig(t, dir)

		createTestJarFile(t, dir, "1.20.1", "actual content")

		// Save an incorrect checksum
		jars.SaveChecksum("1.20.1", "wrongchecksum", dir)

		err := jarVerifyCmd.RunE(jarVerifyCmd, []string{"1.20.1"})
		if err == nil {
			t.Error("Expected error for checksum mismatch")
		}
		if !strings.Contains(err.Error(), "mismatch") {
			t.Errorf("Error should mention mismatch: %v", err)
		}
	})

	t.Run("fails when jar not found", func(t *testing.T) {
		dir := t.TempDir()
		setupTestConfig(t, dir)

		err := jarVerifyCmd.RunE(jarVerifyCmd, []string{"1.99.99"})
		if err == nil {
			t.Error("Expected error when jar not found")
		}
		if !strings.Contains(err.Error(), "not found") {
			t.Errorf("Error should mention 'not found': %v", err)
		}
	})

	t.Run("fails when checksum not in file", func(t *testing.T) {
		dir := t.TempDir()
		setupTestConfig(t, dir)

		// Create jar but no checksums.txt
		createTestJarFile(t, dir, "1.20.1", "content")

		err := jarVerifyCmd.RunE(jarVerifyCmd, []string{"1.20.1"})
		if err == nil {
			t.Error("Expected error when checksum not found")
		}
		if !strings.Contains(err.Error(), "not found") {
			t.Errorf("Error should mention 'not found': %v", err)
		}
	})
}

func TestJarCmdStructure(t *testing.T) {
	t.Run("jar cmd has correct subcommands", func(t *testing.T) {
		subcommands := []string{"list", "download", "verify", "info"}
		for _, name := range subcommands {
			found := false
			for _, cmd := range jarCmd.Commands() {
				if cmd.Name() == name {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("jarCmd missing subcommand %q", name)
			}
		}
	})

	t.Run("jar download cmd has required flags", func(t *testing.T) {
		urlFlag := jarDownloadCmd.Flags().Lookup("url")
		if urlFlag == nil {
			t.Error("jarDownloadCmd should have --url flag")
		}

		sha256Flag := jarDownloadCmd.Flags().Lookup("sha256")
		if sha256Flag == nil {
			t.Error("jarDownloadCmd should have --sha256 flag")
		}
	})

	t.Run("jar list cmd uses", func(t *testing.T) {
		if jarListCmd.Use != "list" {
			t.Errorf("jarListCmd.Use = %q, want 'list'", jarListCmd.Use)
		}
		if jarListCmd.Short == "" {
			t.Error("jarListCmd should have Short description")
		}
	})

	t.Run("jar info cmd uses", func(t *testing.T) {
		if jarInfoCmd.Use != "info <version>" {
			t.Errorf("jarInfoCmd.Use = %q, want 'info <version>'", jarInfoCmd.Use)
		}
	})

	t.Run("jar verify cmd uses", func(t *testing.T) {
		if jarVerifyCmd.Use != "verify <version>" {
			t.Errorf("jarVerifyCmd.Use = %q, want 'verify <version>'", jarVerifyCmd.Use)
		}
	})

	t.Run("jar download cmd uses", func(t *testing.T) {
		if jarDownloadCmd.Use != "download <version>" {
			t.Errorf("jarDownloadCmd.Use = %q, want 'download <version>'", jarDownloadCmd.Use)
		}
	})

	t.Run("jar cmd has description", func(t *testing.T) {
		if jarCmd.Short == "" {
			t.Error("jarCmd should have Short description")
		}
		if jarCmd.Long == "" {
			t.Error("jarCmd should have Long description")
		}
	})
}

func TestFormatSize(t *testing.T) {
	tests := []struct {
		bytes int64
		want  string
	}{
		{0, "0 B"},
		{512, "512 B"},
		{1023, "1023 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1048576, "1.0 MB"},
		{1572864, "1.5 MB"},
		{1073741824, "1.0 GB"},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%d bytes", tt.bytes), func(t *testing.T) {
			got := formatSize(tt.bytes)
			if got != tt.want {
				t.Errorf("formatSize(%d) = %q, want %q", tt.bytes, got, tt.want)
			}
		})
	}
}
