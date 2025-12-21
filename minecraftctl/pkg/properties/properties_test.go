package properties

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoad(t *testing.T) {
	dir := t.TempDir()
	propsPath := filepath.Join(dir, "server.properties")

	content := `#Minecraft server properties
#Generated on test

enable-rcon=true
rcon.port=25575
rcon.password=mysecretpassword
motd=Welcome to the server
level-name=world
max-players=20
`
	if err := os.WriteFile(propsPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test properties file: %v", err)
	}

	p, err := Load(propsPath)
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	tests := []struct {
		key      string
		expected string
		found    bool
	}{
		{"enable-rcon", "true", true},
		{"rcon.port", "25575", true},
		{"rcon.password", "mysecretpassword", true},
		{"motd", "Welcome to the server", true},
		{"level-name", "world", true},
		{"max-players", "20", true},
		{"missing-key", "", false},
	}

	for _, tt := range tests {
		val, ok := p.Get(tt.key)
		if ok != tt.found {
			t.Errorf("Get(%s) found = %v, want %v", tt.key, ok, tt.found)
		}
		if val != tt.expected {
			t.Errorf("Get(%s) = %q, want %q", tt.key, val, tt.expected)
		}
	}
}

func TestLoadMissingFile(t *testing.T) {
	_, err := Load("/nonexistent/path/server.properties")
	if err == nil {
		t.Error("Load() should fail for missing file")
	}
}

func TestGetInt(t *testing.T) {
	dir := t.TempDir()
	propsPath := filepath.Join(dir, "server.properties")

	content := `max-players=20
invalid=notanumber
`
	if err := os.WriteFile(propsPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test properties file: %v", err)
	}

	p, err := Load(propsPath)
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	// Test valid int
	val, err := p.GetInt("max-players")
	if err != nil {
		t.Errorf("GetInt(max-players) failed: %v", err)
	}
	if val != 20 {
		t.Errorf("GetInt(max-players) = %d, want 20", val)
	}

	// Test invalid int
	_, err = p.GetInt("invalid")
	if err == nil {
		t.Error("GetInt(invalid) should fail for non-integer value")
	}

	// Test missing key
	_, err = p.GetInt("missing")
	if err == nil {
		t.Error("GetInt(missing) should fail for missing key")
	}
}

func TestGetBool(t *testing.T) {
	dir := t.TempDir()
	propsPath := filepath.Join(dir, "server.properties")

	content := `enable-rcon=true
pvp=false
invalid=notabool
`
	if err := os.WriteFile(propsPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test properties file: %v", err)
	}

	p, err := Load(propsPath)
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	// Test true
	val, err := p.GetBool("enable-rcon")
	if err != nil {
		t.Errorf("GetBool(enable-rcon) failed: %v", err)
	}
	if val != true {
		t.Errorf("GetBool(enable-rcon) = %v, want true", val)
	}

	// Test false
	val, err = p.GetBool("pvp")
	if err != nil {
		t.Errorf("GetBool(pvp) failed: %v", err)
	}
	if val != false {
		t.Errorf("GetBool(pvp) = %v, want false", val)
	}

	// Test invalid bool
	_, err = p.GetBool("invalid")
	if err == nil {
		t.Error("GetBool(invalid) should fail for non-boolean value")
	}
}

func TestSet(t *testing.T) {
	p := New()

	// Set new key
	p.Set("key1", "value1")
	val, ok := p.Get("key1")
	if !ok || val != "value1" {
		t.Errorf("Set/Get failed: got %q, want %q", val, "value1")
	}

	// Update existing key
	p.Set("key1", "newvalue")
	val, ok = p.Get("key1")
	if !ok || val != "newvalue" {
		t.Errorf("Set update failed: got %q, want %q", val, "newvalue")
	}
}

func TestSetInt(t *testing.T) {
	p := New()
	p.SetInt("max-players", 30)

	val, err := p.GetInt("max-players")
	if err != nil {
		t.Errorf("GetInt failed: %v", err)
	}
	if val != 30 {
		t.Errorf("SetInt/GetInt failed: got %d, want 30", val)
	}
}

func TestSetBool(t *testing.T) {
	p := New()
	p.SetBool("enable-rcon", true)

	val, err := p.GetBool("enable-rcon")
	if err != nil {
		t.Errorf("GetBool failed: %v", err)
	}
	if val != true {
		t.Errorf("SetBool/GetBool failed: got %v, want true", val)
	}
}

func TestDelete(t *testing.T) {
	p := New()
	p.Set("key1", "value1")
	p.Set("key2", "value2")

	p.Delete("key1")

	if p.Has("key1") {
		t.Error("Delete failed: key1 still exists")
	}
	if !p.Has("key2") {
		t.Error("Delete removed wrong key: key2 missing")
	}
}

func TestHas(t *testing.T) {
	p := New()
	p.Set("existing", "value")

	if !p.Has("existing") {
		t.Error("Has(existing) = false, want true")
	}
	if p.Has("missing") {
		t.Error("Has(missing) = true, want false")
	}
}

func TestKeys(t *testing.T) {
	p := New()
	p.Set("key1", "value1")
	p.Set("key2", "value2")
	p.Set("key3", "value3")

	keys := p.Keys()
	if len(keys) != 3 {
		t.Errorf("Keys() returned %d keys, want 3", len(keys))
	}

	// Keys should be in insertion order
	expected := []string{"key1", "key2", "key3"}
	for i, k := range keys {
		if k != expected[i] {
			t.Errorf("Keys()[%d] = %s, want %s", i, k, expected[i])
		}
	}
}

func TestSaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	propsPath := filepath.Join(dir, "server.properties")

	p := New()
	p.SetPath(propsPath)
	p.Set("enable-rcon", "true")
	p.SetInt("rcon.port", 25575)
	p.Set("rcon.password", "secret")
	p.Set("motd", "Test Server")

	if err := p.Save(); err != nil {
		t.Fatalf("Save() failed: %v", err)
	}

	// Load it back
	p2, err := Load(propsPath)
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	// Verify all values
	tests := []struct {
		key      string
		expected string
	}{
		{"enable-rcon", "true"},
		{"rcon.port", "25575"},
		{"rcon.password", "secret"},
		{"motd", "Test Server"},
	}

	for _, tt := range tests {
		val, ok := p2.Get(tt.key)
		if !ok {
			t.Errorf("After reload, key %s not found", tt.key)
		}
		if val != tt.expected {
			t.Errorf("After reload, Get(%s) = %q, want %q", tt.key, val, tt.expected)
		}
	}
}

func TestPreservesCommentsAndOrder(t *testing.T) {
	dir := t.TempDir()
	propsPath := filepath.Join(dir, "server.properties")

	content := `#Minecraft server properties
#Generated on test

enable-rcon=true
#RCON port setting
rcon.port=25575
rcon.password=secret
`
	if err := os.WriteFile(propsPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test properties file: %v", err)
	}

	p, err := Load(propsPath)
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	// Modify a value
	p.Set("rcon.port", "25576")

	// Save to new file
	newPath := filepath.Join(dir, "server2.properties")
	if err := p.SaveTo(newPath); err != nil {
		t.Fatalf("SaveTo() failed: %v", err)
	}

	// Read the raw content
	saved, err := os.ReadFile(newPath)
	if err != nil {
		t.Fatalf("failed to read saved file: %v", err)
	}

	savedStr := string(saved)

	// Verify comments are preserved
	if !strings.Contains(savedStr, "#Minecraft server properties") {
		t.Error("Header comment not preserved")
	}
	if !strings.Contains(savedStr, "#RCON port setting") {
		t.Error("Inline comment not preserved")
	}

	// Verify modified value
	if !strings.Contains(savedStr, "rcon.port=25576") {
		t.Error("Modified value not saved correctly")
	}
}

func TestString(t *testing.T) {
	p := New()
	p.Set("key1", "value1")
	p.Set("key2", "value2")

	str := p.String()

	if !strings.Contains(str, "key1=value1") {
		t.Error("String() missing key1=value1")
	}
	if !strings.Contains(str, "key2=value2") {
		t.Error("String() missing key2=value2")
	}
}

func TestPath(t *testing.T) {
	p := New()
	if p.Path() != "" {
		t.Errorf("New properties should have empty path, got %q", p.Path())
	}

	p.SetPath("/some/path")
	if p.Path() != "/some/path" {
		t.Errorf("SetPath/Path failed: got %q", p.Path())
	}
}

func TestLen(t *testing.T) {
	p := New()
	if p.Len() != 0 {
		t.Errorf("New properties Len() = %d, want 0", p.Len())
	}

	p.Set("key1", "value1")
	p.Set("key2", "value2")

	if p.Len() != 2 {
		t.Errorf("Len() = %d, want 2", p.Len())
	}

	p.Delete("key1")
	if p.Len() != 1 {
		t.Errorf("After delete, Len() = %d, want 1", p.Len())
	}
}

func TestSaveNoPath(t *testing.T) {
	p := New()
	p.Set("key", "value")

	err := p.Save()
	if err == nil {
		t.Error("Save() should fail when no path is set")
	}
}

func TestBlankLines(t *testing.T) {
	dir := t.TempDir()
	propsPath := filepath.Join(dir, "server.properties")

	content := `key1=value1

key2=value2

key3=value3
`
	if err := os.WriteFile(propsPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test properties file: %v", err)
	}

	p, err := Load(propsPath)
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	// Verify blank lines are preserved
	str := p.String()
	lines := strings.Split(str, "\n")
	blankCount := 0
	for _, line := range lines {
		if line == "" {
			blankCount++
		}
	}
	// Should have at least 2 blank lines (the ones between entries)
	if blankCount < 2 {
		t.Errorf("Blank lines not preserved, got %d", blankCount)
	}
}
