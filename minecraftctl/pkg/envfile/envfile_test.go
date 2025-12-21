package envfile

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad(t *testing.T) {
	// Create a temp env file
	dir := t.TempDir()
	envPath := filepath.Join(dir, "test.env")

	content := `# This is a comment
RCON_PASSWORD=mysecretpassword
RCON_PORT=25575

# Another comment
EMPTY_VALUE=
QUOTED_VALUE="hello world"
SINGLE_QUOTED='single quoted'
`
	if err := os.WriteFile(envPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test env file: %v", err)
	}

	ef, err := Load(envPath)
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	tests := []struct {
		key      string
		expected string
		found    bool
	}{
		{"RCON_PASSWORD", "mysecretpassword", true},
		{"RCON_PORT", "25575", true},
		{"EMPTY_VALUE", "", true},
		{"QUOTED_VALUE", "hello world", true},
		{"SINGLE_QUOTED", "single quoted", true},
		{"MISSING_KEY", "", false},
	}

	for _, tt := range tests {
		val, ok := ef.Get(tt.key)
		if ok != tt.found {
			t.Errorf("Get(%s) found = %v, want %v", tt.key, ok, tt.found)
		}
		if val != tt.expected {
			t.Errorf("Get(%s) = %q, want %q", tt.key, val, tt.expected)
		}
	}
}

func TestLoadMissingFile(t *testing.T) {
	_, err := Load("/nonexistent/path/file.env")
	if err == nil {
		t.Error("Load() should fail for missing file")
	}
}

func TestGetInt(t *testing.T) {
	dir := t.TempDir()
	envPath := filepath.Join(dir, "test.env")

	content := `PORT=25575
INVALID=notanumber
`
	if err := os.WriteFile(envPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test env file: %v", err)
	}

	ef, err := Load(envPath)
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	// Test valid int
	port, err := ef.GetInt("PORT")
	if err != nil {
		t.Errorf("GetInt(PORT) failed: %v", err)
	}
	if port != 25575 {
		t.Errorf("GetInt(PORT) = %d, want 25575", port)
	}

	// Test invalid int
	_, err = ef.GetInt("INVALID")
	if err == nil {
		t.Error("GetInt(INVALID) should fail for non-integer value")
	}

	// Test missing key
	_, err = ef.GetInt("MISSING")
	if err == nil {
		t.Error("GetInt(MISSING) should fail for missing key")
	}
}

func TestExport(t *testing.T) {
	dir := t.TempDir()
	envPath := filepath.Join(dir, "test.env")

	content := `TEST_EXPORT_VAR=testvalue
`
	if err := os.WriteFile(envPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test env file: %v", err)
	}

	// Clear any existing value
	os.Unsetenv("TEST_EXPORT_VAR")

	ef, err := Load(envPath)
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if err := ef.Export(); err != nil {
		t.Fatalf("Export() failed: %v", err)
	}

	if val := os.Getenv("TEST_EXPORT_VAR"); val != "testvalue" {
		t.Errorf("After Export(), TEST_EXPORT_VAR = %q, want %q", val, "testvalue")
	}

	// Cleanup
	os.Unsetenv("TEST_EXPORT_VAR")
}

func TestExportIfNotSet(t *testing.T) {
	dir := t.TempDir()
	envPath := filepath.Join(dir, "test.env")

	content := `TEST_EXPORT_IF_NOT_SET=fromfile
`
	if err := os.WriteFile(envPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test env file: %v", err)
	}

	// Set an existing value
	os.Setenv("TEST_EXPORT_IF_NOT_SET", "existing")
	defer os.Unsetenv("TEST_EXPORT_IF_NOT_SET")

	ef, err := Load(envPath)
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if err := ef.ExportIfNotSet(); err != nil {
		t.Fatalf("ExportIfNotSet() failed: %v", err)
	}

	// Should keep existing value
	if val := os.Getenv("TEST_EXPORT_IF_NOT_SET"); val != "existing" {
		t.Errorf("After ExportIfNotSet(), TEST_EXPORT_IF_NOT_SET = %q, want %q", val, "existing")
	}
}

func TestLoadWithFallback(t *testing.T) {
	dir := t.TempDir()
	envPath := filepath.Join(dir, "test.env")

	content := `FALLBACK_VAR=found
`
	if err := os.WriteFile(envPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test env file: %v", err)
	}

	// First path doesn't exist, second does
	ef, err := LoadWithFallback("/nonexistent/path", envPath)
	if err != nil {
		t.Fatalf("LoadWithFallback() failed: %v", err)
	}

	val, ok := ef.Get("FALLBACK_VAR")
	if !ok || val != "found" {
		t.Errorf("LoadWithFallback() didn't load correct file")
	}

	// Test all paths missing
	_, err = LoadWithFallback("/nonexistent/path1", "/nonexistent/path2")
	if err == nil {
		t.Error("LoadWithFallback() should fail when all paths are missing")
	}
}

func TestPath(t *testing.T) {
	dir := t.TempDir()
	envPath := filepath.Join(dir, "test.env")

	if err := os.WriteFile(envPath, []byte("KEY=value\n"), 0644); err != nil {
		t.Fatalf("failed to write test env file: %v", err)
	}

	ef, err := Load(envPath)
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if ef.Path() != envPath {
		t.Errorf("Path() = %q, want %q", ef.Path(), envPath)
	}
}

func TestKeys(t *testing.T) {
	dir := t.TempDir()
	envPath := filepath.Join(dir, "test.env")

	content := `KEY1=value1
KEY2=value2
KEY3=value3
`
	if err := os.WriteFile(envPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test env file: %v", err)
	}

	ef, err := Load(envPath)
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	keys := ef.Keys()
	if len(keys) != 3 {
		t.Errorf("Keys() returned %d keys, want 3", len(keys))
	}

	keySet := make(map[string]bool)
	for _, k := range keys {
		keySet[k] = true
	}

	for _, expected := range []string{"KEY1", "KEY2", "KEY3"} {
		if !keySet[expected] {
			t.Errorf("Keys() missing key: %s", expected)
		}
	}
}

func TestLen(t *testing.T) {
	dir := t.TempDir()
	envPath := filepath.Join(dir, "test.env")

	content := `KEY1=value1
KEY2=value2
`
	if err := os.WriteFile(envPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test env file: %v", err)
	}

	ef, err := Load(envPath)
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if ef.Len() != 2 {
		t.Errorf("Len() = %d, want 2", ef.Len())
	}
}

func TestMustGet(t *testing.T) {
	dir := t.TempDir()
	envPath := filepath.Join(dir, "test.env")

	content := `EXISTING_KEY=value
`
	if err := os.WriteFile(envPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test env file: %v", err)
	}

	ef, err := Load(envPath)
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	// Test existing key
	val := ef.MustGet("EXISTING_KEY")
	if val != "value" {
		t.Errorf("MustGet(EXISTING_KEY) = %q, want %q", val, "value")
	}

	// Test missing key panics
	defer func() {
		if r := recover(); r == nil {
			t.Error("MustGet(MISSING_KEY) should panic")
		}
	}()
	ef.MustGet("MISSING_KEY")
}
