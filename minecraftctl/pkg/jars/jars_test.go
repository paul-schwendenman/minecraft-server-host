package jars

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Helper to create a test JAR file with known content
func createTestJar(t *testing.T, dir, version, content string) string {
	t.Helper()
	jarPath := filepath.Join(dir, fmt.Sprintf("%s%s%s", jarPrefix, version, jarSuffix))
	if err := os.WriteFile(jarPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test jar: %v", err)
	}
	return jarPath
}

// Helper to compute SHA256 of a string
func sha256sum(content string) string {
	hash := sha256.Sum256([]byte(content))
	return hex.EncodeToString(hash[:])
}

func TestListJars(t *testing.T) {
	t.Run("empty directory", func(t *testing.T) {
		dir := t.TempDir()
		jars, err := ListJars(dir)
		if err != nil {
			t.Fatalf("ListJars failed: %v", err)
		}
		if len(jars) != 0 {
			t.Errorf("Expected 0 jars, got %d", len(jars))
		}
	})

	t.Run("multiple jars", func(t *testing.T) {
		dir := t.TempDir()
		createTestJar(t, dir, "1.20.1", "content1")
		createTestJar(t, dir, "1.20.4", "content2")
		createTestJar(t, dir, "1.21", "content3")

		jars, err := ListJars(dir)
		if err != nil {
			t.Fatalf("ListJars failed: %v", err)
		}
		if len(jars) != 3 {
			t.Errorf("Expected 3 jars, got %d", len(jars))
		}

		// Check versions are extracted correctly
		versions := make(map[string]bool)
		for _, jar := range jars {
			versions[jar.Version] = true
		}
		for _, v := range []string{"1.20.1", "1.20.4", "1.21"} {
			if !versions[v] {
				t.Errorf("Expected version %s not found", v)
			}
		}
	})

	t.Run("ignores non-jar files", func(t *testing.T) {
		dir := t.TempDir()
		createTestJar(t, dir, "1.20.1", "content")
		// Create non-jar files
		os.WriteFile(filepath.Join(dir, "readme.txt"), []byte("readme"), 0644)
		os.WriteFile(filepath.Join(dir, "server.jar"), []byte("wrong name"), 0644)
		os.WriteFile(filepath.Join(dir, "minecraft_server_1.20.zip"), []byte("wrong ext"), 0644)

		jars, err := ListJars(dir)
		if err != nil {
			t.Fatalf("ListJars failed: %v", err)
		}
		if len(jars) != 1 {
			t.Errorf("Expected 1 jar, got %d", len(jars))
		}
	})

	t.Run("ignores directories", func(t *testing.T) {
		dir := t.TempDir()
		createTestJar(t, dir, "1.20.1", "content")
		// Create a directory with jar-like name
		os.Mkdir(filepath.Join(dir, "minecraft_server_1.21.jar"), 0755)

		jars, err := ListJars(dir)
		if err != nil {
			t.Fatalf("ListJars failed: %v", err)
		}
		if len(jars) != 1 {
			t.Errorf("Expected 1 jar, got %d", len(jars))
		}
	})

	t.Run("nonexistent directory", func(t *testing.T) {
		_, err := ListJars("/nonexistent/path")
		if err == nil {
			t.Error("Expected error for nonexistent directory")
		}
	})
}

func TestGetJarInfo(t *testing.T) {
	t.Run("existing jar", func(t *testing.T) {
		dir := t.TempDir()
		content := "test jar content"
		createTestJar(t, dir, "1.20.4", content)

		info, err := GetJarInfo("1.20.4", dir)
		if err != nil {
			t.Fatalf("GetJarInfo failed: %v", err)
		}

		if info.Version != "1.20.4" {
			t.Errorf("Version = %q, want %q", info.Version, "1.20.4")
		}
		if info.Size != int64(len(content)) {
			t.Errorf("Size = %d, want %d", info.Size, len(content))
		}
		if info.Checksum != sha256sum(content) {
			t.Errorf("Checksum mismatch")
		}
		if !strings.HasSuffix(info.Path, "minecraft_server_1.20.4.jar") {
			t.Errorf("Path = %q, expected to end with jar filename", info.Path)
		}
	})

	t.Run("nonexistent jar", func(t *testing.T) {
		dir := t.TempDir()
		_, err := GetJarInfo("1.99.99", dir)
		if err == nil {
			t.Error("Expected error for nonexistent jar")
		}
	})
}

func TestComputeSHA256(t *testing.T) {
	t.Run("known content", func(t *testing.T) {
		dir := t.TempDir()
		content := "hello world"
		path := filepath.Join(dir, "test.txt")
		os.WriteFile(path, []byte(content), 0644)

		got, err := computeSHA256(path)
		if err != nil {
			t.Fatalf("computeSHA256 failed: %v", err)
		}

		// Pre-computed SHA256 of "hello world"
		want := "b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9"
		if got != want {
			t.Errorf("computeSHA256 = %q, want %q", got, want)
		}
	})

	t.Run("empty file", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "empty.txt")
		os.WriteFile(path, []byte{}, 0644)

		got, err := computeSHA256(path)
		if err != nil {
			t.Fatalf("computeSHA256 failed: %v", err)
		}

		// SHA256 of empty string
		want := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
		if got != want {
			t.Errorf("computeSHA256 = %q, want %q", got, want)
		}
	})

	t.Run("nonexistent file", func(t *testing.T) {
		_, err := computeSHA256("/nonexistent/file")
		if err == nil {
			t.Error("Expected error for nonexistent file")
		}
	})
}

func TestLoadChecksums(t *testing.T) {
	t.Run("valid checksums file", func(t *testing.T) {
		dir := t.TempDir()
		content := `abc123  minecraft_server_1.20.1.jar
def456  minecraft_server_1.20.4.jar
`
		os.WriteFile(filepath.Join(dir, checksumsFileName), []byte(content), 0644)

		checksums, err := LoadChecksums(dir)
		if err != nil {
			t.Fatalf("LoadChecksums failed: %v", err)
		}

		if checksums["minecraft_server_1.20.1.jar"] != "abc123" {
			t.Errorf("Wrong checksum for 1.20.1")
		}
		if checksums["minecraft_server_1.20.4.jar"] != "def456" {
			t.Errorf("Wrong checksum for 1.20.4")
		}
	})

	t.Run("missing file returns empty map", func(t *testing.T) {
		dir := t.TempDir()
		checksums, err := LoadChecksums(dir)
		if err != nil {
			t.Fatalf("LoadChecksums failed: %v", err)
		}
		if len(checksums) != 0 {
			t.Errorf("Expected empty map, got %d entries", len(checksums))
		}
	})

	t.Run("skips comments and empty lines", func(t *testing.T) {
		dir := t.TempDir()
		content := `# This is a comment
abc123  minecraft_server_1.20.1.jar

# Another comment
def456  minecraft_server_1.20.4.jar
`
		os.WriteFile(filepath.Join(dir, checksumsFileName), []byte(content), 0644)

		checksums, err := LoadChecksums(dir)
		if err != nil {
			t.Fatalf("LoadChecksums failed: %v", err)
		}
		if len(checksums) != 2 {
			t.Errorf("Expected 2 checksums, got %d", len(checksums))
		}
	})

	t.Run("handles asterisk prefix", func(t *testing.T) {
		dir := t.TempDir()
		// sha256sum binary mode uses asterisk prefix
		content := `abc123 *minecraft_server_1.20.1.jar
`
		os.WriteFile(filepath.Join(dir, checksumsFileName), []byte(content), 0644)

		checksums, err := LoadChecksums(dir)
		if err != nil {
			t.Fatalf("LoadChecksums failed: %v", err)
		}
		if checksums["minecraft_server_1.20.1.jar"] != "abc123" {
			t.Errorf("Wrong checksum, asterisk not stripped properly")
		}
	})

	t.Run("skips malformed lines", func(t *testing.T) {
		dir := t.TempDir()
		content := `abc123  minecraft_server_1.20.1.jar
invalidline
def456  minecraft_server_1.20.4.jar
`
		os.WriteFile(filepath.Join(dir, checksumsFileName), []byte(content), 0644)

		checksums, err := LoadChecksums(dir)
		if err != nil {
			t.Fatalf("LoadChecksums failed: %v", err)
		}
		if len(checksums) != 2 {
			t.Errorf("Expected 2 checksums, got %d", len(checksums))
		}
	})
}

func TestSaveChecksum(t *testing.T) {
	t.Run("create new file", func(t *testing.T) {
		dir := t.TempDir()
		err := SaveChecksum("1.20.1", "abc123", dir)
		if err != nil {
			t.Fatalf("SaveChecksum failed: %v", err)
		}

		checksums, err := LoadChecksums(dir)
		if err != nil {
			t.Fatalf("LoadChecksums failed: %v", err)
		}
		if checksums["minecraft_server_1.20.1.jar"] != "abc123" {
			t.Errorf("Checksum not saved correctly")
		}
	})

	t.Run("update existing file", func(t *testing.T) {
		dir := t.TempDir()
		// Create initial file
		SaveChecksum("1.20.1", "abc123", dir)
		// Add another
		SaveChecksum("1.20.4", "def456", dir)

		checksums, err := LoadChecksums(dir)
		if err != nil {
			t.Fatalf("LoadChecksums failed: %v", err)
		}
		if len(checksums) != 2 {
			t.Errorf("Expected 2 checksums, got %d", len(checksums))
		}
		if checksums["minecraft_server_1.20.1.jar"] != "abc123" {
			t.Errorf("First checksum lost")
		}
		if checksums["minecraft_server_1.20.4.jar"] != "def456" {
			t.Errorf("Second checksum not saved")
		}
	})

	t.Run("overwrite existing checksum", func(t *testing.T) {
		dir := t.TempDir()
		SaveChecksum("1.20.1", "abc123", dir)
		SaveChecksum("1.20.1", "newchecksum", dir)

		checksums, err := LoadChecksums(dir)
		if err != nil {
			t.Fatalf("LoadChecksums failed: %v", err)
		}
		if checksums["minecraft_server_1.20.1.jar"] != "newchecksum" {
			t.Errorf("Checksum not updated")
		}
	})
}

func TestVerifyJar(t *testing.T) {
	t.Run("valid checksum", func(t *testing.T) {
		dir := t.TempDir()
		content := "test jar content"
		createTestJar(t, dir, "1.20.1", content)
		SaveChecksum("1.20.1", sha256sum(content), dir)

		err := VerifyJar("1.20.1", dir)
		if err != nil {
			t.Errorf("VerifyJar failed: %v", err)
		}
	})

	t.Run("invalid checksum", func(t *testing.T) {
		dir := t.TempDir()
		createTestJar(t, dir, "1.20.1", "test content")
		SaveChecksum("1.20.1", "wrongchecksum", dir)

		err := VerifyJar("1.20.1", dir)
		if err == nil {
			t.Error("Expected error for invalid checksum")
		}
		if !strings.Contains(err.Error(), "mismatch") {
			t.Errorf("Error should mention mismatch: %v", err)
		}
	})

	t.Run("missing jar", func(t *testing.T) {
		dir := t.TempDir()
		SaveChecksum("1.20.1", "somechecksum", dir)

		err := VerifyJar("1.20.1", dir)
		if err == nil {
			t.Error("Expected error for missing jar")
		}
	})

	t.Run("missing checksum", func(t *testing.T) {
		dir := t.TempDir()
		createTestJar(t, dir, "1.20.1", "content")
		// No checksums.txt

		err := VerifyJar("1.20.1", dir)
		if err == nil {
			t.Error("Expected error for missing checksum")
		}
	})
}

func TestDownloadJar(t *testing.T) {
	t.Run("successful download", func(t *testing.T) {
		content := "fake jar content for testing"
		checksum := sha256sum(content)

		// Create test server
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(content))
		}))
		defer server.Close()

		dir := t.TempDir()
		err := DownloadJar("1.20.1", server.URL, dir, checksum)
		if err != nil {
			t.Fatalf("DownloadJar failed: %v", err)
		}

		// Verify file exists
		jarPath := filepath.Join(dir, "minecraft_server_1.20.1.jar")
		if _, err := os.Stat(jarPath); err != nil {
			t.Errorf("Downloaded jar not found: %v", err)
		}

		// Verify content
		data, _ := os.ReadFile(jarPath)
		if string(data) != content {
			t.Errorf("Downloaded content mismatch")
		}

		// Verify checksum was saved
		checksums, _ := LoadChecksums(dir)
		if checksums["minecraft_server_1.20.1.jar"] != checksum {
			t.Errorf("Checksum not saved after download")
		}
	})

	t.Run("checksum mismatch", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("actual content"))
		}))
		defer server.Close()

		dir := t.TempDir()
		err := DownloadJar("1.20.1", server.URL, dir, "wrongchecksum")
		if err == nil {
			t.Error("Expected error for checksum mismatch")
		}
		if !strings.Contains(err.Error(), "mismatch") {
			t.Errorf("Error should mention mismatch: %v", err)
		}

		// Verify jar was NOT created
		jarPath := filepath.Join(dir, "minecraft_server_1.20.1.jar")
		if _, err := os.Stat(jarPath); !os.IsNotExist(err) {
			t.Error("Jar should not exist after checksum failure")
		}
	})

	t.Run("jar already exists", func(t *testing.T) {
		dir := t.TempDir()
		createTestJar(t, dir, "1.20.1", "existing")

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("new content"))
		}))
		defer server.Close()

		err := DownloadJar("1.20.1", server.URL, dir, "")
		if err == nil {
			t.Error("Expected error when jar already exists")
		}
		if !strings.Contains(err.Error(), "already exists") {
			t.Errorf("Error should mention already exists: %v", err)
		}
	})

	t.Run("server error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer server.Close()

		dir := t.TempDir()
		err := DownloadJar("1.20.1", server.URL, dir, "")
		if err == nil {
			t.Error("Expected error for server error")
		}
	})

	t.Run("download without checksum verification", func(t *testing.T) {
		content := "jar content"
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(content))
		}))
		defer server.Close()

		dir := t.TempDir()
		// Empty expected checksum means skip verification
		err := DownloadJar("1.20.1", server.URL, dir, "")
		if err != nil {
			t.Fatalf("DownloadJar failed: %v", err)
		}

		// Verify checksum was still saved
		checksums, _ := LoadChecksums(dir)
		if checksums["minecraft_server_1.20.1.jar"] != sha256sum(content) {
			t.Errorf("Checksum should be saved even without verification")
		}
	})
}

func TestJarInfoStruct(t *testing.T) {
	// Test that JarInfo struct fields are accessible
	info := JarInfo{
		Version:  "1.20.4",
		Path:     "/path/to/jar",
		Size:     12345,
		Checksum: "abc123",
	}

	if info.Version != "1.20.4" {
		t.Errorf("Version = %q, want %q", info.Version, "1.20.4")
	}
	if info.Size != 12345 {
		t.Errorf("Size = %d, want %d", info.Size, 12345)
	}
}
